package session

import (
	"path/filepath"
	"testing"
)

func TestIsWithin(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "nested", "repo")
	outside := filepath.Join(filepath.Dir(root), "other")
	if !IsWithin(root, child) {
		t.Fatalf("IsWithin(root, child) = false")
	}
	if !IsWithin(root, root) {
		t.Fatalf("IsWithin(root, root) = false")
	}
	if IsWithin(root, outside) {
		t.Fatalf("IsWithin(root, outside) = true")
	}
}
