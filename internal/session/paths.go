package session

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func CurrentDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	return NormalizePath(cwd), nil
}

func UserHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return NormalizePath(home), nil
}

func ClaudeRoot() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); value != "" {
		return NormalizePath(value), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

func CodexRoot() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return NormalizePath(value), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

// ClaudeProjectSlug 把工作目录编码成 Claude Code 用的 projects 子目录名。
// Claude 把路径里每个非字母数字字符替换成一个连字符,大小写原样保留,
// 例如 `d:\Documents\Code\Aventon_MMM` -> `d--Documents-Code-Aventon-MMM`。
//
// 这个 slug 才是 Claude 判定会话归属的事实来源:会话文件内部记录的 `cwd`
// 只是当时的历史值,目录被改名或会话被搬迁后就会失配,而 Claude 自己的
// `--resume` 依然按 slug 列出它们。
func ClaudeProjectSlug(value string) string {
	value = NormalizePath(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

// ClaudeSlugMatch 判断 projects 下的某个目录名与 root 作用域的关系。
//
// within 表示该目录的会话应纳入作用域;exact 表示目录名与 root 的 slug 完全相同,
// 也就是"这些会话就属于 root 本身"。以 `slug-` 开头的是 root 子目录里开出的会话,
// within 为真但 exact 为假 —— 它们的工作目录是子目录而非 root,不可混为一谈。
func ClaudeSlugMatch(root, dirName string) (within, exact bool) {
	slug := ClaudeProjectSlug(root)
	if slug == "" || dirName == "" {
		return false, false
	}
	if runtime.GOOS == "windows" {
		slug = strings.ToLower(slug)
		dirName = strings.ToLower(dirName)
	}
	if dirName == slug {
		return true, true
	}
	return strings.HasPrefix(dirName, slug+"-"), false
}

func NormalizePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		abs = value
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = filepath.Clean(resolved)
	}
	return abs
}

func IsWithin(root, candidate string) bool {
	root = comparisonPath(root)
	candidate = comparisonPath(candidate)
	if root == "" || candidate == "" {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return false
	}
	return true
}

func comparisonPath(value string) string {
	value = NormalizePath(value)
	if runtime.GOOS == "windows" {
		return strings.ToLower(value)
	}
	return value
}
