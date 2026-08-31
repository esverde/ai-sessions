package provider

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/esverde/ais/internal/session"
)

// fileURI 把本地目录编成 Antigravity 元数据里那种 file:// URI,
// 顺带处理平台差异(Windows 的盘符前多一个斜杠)与百分号转义。
func fileURI(dir string) string {
	uri := url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(dir), "/")}
	return uri.String()
}

// protoString 模拟 protobuf 的 string 字段编码:单字节长度前缀 + 原始字节。
func protoString(value string) []byte {
	return append([]byte{byte(len(value))}, value...)
}

// 有 history.jsonl 记录时,工作目录、标题与预览都应当来自它。
func TestAntigravityDiscoverUsesHistory(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	appDir := filepath.Join(root, "antigravity-cli")
	conversations := filepath.Join(appDir, "conversations")
	if err := os.MkdirAll(conversations, 0o755); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(conversations, "agy-1.db"), []byte("not a real sqlite file"), 0o600); err != nil {
		t.Fatalf("write conversation: %v", err)
	}
	history := "{\"display\":\"/mcp\",\"type\":\"slash_command\",\"timestamp\":1788170122347," +
		"\"workspace\":" + quoteJSON(workspace) + ",\"conversationId\":\"agy-1\"}\n" +
		"{\"display\":\"first prompt\",\"timestamp\":1788170347839," +
		"\"workspace\":" + quoteJSON(workspace) + ",\"conversationId\":\"agy-1\"}\n" +
		"{\"display\":\"last prompt\",\"timestamp\":1788170895899," +
		"\"workspace\":" + quoteJSON(workspace) + ",\"conversationId\":\"agy-1\"}\n"
	if err := os.WriteFile(filepath.Join(appDir, "history.jsonl"), []byte(history), 0o600); err != nil {
		t.Fatalf("write history: %v", err)
	}
	t.Setenv("ANTIGRAVITY_HOME", root)

	items, err := Antigravity{}.Discover(session.ScanOptions{ScopeRoot: workspace, PreviewLength: 160})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Discover() = %#v, want one session", items)
	}
	item := items[0]
	if item.ID != "agy-1" || item.Title != "first prompt" || item.LastUser != "last prompt" {
		t.Fatalf("discovered session = %#v", item)
	}
	if item.Cwd != session.NormalizePath(workspace) {
		t.Fatalf("Cwd = %q, want %q", item.Cwd, session.NormalizePath(workspace))
	}
	// 斜杠命令是操作而不是提问,不该被当成标题。
	if item.Title == "/mcp" {
		t.Fatalf("斜杠命令不应成为标题, got %#v", item)
	}
}

// `agy -p` 的一次性提问不写 history.jsonl,只能从会话库的字节里捞工作目录。
func TestAntigravityDiscoverFallsBackToStoreBytes(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	conversations := filepath.Join(root, "antigravity-cli", "conversations")
	if err := os.MkdirAll(conversations, 0o755); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}
	blob := append([]byte("SQLite format 3\x00padding"), protoString(fileURI(workspace))...)
	if err := os.WriteFile(filepath.Join(conversations, "agy-2.db"), blob, 0o600); err != nil {
		t.Fatalf("write conversation: %v", err)
	}
	t.Setenv("ANTIGRAVITY_HOME", root)

	items, err := Antigravity{}.Discover(session.ScanOptions{ScopeRoot: workspace, PreviewLength: 160})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 1 || items[0].Cwd != session.NormalizePath(workspace) {
		t.Fatalf("Discover() = %#v, want the workspace from the store bytes", items)
	}
	// 没有任何提示词可用时,标题退回会话 ID,而不是留空。
	if items[0].Title != "agy-2" {
		t.Fatalf("Title = %q, want the session id", items[0].Title)
	}
}

