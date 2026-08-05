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
	item, ok, err := parseClaudeFile(path, session.ScanOptions{ScopeRoot: root, PreviewLength: 160})
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
