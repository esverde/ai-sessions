package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProviderAll    = "all"
	ProviderClaude = "claude"
	ProviderCodex  = "codex"

	ScopeCurrent = "current"
	ScopeAll     = "all"

	SortActive = "active"
	SortPath   = "path"
)

// Config contains the user-facing defaults for ais. The file is deliberately
// small and stable; provider-specific parsing never depends on it.
type Config struct {
	Provider        string `json:"provider"`
	Scope           string `json:"scope"`
	Sort            string `json:"sort"`
	IncludeArchived bool   `json:"include_archived"`
	PreviewLength   int    `json:"preview_length"`
	MaxSessions     int    `json:"max_sessions"`
}

func Default() Config {
	return Config{
		Provider:        ProviderAll,
		Scope:           ScopeCurrent,
		Sort:            SortActive,
		IncludeArchived: false,
		PreviewLength:   160,
		MaxSessions:     200,
	}
}

func (c Config) Normalize() Config {
	defaults := Default()
	c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
	if c.Provider != ProviderAll && c.Provider != ProviderClaude && c.Provider != ProviderCodex {
		c.Provider = defaults.Provider
	}

	c.Scope = strings.ToLower(strings.TrimSpace(c.Scope))
	if c.Scope != ScopeCurrent && c.Scope != ScopeAll {
		c.Scope = defaults.Scope
	}

	c.Sort = strings.ToLower(strings.TrimSpace(c.Sort))
	if c.Sort != SortActive && c.Sort != SortPath {
		c.Sort = defaults.Sort
	}

	if c.PreviewLength < 40 || c.PreviewLength > 2000 {
		c.PreviewLength = defaults.PreviewLength
	}
	if c.MaxSessions < 1 || c.MaxSessions > 5000 {
		c.MaxSessions = defaults.MaxSessions
	}
	return c
}

func Path() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AIS_CONFIG")); override != "" {
		return filepath.Clean(override), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(base, "ais", "config.json"), nil
}

func Load() (Config, string, error) {
	path, err := Path()
	if err != nil {
		return Default(), "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), path, nil
	}
	if err != nil {
		return Default(), path, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), path, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg.Normalize(), path, nil
}

func Save(path string, cfg Config) error {
	cfg = cfg.Normalize()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
