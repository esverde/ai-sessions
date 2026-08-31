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

func TestClaudeSlugMatches(t *testing.T) {
	root := `d:\Documents\Code\Aventon_MMM`
	if !ClaudeSlugMatches(root, "d--Documents-Code-Aventon-MMM") {
		t.Fatal("同名 slug 应匹配")
	}
	if !ClaudeSlugMatches(root, "d--Documents-Code-Aventon-MMM-src") {
		t.Fatal("子目录会话应匹配")
	}
	if ClaudeSlugMatches(root, "d--Documents-Code-Aventon-MMMX") {
		t.Fatal("仅前缀相同的兄弟项目不应匹配")
	}
	if ClaudeSlugMatches(root, "d--Documents-Code-MMM-Meridian") {
		t.Fatal("其它项目不应匹配")
	}
}
