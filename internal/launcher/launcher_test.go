package launcher

import (
	"runtime"
	"testing"

	"github.com/esverde/ais/internal/session"
)

func TestResumeArgs(t *testing.T) {
	claude, err := ResumeSpec(session.Session{Provider: session.ProviderClaude, ID: "c1", Cwd: t.TempDir()})
	if err != nil {
		// The test machine may not have the CLI in a minimal CI environment.
		// Verify the provider argument shape through the pure helper instead.
		if got := resumeArgs(session.Session{Provider: session.ProviderClaude, ID: "c1"}); len(got) != 2 || got[0] != "--resume" || got[1] != "c1" {
			t.Fatalf("Claude resume args = %#v", got)
		}
		return
	}
	if len(claude.Args) != 2 || claude.Args[0] != "--resume" || claude.Args[1] != "c1" {
		t.Fatalf("Claude spec = %#v", claude)
	}

	if got := resumeArgs(session.Session{Provider: session.ProviderCodex, ID: "x1"}); got[0] != "resume" || got[1] != "x1" {
		t.Fatalf("Codex resume args = %#v", got)
	}
}

func TestCommandUsesWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		spec := Spec{Program: "cmd.exe", Args: []string{"/C", "exit", "0"}, Dir: t.TempDir()}
		cmd, err := Command(spec)
		if err != nil || cmd.Dir != spec.Dir {
			t.Fatalf("Command() = %#v, %v", cmd, err)
		}
	}
}
