package session

import (
	"path/filepath"
	"runtime"
	"strings"
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

// 用例必须用当前平台的绝对路径:ClaudeProjectSlug 会先走 NormalizePath,
// 而 filepath.Abs 在 Linux/macOS 上不认 `c:\...` 是绝对路径,会把它拼到 cwd 后面。
// 各平台的 Claude 只会遇到本平台的路径,所以按平台取用例才是真实场景。
// 目录名里的下划线用于验证它同样会被编码成连字符。
func projectPaths() (primary, sibling string) {
	if runtime.GOOS == "windows" {
		return `c:\work\alpha_app`, `c:\work\beta_app`
	}
	return "/work/alpha_app", "/work/beta_app"
}

func TestClaudeProjectSlug(t *testing.T) {
	primary, sibling := projectPaths()
	for _, in := range []string{primary, sibling} {
		got := ClaudeProjectSlug(in)
		// 逐字符核对编码规则:字母数字原样保留,其余一律变成单个连字符。
		want := make([]byte, 0, len(got))
		for _, r := range NormalizePath(in) {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
				want = append(want, byte(r))
			default:
				want = append(want, '-')
			}
		}
		if got != string(want) {
			t.Fatalf("ClaudeProjectSlug(%q) = %q, want %q", in, got, string(want))
		}
		if strings.ContainsAny(got, `\/:_. `) {
			t.Fatalf("slug 不应残留路径分隔符或下划线: %q", got)
		}
	}
	if ClaudeProjectSlug(primary) == ClaudeProjectSlug(sibling) {
		t.Fatal("不同项目的 slug 不应相同")
	}
}

func TestClaudeSlugMatch(t *testing.T) {
	primary, sibling := projectPaths()
	slug := ClaudeProjectSlug(primary)

	if within, exact := ClaudeSlugMatch(primary, slug); !within || !exact {
		t.Fatalf("同名 slug 应精确匹配, got within=%v exact=%v", within, exact)
	}
	// 子目录会话要纳入,但不是 exact —— 它的工作目录是子目录,不能被作用域覆盖。
	if within, exact := ClaudeSlugMatch(primary, slug+"-src"); !within || exact {
		t.Fatalf("子目录会话应 within 但非 exact, got within=%v exact=%v", within, exact)
	}
	if within, _ := ClaudeSlugMatch(primary, slug+"X"); within {
		t.Fatal("仅前缀相同的兄弟项目不应匹配")
	}
	if within, _ := ClaudeSlugMatch(primary, ClaudeProjectSlug(sibling)); within {
		t.Fatal("其它项目不应匹配")
	}
}
