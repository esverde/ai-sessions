package provider

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/esverde/ais/internal/session"
)

type Codex struct{}

func (Codex) Name() string { return session.ProviderCodex }

func (Codex) Discover(options session.ScanOptions) ([]session.Session, error) {
	root, err := session.CodexRoot()
	if err != nil {
		return nil, err
	}
	titles := readCodexTitles(filepath.Join(root, "session_index.jsonl"))
	result := make([]session.Session, 0)
	if err := walkCodex(filepath.Join(root, "sessions"), false, titles, options, &result); err != nil {
		return nil, err
	}
	if options.IncludeArchived {
		if err := walkCodex(filepath.Join(root, "archived_sessions"), true, titles, options, &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func walkCodex(root string, archived bool, titles map[string]string, options session.ScanOptions, result *[]session.Session) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".jsonl" {
			return nil
		}
		item, ok, err := parseCodexFile(path, archived, titles, options)
		if err != nil {
			return nil
		}
		if ok {
			*result = append(*result, item)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func readCodexTitles(path string) map[string]string {
	titles := make(map[string]string)
	_ = session.ReadJSONLLines(path, func(record map[string]any) error {
		id := FirstNonEmpty(StringValue(record, "id"), StringValue(record, "session_id"))
		title := FirstNonEmpty(StringValue(record, "thread_name"), StringValue(record, "title"), StringValue(record, "name"))
		if id != "" && title != "" {
			titles[id] = title
		}
		return nil
	})
	return titles
}

func parseCodexFile(path string, archived bool, titles map[string]string, options session.ScanOptions) (session.Session, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return session.Session{}, false, err
	}
	item := session.Session{
		Provider:   session.ProviderCodex,
		SourcePath: path,
		Archived:   archived,
		UpdatedAt:  info.ModTime(),
	}
	var firstUser string
	var fallbackAssistant string
	var isSubagent bool

	err = session.ReadJSONLLines(path, func(record map[string]any) error {
		payload := MapValue(record, "payload")
		if payload == nil {
			payload = record
		}
		item.ID = FirstNonEmpty(item.ID, StringValue(payload, "id"), StringValue(record, "session_id"))
		item.Cwd = FirstNonEmpty(item.Cwd, StringValue(payload, "cwd"), StringValue(record, "cwd"))
		if threadSource := StringValue(payload, "thread_source"); threadSource == "subagent" {
			isSubagent = true
		}
		if source := MapValue(payload, "source"); source != nil && source["subagent"] != nil {
			isSubagent = true
		}
		if timestamp, ok := ParseTime(record["timestamp"]); ok {
			item.UpdatedAt = MaxTime(item.UpdatedAt, timestamp)
		}

		eventType := StringValue(record, "type")
		payloadType := StringValue(payload, "type")
		if eventType == "event_msg" && payloadType == "user_message" {
			text := FirstNonEmpty(StringValue(payload, "message"), ExtractText(payload["content"]))
			if text != "" {
				if firstUser == "" {
					firstUser = text
				}
				item.LastUser = text
			}
		}
		if eventType == "event_msg" && payloadType == "task_complete" {
			fallbackAssistant = FirstNonEmpty(StringValue(payload, "last_agent_message"), fallbackAssistant)
		}
		if eventType == "response_item" && payloadType == "message" {
			role := StringValue(payload, "role")
			text := ExtractText(payload["content"])
			if role == "user" && item.LastUser == "" {
				item.LastUser = text
			}
			if role == "assistant" && text != "" {
				item.LastAssistant = text
			}
		}
		if eventType == "response_item" && payloadType == "agent_message" {
			fallbackAssistant = FirstNonEmpty(ExtractText(payload["text"]), fallbackAssistant)
		}
		return nil
	})
	if err != nil {
		return session.Session{}, false, err
	}
	if isSubagent || item.ID == "" || item.Cwd == "" {
		return session.Session{}, false, nil
	}
	item.Cwd = session.NormalizePath(item.Cwd)
	if !options.ScopeAll && !session.IsWithin(options.ScopeRoot, item.Cwd) {
		return session.Session{}, false, nil
	}
	item.Title = FirstNonEmpty(titles[item.ID], firstUser, fallbackAssistant, item.ID)
	item.LastAssistant = FirstNonEmpty(item.LastAssistant, fallbackAssistant)
	item.LastUser = Truncate(item.LastUser, options.PreviewLength)
	item.LastAssistant = Truncate(item.LastAssistant, options.PreviewLength)
	item.Title = Truncate(item.Title, options.PreviewLength)
	return item, true, nil
}
