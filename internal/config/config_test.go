package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenConfigDoesNotExist(t *testing.T) {
	t.Setenv("AIS_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	cfg, path, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if path == "" || cfg != Default() {
		t.Fatalf("Load() = %#v, %q; want defaults and a path", cfg, path)
	}
}

func TestSaveAndLoadNormalizesValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	if err := Save(path, Config{Provider: "CLAUDE", Scope: "ALL", Sort: "PATH", PreviewLength: 80, MaxSessions: 25}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	t.Setenv("AIS_CONFIG", path)
	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Provider != ProviderClaude || cfg.Scope != ScopeAll || cfg.Sort != SortPath {
		t.Fatalf("normalized config = %#v", cfg)
	}
}

func TestNormalizeFallsBackForInvalidValues(t *testing.T) {
	cfg := (Config{Provider: "other", Scope: "project", Sort: "newest", PreviewLength: 2, MaxSessions: 0}).Normalize()
	if cfg != Default() {
		t.Fatalf("Normalize() = %#v, want defaults", cfg)
	}
}
