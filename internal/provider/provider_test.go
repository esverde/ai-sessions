package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/esverde/ais/internal/session"
)

func TestParseClaudeFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "claude.jsonl")
	cwd := filepath.ToSlash(root)
	data := "{\"type\":\"user\",\"sessionId\":\"claude-1\",\"cwd\":\"" + cwd + "\",\"timestamp\":\"2026-08-05T01:02:03Z\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"first prompt\"}]}}\n" +
		"{\"type\":\"ai-title\",\"sessionId\":\"claude-1\",\"aiTitle\":\"A Claude session\"}\n" +
		"{\"type\":\"assistant\",\"sessionId\":\"claude-1\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"assistant answer\"}]}}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	item, ok, err := parseClaudeFile(path, session.ScanOptions{ScopeRoot: root, PreviewLength: 160}, true, true)
	if err != nil || !ok {
		t.Fatalf("parseClaudeFile() = %#v, %v, %v", item, ok, err)
	}
	if item.ID != "claude-1" || item.Title != "A Claude session" || item.LastUser != "first prompt" || item.LastAssistant != "assistant answer" {
		t.Fatalf("parsed Claude session = %#v", item)
	}
}

func TestParseCodexFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout.jsonl")
	cwd := filepath.ToSlash(root)
	data := "{\"type\":\"session_meta\",\"timestamp\":\"2026-08-05T01:02:03Z\",\"payload\":{\"id\":\"codex-1\",\"cwd\":\"" + cwd + "\"}}\n" +
		"{\"type\":\"event_msg\",\"payload\":{\"type\":\"user_message\",\"message\":\"codex prompt\"}}\n" +
		"{\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"codex answer\"}]}}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	item, ok, err := parseCodexFile(path, false, map[string]string{"codex-1": "Codex title"}, session.ScanOptions{ScopeRoot: root, PreviewLength: 160})
	if err != nil || !ok {
		t.Fatalf("parseCodexFile() = %#v, %v, %v", item, ok, err)
	}
	if item.ID != "codex-1" || item.Title != "Codex title" || item.LastUser != "codex prompt" || item.LastAssistant != "codex answer" {
		t.Fatalf("parsed Codex session = %#v", item)
	}
}

func TestDeduplicatePrefersActiveSession(t *testing.T) {
	items := Deduplicate([]session.Session{{Provider: "codex", ID: "1", Archived: true}, {Provider: "codex", ID: "1", Archived: false}})
	if len(items) != 1 || items[0].Archived {
		t.Fatalf("Deduplicate() = %#v", items)
	}
}

// 会话被搬到另一个项目的 projects/<slug>/ 目录后,文件里的 cwd 仍是搬迁前的旧路径。
// 归属只认所在 slug 目录:slug 匹配就纳入(哪怕 cwd 是别的路径),不匹配就排除
// (哪怕 cwd 指向当前作用域)——否则搬走的会话会同时出现在新旧两个项目里。
func TestParseClaudeFileAcceptsRelocatedSession(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "claude.jsonl")
	data := "{\"type\":\"user\",\"sessionId\":\"claude-moved\",\"cwd\":\"D:\\\\Old\\\\Project\"," +
		"\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"hello\"}]}}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	options := session.ScanOptions{ScopeRoot: root, PreviewLength: 160}

	if _, ok, err := parseClaudeFile(path, options, false, false); err != nil || ok {
		t.Fatalf("slug 不匹配时应排除, got ok=%v err=%v", ok, err)
	}
	item, ok, err := parseClaudeFile(path, options, true, false)
	if err != nil || !ok {
		t.Fatalf("slug 匹配时应纳入: ok=%v err=%v", ok, err)
	}
	if item.ID != "claude-moved" {
		t.Fatalf("parsed session = %#v", item)
	}
}

// 只有元信息、没有 cwd 的会话文件不应被丢弃,Cwd 用作用域补齐。
func TestParseClaudeFileWithoutCwd(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "claude.jsonl")
	data := "{\"type\":\"user\",\"sessionId\":\"claude-nocwd\"," +
		"\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"hi\"}]}}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	item, ok, err := parseClaudeFile(path, session.ScanOptions{ScopeRoot: root, PreviewLength: 160}, true, true)
	if err != nil || !ok {
		t.Fatalf("无 cwd 但 slug 匹配时应纳入: ok=%v err=%v", ok, err)
	}
	if item.Cwd == "" {
		t.Fatalf("Cwd 应回填为作用域, got %#v", item)
	}
}

// 核心回归:搬迁后的会话文件里 cwd 仍是旧项目路径,但 slug 精确匹配当前作用域。
// 此时 Cwd 必须以作用域为准 —— 它会被 launcher 当作 resume 的工作目录,
// 照旧值启动等于把人带回旧目录。
func TestParseClaudeFileOverridesCwdOnExactSlug(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "claude.jsonl")
	data := "{\"type\":\"user\",\"sessionId\":\"claude-moved\",\"cwd\":\"D:/Old/Project\"," +
		"\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"hello\"}]}}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	options := session.ScanOptions{ScopeRoot: root, PreviewLength: 160}

	item, ok, err := parseClaudeFile(path, options, true, true)
	if err != nil || !ok {
		t.Fatalf("parseClaudeFile: ok=%v err=%v", ok, err)
	}
	if item.Cwd != session.NormalizePath(root) {
		t.Fatalf("exact slug 时 Cwd 应为作用域, got %q want %q", item.Cwd, session.NormalizePath(root))
	}

	// 子目录会话(within 但非 exact)保留自己的真实路径,不被作用域覆盖。
	item, ok, err = parseClaudeFile(path, options, true, false)
	if err != nil || !ok {
		t.Fatalf("parseClaudeFile: ok=%v err=%v", ok, err)
	}
	if item.Cwd == session.NormalizePath(root) {
		t.Fatalf("非 exact 时不应覆盖 Cwd, got %q", item.Cwd)
	}
}
