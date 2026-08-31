package provider

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/esverde/ais/internal/session"
)

type Claude struct{}

func (Claude) Name() string { return session.ProviderClaude }

func (Claude) Discover(options session.ScanOptions) ([]session.Session, error) {
	root, err := session.ClaudeRoot()
	if err != nil {
		return nil, err
	}
	projects := filepath.Join(root, "projects")
	if _, err := os.Stat(projects); os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(projects)
	if err != nil {
		return nil, err
	}

	result := make([]session.Session, 0)
	for _, project := range entries {
		if !project.IsDir() {
			continue
		}
		// Claude 按 projects/<slug>/ 归置会话,slug 由工作目录编码而来。
		// 目录匹配即在作用域内,不必再依赖会话文件里那个可能已过时的 cwd。
		inScope, exactScope := session.ClaudeSlugMatch(options.ScopeRoot, project.Name())
		dir := filepath.Join(projects, project.Name())
		walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				name := strings.ToLower(entry.Name())
				if name == "subagents" || name == "tool-results" || name == "memory" {
					return fs.SkipDir
				}
				return nil
			}
			if strings.ToLower(filepath.Ext(entry.Name())) != ".jsonl" {
				return nil
			}
			item, ok, parseErr := parseClaudeFile(path, options, inScope, exactScope)
			if parseErr != nil {
				return nil
			}
			if ok {
				result = append(result, item)
			}
			return nil
		})
		if walkErr != nil {
			return result, walkErr
		}
	}
	return result, nil
}

// inScopeBySlug 表示该文件所在的 projects/<slug>/ 目录匹配当前作用域;
// exactSlug 进一步表示 slug 与作用域完全相同,即会话就属于 ScopeRoot 本身。
func parseClaudeFile(path string, options session.ScanOptions, inScopeBySlug, exactSlug bool) (session.Session, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return session.Session{}, false, err
	}
	item := session.Session{
		Provider:   session.ProviderClaude,
		SourcePath: path,
		UpdatedAt:  info.ModTime(),
	}
	var firstUser string
	var lastPrompt string
	var hasRecord bool

	err = session.ReadJSONLLines(path, func(record map[string]any) error {
		hasRecord = true
		item.ID = FirstNonEmpty(item.ID, StringValue(record, "sessionId"), StringValue(record, "session_id"))
		item.Cwd = FirstNonEmpty(item.Cwd, StringValue(record, "cwd"))
		item.Title = FirstNonEmpty(item.Title, StringValue(record, "customTitle"), StringValue(record, "aiTitle"), StringValue(record, "summary"))
		lastPrompt = FirstNonEmpty(lastPrompt, StringValue(record, "lastPrompt"))
		if timestamp, ok := ParseTime(record["timestamp"]); ok {
			item.UpdatedAt = MaxTime(item.UpdatedAt, timestamp)
		}

		switch StringValue(record, "type") {
		case "ai-title", "custom-title":
			item.Title = FirstNonEmpty(StringValue(record, "customTitle"), StringValue(record, "aiTitle"), item.Title)
		case "last-prompt":
			lastPrompt = FirstNonEmpty(StringValue(record, "lastPrompt"), lastPrompt)
		case "user":
			message := MapValue(record, "message")
			text := ExtractText(message)
			if text == "" {
				text = ExtractText(record["content"])
			}
			if text != "" {
				if firstUser == "" {
					firstUser = text
				}
				item.LastUser = text
			}
		case "assistant":
			message := MapValue(record, "message")
			text := ExtractText(message)
			if text == "" {
				text = ExtractText(record["content"])
			}
			if text != "" {
				item.LastAssistant = text
			}
		}
		return nil
	})
	if err != nil {
		return session.Session{}, false, err
	}
	if !hasRecord || item.ID == "" {
		return session.Session{}, false, nil
	}
	// 归属只看所在的 projects/<slug>/ 目录 —— 那是 Claude 判定归属的事实来源。
	// 不用文件里的 cwd:它是会话开始时的历史值,目录改名或会话被搬到别的项目后
	// 就会失配;若拿它兜底,搬走的会话还会同时出现在旧项目里,造成重复归属。
	if !options.ScopeAll && !inScopeBySlug {
		return session.Session{}, false, nil
	}
	// Cwd 会被当作 resume 时的工作目录(launcher.ResumeSpec 的 Dir)。
	// slug 精确匹配时以作用域为准:文件里的 cwd 是会话开始时的历史值,项目改名或
	// 会话被搬迁后就是旧路径,照它启动会把人带回旧目录。slug 只是前缀匹配(子目录
	// 会话)或全局扫描到的别的项目,则保留文件里的 cwd —— 那才是它们的真实位置。
	switch {
	case exactSlug:
		item.Cwd = session.NormalizePath(options.ScopeRoot)
	case item.Cwd != "":
		item.Cwd = session.NormalizePath(item.Cwd)
	case inScopeBySlug:
		item.Cwd = session.NormalizePath(options.ScopeRoot)
	}
	item.LastUser = FirstNonEmpty(lastPrompt, item.LastUser)
	item.Title = FirstNonEmpty(item.Title, lastPrompt, firstUser, item.ID)
	item.LastUser = Truncate(item.LastUser, options.PreviewLength)
	item.LastAssistant = Truncate(item.LastAssistant, options.PreviewLength)
	return item, true, nil
}
