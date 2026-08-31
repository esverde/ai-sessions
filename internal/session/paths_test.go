package session

import (
	"path/filepath"
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

func TestClaudeProjectSlug(t *testing.T) {
	cases := map[string]string{
		`d:\Documents\Code\Aventon_MMM`:  "d--Documents-Code-Aventon-MMM",
		`d:\Documents\Code\MMM_Meridian`: "d--Documents-Code-MMM-Meridian",
		`D:\Documents\Claude`:            "D--Documents-Claude",
	}
	for in, want := range cases {
		if got := ClaudeProjectSlug(in); !strings.EqualFold(got, want) {
			t.Fatalf("ClaudeProjectSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClaudeSlugMatch(t *testing.T) {
	root := `d:\Documents\Code\Aventon_MMM`
	if within, exact := ClaudeSlugMatch(root, "d--Documents-Code-Aventon-MMM"); !within || !exact {
		t.Fatalf("同名 slug 应精确匹配, got within=%v exact=%v", within, exact)
	}
	// 子目录会话要纳入,但不是 exact —— 它的工作目录是子目录,不能被作用域覆盖。
	if within, exact := ClaudeSlugMatch(root, "d--Documents-Code-Aventon-MMM-src"); !within || exact {
		t.Fatalf("子目录会话应 within 但非 exact, got within=%v exact=%v", within, exact)
	}
	if within, _ := ClaudeSlugMatch(root, "d--Documents-Code-Aventon-MMMX"); within {
		t.Fatal("仅前缀相同的兄弟项目不应匹配")
	}
	if within, _ := ClaudeSlugMatch(root, "d--Documents-Code-MMM-Meridian"); within {
		t.Fatal("其它项目不应匹配")
	}
}
