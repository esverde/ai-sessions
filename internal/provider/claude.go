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

	result := make([]session.Session, 0)
	err = filepath.WalkDir(projects, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := strings.ToLower(entry.Name())
			if name == "subagents" || name == "tool-results" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".jsonl" {
			return nil
		}
		item, ok, parseErr := parseClaudeFile(path, options)
		if parseErr != nil {
			return nil
		}
		if ok {
			result = append(result, item)
		}
		return nil
	})
	return result, err
}

func parseClaudeFile(path string, options session.ScanOptions) (session.Session, bool, error) {
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
	if item.Cwd == "" {
		return session.Session{}, false, nil
	}
	item.Cwd = session.NormalizePath(item.Cwd)
	if !options.ScopeAll && !session.IsWithin(options.ScopeRoot, item.Cwd) {
		return session.Session{}, false, nil
	}
	item.LastUser = FirstNonEmpty(lastPrompt, item.LastUser)
	item.Title = FirstNonEmpty(item.Title, lastPrompt, firstUser, item.ID)
	item.LastUser = Truncate(item.LastUser, options.PreviewLength)
	item.LastAssistant = Truncate(item.LastAssistant, options.PreviewLength)
	return item, true, nil
}
