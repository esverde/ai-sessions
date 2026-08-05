package session

import "time"

const (
	ProviderClaude = "claude"
	ProviderCodex  = "codex"
)

type Session struct {
	Provider      string
	ID            string
	Cwd           string
	Title         string
	LastUser      string
	LastAssistant string
	UpdatedAt     time.Time
	SourcePath    string
	Archived      bool
}

type ScanOptions struct {
	ScopeRoot       string
	ScopeAll        bool
	IncludeArchived bool
	PreviewLength   int
}
