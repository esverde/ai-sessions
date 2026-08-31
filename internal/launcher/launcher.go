package launcher

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/esverde/ais/internal/session"
)

type Spec struct {
	Program string
	Args    []string
	Dir     string
}

func ResumeSpec(item session.Session) (Spec, error) {
	if item.ID == "" {
		return Spec{}, errors.New("session ID is empty")
	}
	if item.Cwd == "" {
		return Spec{}, errors.New("session working directory is empty")
	}
	var program string
	switch item.Provider {
	case session.ProviderClaude:
		program = "claude"
	case session.ProviderCodex:
		program = "codex"
	default:
		return Spec{}, fmt.Errorf("unsupported provider %q", item.Provider)
	}
	resolved, err := exec.LookPath(program)
	if err != nil {
		return Spec{}, fmt.Errorf("%s CLI was not found on PATH: %w", program, err)
	}
	return Spec{Program: resolved, Args: resumeArgs(item), Dir: item.Cwd}, nil
}

func Command(spec Spec) (*exec.Cmd, error) {
	if spec.Program == "" || spec.Dir == "" {
		return nil, errors.New("resume command requires program and working directory")
	}
	if runtime.GOOS == "windows" && isWindowsScript(spec.Program) {
		commandLine := quoteWindowsArg(spec.Program)
		for _, arg := range spec.Args {
			commandLine += " " + quoteWindowsArg(arg)
		}
		cmd := exec.Command("cmd.exe", "/D", "/S", "/C", commandLine)
		cmd.Dir = spec.Dir
		return cmd, nil
	}
	cmd := exec.Command(spec.Program, spec.Args...)
	cmd.Dir = spec.Dir
	return cmd, nil
}

func resumeArgs(item session.Session) []string {
	if item.Provider == session.ProviderClaude {
		return []string{"--resume", item.ID}
	}
	return []string{"resume", item.ID}
}

func isWindowsScript(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".cmd" || ext == ".bat"
}

func quoteWindowsArg(value string) string {
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, " \t\"") {
		return value
	}
	var builder strings.Builder
	builder.WriteByte('"')
	backslashes := 0
	for _, char := range value {
		switch char {
		case '\\':
			backslashes++
		case '"':
			builder.WriteString(strings.Repeat("\\", backslashes*2+1))
			builder.WriteRune(char)
			backslashes = 0
		default:
			if backslashes > 0 {
				builder.WriteString(strings.Repeat("\\", backslashes))
				backslashes = 0
			}
			builder.WriteRune(char)
		}
	}
	if backslashes > 0 {
		builder.WriteString(strings.Repeat("\\", backslashes*2))
	}
	builder.WriteByte('"')
	return builder.String()
}
