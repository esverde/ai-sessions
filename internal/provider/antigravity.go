package provider

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/esverde/ais/internal/session"
)

// antigravityAppDir 是 CLI(`agy`)自己的应用数据目录。
//
// `~/.gemini` 下还有 antigravity/ 与 antigravity-ide/ 两份 IDE 的会话,但
// `agy --conversation <id>` 只认自己这一份:拿 IDE 的会话 ID 去恢复,CLI 会直接
// 报 "trajectory not found"。列出恢复不了的条目只会误导人,所以这里只扫 CLI。
const antigravityAppDir = "antigravity-cli"

// antigravityScanBytes 是回退扫描单个文件时最多读取的字节数。会话正文越滚越大,
// 而我们要找的工作目录写在元数据里,不值得为它把几十兆的文件整个读进内存。
const antigravityScanBytes = 8 << 20

type Antigravity struct{}

func (Antigravity) Name() string { return session.ProviderAntigravity }

func (Antigravity) Discover(options session.ScanOptions) ([]session.Session, error) {
	root, err := session.AntigravityRoot()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(root, antigravityAppDir)
	conversations := filepath.Join(appDir, "conversations")
	entries, err := os.ReadDir(conversations)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	history := readAntigravityHistory(filepath.Join(appDir, "history.jsonl"))
	result := make([]session.Session, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// 每个会话是一个 SQLite 库,同名的 -wal / -shm 是它的附属文件。
		// filepath.Ext 把 `x.db-wal` 的扩展名算作 `.db-wal`,所以这一条就能筛干净。
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".db") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		item, ok := buildAntigravitySession(conversations, id, history[id], options)
		if ok {
			result = append(result, item)
		}
	}
	return result, nil
}

// antigravityHistory 是 history.jsonl 为某个会话攒出来的信息。
type antigravityHistory struct {
	Workspace   string
	FirstPrompt string
	LastPrompt  string
	UpdatedAt   time.Time
}

func buildAntigravitySession(dir, id string, entry antigravityHistory, options session.ScanOptions) (session.Session, bool) {
	if id == "" {
		return session.Session{}, false
	}
	dbPath := filepath.Join(dir, id+".db")
	walPath := dbPath + "-wal"

	item := session.Session{
		Provider:   session.ProviderAntigravity,
		ID:         id,
		SourcePath: dbPath,
		UpdatedAt:  entry.UpdatedAt,
	}
	// WAL 尚未 checkpoint 时主库文件的 mtime 是旧的,取两者较晚的那个才是真实活跃时间。
	for _, path := range []string{dbPath, walPath} {
		if info, err := os.Stat(path); err == nil {
			item.UpdatedAt = MaxTime(item.UpdatedAt, info.ModTime())
		}
	}

	cwd := entry.Workspace
	if cwd == "" {
		cwd = antigravityWorkspaceFromStore(walPath, dbPath)
	}
	if cwd == "" {
		// 没有工作目录就无从判断归属,也没法在正确的位置恢复 —— 与 Codex 一致地丢弃。
		return session.Session{}, false
	}
	item.Cwd = session.NormalizePath(cwd)
	if !options.ScopeAll && !session.IsWithin(options.ScopeRoot, item.Cwd) {
		return session.Session{}, false
	}

	item.Title = Truncate(FirstNonEmpty(entry.FirstPrompt, id), options.PreviewLength)
	item.LastUser = Truncate(entry.LastPrompt, options.PreviewLength)
	return item, true
}

// readAntigravityHistory 读取 CLI 的输入历史。
//
// 这是唯一一处以纯文本记录"某个会话属于哪个工作目录"的地方,每行形如
// {"display":"…","timestamp":1788170347839,"workspace":"D:\\…","conversationId":"…"}。
// 只有交互式输入会落到这里,`agy -p` 的一次性提问不会 —— 那些会话由
// antigravityWorkspaceFromStore 兜底。
func readAntigravityHistory(path string) map[string]antigravityHistory {
	result := make(map[string]antigravityHistory)
	_ = session.ReadJSONLLines(path, func(record map[string]any) error {
		id := FirstNonEmpty(StringValue(record, "conversationId"), StringValue(record, "conversation_id"))
		if id == "" {
			return nil
		}
		entry := result[id]
		if workspace := StringValue(record, "workspace"); workspace != "" {
			entry.Workspace = workspace
		}
		if stamp, ok := antigravityTime(record["timestamp"]); ok {
			entry.UpdatedAt = MaxTime(entry.UpdatedAt, stamp)
		}
		// 斜杠命令(/mcp、/settings)是操作而不是提问,拿它当标题或预览没有信息量,
		// 但它同样带着 workspace 与时间戳,上面两项照收。
		if StringValue(record, "type") == "slash_command" {
			return nil
		}
		if display := cleanText(StringValue(record, "display")); display != "" {
			if entry.FirstPrompt == "" {
				entry.FirstPrompt = display
			}
			entry.LastPrompt = display
		}
		result[id] = entry
		return nil
	})
	return result
}

