package ui

import (
	"testing"

	"github.com/esverde/ais/internal/config"
	"github.com/esverde/ais/internal/session"
)

func TestNextProvider(t *testing.T) {
	checks := map[string]string{config.ProviderAll: config.ProviderClaude, config.ProviderClaude: config.ProviderCodex, config.ProviderCodex: config.ProviderAll}
	for input, want := range checks {
		if got := nextProvider(input); got != want {
			t.Fatalf("nextProvider(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSessionItemFilterValueIncludesSearchableFields(t *testing.T) {
	item := sessionItem{value: session.Session{Provider: session.ProviderCodex, ID: "abc", Cwd: "repo", Title: "title", LastUser: "prompt"}}
	value := item.FilterValue()
	for _, fragment := range []string{"codex", "abc", "repo", "title", "prompt"} {
		if !contains(value, fragment) {
			t.Fatalf("FilterValue() = %q, missing %q", value, fragment)
		}
	}
}

func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
