package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadJSONLLinesSkipsMalformedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"user\"}\nnot-json\n{\"type\":\"assistant\"}\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var types []string
	if err := ReadJSONLLines(path, func(record map[string]any) error {
		types = append(types, record["type"].(string))
		return nil
	}); err != nil {
		t.Fatalf("ReadJSONLLines() error = %v", err)
	}
	if len(types) != 2 || types[0] != "user" || types[1] != "assistant" {
		t.Fatalf("types = %#v", types)
	}
}