// antigravityTime 解析 history.jsonl 里的毫秒时间戳。
func antigravityTime(value any) (time.Time, bool) {
	millis, ok := value.(float64)
	if !ok || millis <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(millis)), true
}

// antigravityWorkspaceFromStore 从会话自己的 SQLite 文件里捞工作目录。
//
// 会话正文存在 SQLite 里,工作目录写在 trajectory_metadata_blob 这个 protobuf
// blob 中,形如 `file:///D:/Documents/...`。为一个会话列表引入 SQLite 驱动不划算
// —— 纯 Go 的实现都是几十兆生成代码,而我们只要那一个字符串。protobuf 的 string
// 字段是"长度前缀 + 原始字节",blob 又是原样落在数据页里的,所以直接在文件字节里
// 找 `file://`、再用紧邻的长度字节校验边界就够了。WAL 排在主库前面 —— 新会话的
// 元数据可能还只在尚未 checkpoint 的 WAL 里。
func antigravityWorkspaceFromStore(paths ...string) string {
	counts := make(map[string]int)
	order := make([]string, 0, 8)
	for _, path := range paths {
		data, err := readHead(path, antigravityScanBytes)
		if err != nil {
			continue
		}
		for _, candidate := range antigravityWorkspaces(data) {
			if counts[candidate] == 0 {
				order = append(order, candidate)
			}
			counts[candidate]++
		}
	}
	return pickAntigravityWorkspace(order, counts)
}

// pickAntigravityWorkspace 从候选里挑出真正的工作目录。
//
// 会话正文里还夹着别的 file:// URI(工具说明中的示例路径、打开过的文件),光靠
// 长度前缀分辨不出来。真正的工作目录有两个特征:它在磁盘上是一个存在的目录,
// 而且会被反复写进元数据。先按"是不是存在的目录"过滤,再取出现次数最多的那个;
// 一个都不存在就宁可放弃 —— 与其给出错误的恢复目录,不如让这条会话不出现。
func pickAntigravityWorkspace(order []string, counts map[string]int) string {
	best := ""
	bestCount := 0
	for _, candidate := range order {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		if counts[candidate] > bestCount {
			best, bestCount = candidate, counts[candidate]
		}
	}
	return best
}

// antigravityWorkspaces 按出现顺序返回文件字节里所有带合法长度前缀的 file:// 路径。
func antigravityWorkspaces(data []byte) []string {
	marker := []byte("file://")
	result := make([]string, 0, 8)
	for index := 0; index < len(data); {
		offset := bytes.Index(data[index:], marker)
		if offset < 0 {
			break
		}
		start := index + offset
		index = start + len(marker)
		if start == 0 {
			continue
		}
		// protobuf 的长度前缀是 varint;单字节形式(最高位为 0)最长表示 127 字节,
		// 足够覆盖一个工作目录 URI,更长的编码不必处理。
		length := int(data[start-1])
		if length >= 0x80 || length <= len(marker) || start+length > len(data) {
			continue
		}
		if workspace := decodeFileURI(string(data[start : start+length])); workspace != "" {
			result = append(result, workspace)
		}
	}
	return result
}

func decodeFileURI(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "file" || parsed.Path == "" {
		return ""
	}
	path := parsed.Path
	// file:///D:/… 解析出来是 /D:/…,盘符前那个斜杠得去掉,否则不是合法的 Windows 路径。
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

func readHead(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// 只读文件,Close 的错误没有可采取的行动,显式丢弃以示有意为之。
	defer func() { _ = file.Close() }()
	return io.ReadAll(io.LimitReader(file, int64(maxBytes)))
}
