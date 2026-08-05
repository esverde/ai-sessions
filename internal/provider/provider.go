package provider

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/esverde/ais/internal/session"
)

type Provider interface {
	Name() string
	Discover(options session.ScanOptions) ([]session.Session, error)
}

func Discover(providerName string, options session.ScanOptions) ([]session.Session, []error) {
	providers := []Provider{}
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "claude":
		providers = append(providers, Claude{})
	case "codex":
		providers = append(providers, Codex{})
	default:
		providers = append(providers, Claude{}, Codex{})
	}

	var sessions []session.Session
	var errs []error
	for _, item := range providers {
		found, err := item.Discover(options)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", item.Name(), err))
		}
		sessions = append(sessions, found...)
	}
	return sessions, errs
}

func Sort(sessions []session.Session, mode string) {
	sort.SliceStable(sessions, func(i, j int) bool {
		left, right := sessions[i], sessions[j]
		if mode == "path" {
			leftPath := strings.ToLower(left.Cwd)
			rightPath := strings.ToLower(right.Cwd)
			if leftPath != rightPath {
				return leftPath < rightPath
			}
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		if left.Provider != right.Provider {
			return left.Provider < right.Provider
		}
		return left.ID < right.ID
	})
}

func Deduplicate(sessions []session.Session) []session.Session {
	seen := make(map[string]int, len(sessions))
	result := make([]session.Session, 0, len(sessions))
	for _, item := range sessions {
		key := item.Provider + "\x00" + item.ID
		if index, ok := seen[key]; ok {
			// Prefer an active record over an archived copy of the same native id.
			if result[index].Archived && !item.Archived {
				result[index] = item
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, item)
	}
	return result
}

func StringValue(record map[string]any, key string) string {
	value, ok := record[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func MapValue(record map[string]any, key string) map[string]any {
	value, ok := record[key]
	if !ok {
		return nil
	}
	result, _ := value.(map[string]any)
	return result
}

func ParseTime(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05-07:00"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func MaxTime(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}

func ExtractText(value any) string {
	switch typed := value.(type) {
	case string:
		return cleanText(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if record, ok := item.(map[string]any); ok {
				typeName := StringValue(record, "type")
				if typeName == "tool_use" || typeName == "tool_result" || typeName == "thinking" || typeName == "reasoning" {
					continue
				}
			}
			if text := ExtractText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return cleanText(strings.Join(parts, " "))
	case map[string]any:
		for _, key := range []string{"text", "content", "message", "output_text", "input_text"} {
			if value, ok := typed[key]; ok {
				if text := ExtractText(value); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func Truncate(value string, maxRunes int) string {
	value = cleanText(value)
	if maxRunes < 1 || utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes-1]) + "…"
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