// -wal / -shm 是会话库的附属文件,不能被当成第二个会话。
func TestAntigravityDiscoverIgnoresSidecarFiles(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	conversations := filepath.Join(root, "antigravity-cli", "conversations")
	if err := os.MkdirAll(conversations, 0o755); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}
	// 主库里没有元数据,工作目录只存在于尚未 checkpoint 的 WAL 中。
	if err := os.WriteFile(filepath.Join(conversations, "agy-3.db"), []byte("empty"), 0o600); err != nil {
		t.Fatalf("write conversation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(conversations, "agy-3.db-wal"), protoString(fileURI(workspace)), 0o600); err != nil {
		t.Fatalf("write wal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(conversations, "agy-3.db-shm"), []byte("shm"), 0o600); err != nil {
		t.Fatalf("write shm: %v", err)
	}
	t.Setenv("ANTIGRAVITY_HOME", root)

	items, err := Antigravity{}.Discover(session.ScanOptions{ScopeRoot: workspace, PreviewLength: 160})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != "agy-3" {
		t.Fatalf("Discover() = %#v, want exactly one session named agy-3", items)
	}
}

// 作用域之外的会话不该出现;数据目录不存在时也不该报错。
func TestAntigravityDiscoverRespectsScopeAndMissingRoot(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	conversations := filepath.Join(root, "antigravity-cli", "conversations")
	if err := os.MkdirAll(conversations, 0o755); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}
	blob := protoString(fileURI(workspace))
	if err := os.WriteFile(filepath.Join(conversations, "agy-4.db"), blob, 0o600); err != nil {
		t.Fatalf("write conversation: %v", err)
	}
	t.Setenv("ANTIGRAVITY_HOME", root)

	elsewhere := t.TempDir()
	items, err := Antigravity{}.Discover(session.ScanOptions{ScopeRoot: elsewhere, PreviewLength: 160})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("作用域外的会话应被排除, got %#v", items)
	}
	if items, err := (Antigravity{}).Discover(session.ScanOptions{ScopeRoot: elsewhere, ScopeAll: true, PreviewLength: 160}); err != nil || len(items) != 1 {
		t.Fatalf("ScopeAll 应当纳入全部会话, got %#v, %v", items, err)
	}

	t.Setenv("ANTIGRAVITY_HOME", filepath.Join(t.TempDir(), "absent"))
	if items, err := (Antigravity{}).Discover(session.ScanOptions{ScopeAll: true}); err != nil || len(items) != 0 {
		t.Fatalf("数据目录缺失时应安静返回空, got %#v, %v", items, err)
	}
}

// 长度前缀对不上的 `file://` 只是碰巧出现的片段,不能采信。
func TestAntigravityWorkspacesRequireMatchingLengthPrefix(t *testing.T) {
	if got := antigravityWorkspaces([]byte("\x05file:///tmp/whatever")); len(got) != 0 {
		t.Fatalf("长度前缀不符时应放弃, got %#v", got)
	}
	if got := antigravityWorkspaces([]byte("file:///tmp/whatever")); len(got) != 0 {
		t.Fatalf("没有长度前缀时应放弃, got %#v", got)
	}
	// 前一个片段不可信时,应继续找后面那个合法的。
	data := append([]byte("\x05file:///bogus"), protoString("file:///tmp/real")...)
	got := antigravityWorkspaces(data)
	if len(got) != 1 || got[0] != filepath.FromSlash("/tmp/real") {
		t.Fatalf("antigravityWorkspaces() = %#v, want [/tmp/real]", got)
	}
}

// 会话正文里夹着的示例路径也带合法长度前缀,只能靠"是不是存在的目录"分辨。
// 真的工作目录还会被反复写进元数据,所以次数多的那个胜出。
func TestAntigravityWorkspaceFallbackRejectsNonDirectories(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	conversations := filepath.Join(root, "antigravity-cli", "conversations")
	if err := os.MkdirAll(conversations, 0o755); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}
	// 先写一个不存在的示例路径,再写两次真实工作目录 —— 取第一个匹配就会挑错。
	blob := protoString("file:///absolute/path/to/file")
	blob = append(blob, protoString(fileURI(workspace))...)
	blob = append(blob, protoString(fileURI(workspace))...)
	if err := os.WriteFile(filepath.Join(conversations, "agy-5.db"), blob, 0o600); err != nil {
		t.Fatalf("write conversation: %v", err)
	}
	t.Setenv("ANTIGRAVITY_HOME", root)

	items, err := Antigravity{}.Discover(session.ScanOptions{ScopeRoot: workspace, PreviewLength: 160})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(items) != 1 || items[0].Cwd != session.NormalizePath(workspace) {
		t.Fatalf("Discover() = %#v, want the real workspace", items)
	}
}

func quoteJSON(value string) string {
	return "\"" + strings.ReplaceAll(value, "\\", "\\\\") + "\""
}
