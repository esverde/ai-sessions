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

// AntigravityRoot 返回 Antigravity(即原 Gemini CLI)的数据根目录。
//
// Antigravity 沿用了 Gemini CLI 的 `~/.gemini`,并在其下按"哪个客户端写的"
// 分成 antigravity-cli / antigravity / antigravity-ide 几个应用数据目录。
// 官方没有提供改这个位置的环境变量,ANTIGRAVITY_HOME 是 ais 自己的覆盖开关
// —— 测试需要它,把 `~/.gemini` 搬走的人也需要它。
func AntigravityRoot() (string, error) {
	if value := strings.TrimSpace(os.Getenv("ANTIGRAVITY_HOME")); value != "" {
		return NormalizePath(value), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini"), nil
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
	return resolveExistingPrefix(abs)
}

// resolveExistingPrefix 解析路径中最长的已存在前缀,再把余下的部分拼回去。
//
// filepath.EvalSymlinks 要求整条路径都存在,而作用域比较经常涉及尚不存在的
// 子目录。若只对存在的整条路径解析,同一棵树下的两个路径会得到不一致的表示
// —— macOS 上 /var 是 /private/var 的符号链接,已存在的 root 被解析成
// /private/var/...,而尚不存在的 root/child 原样保留 /var/...,两者一比就
// 成了互不包含。逐级回退到能解析的那一层,可以让二者落在同一前缀上。
func resolveExistingPrefix(abs string) string {
	rest := ""
	current := abs
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			if rest == "" {
				return filepath.Clean(resolved)
			}
			return filepath.Clean(filepath.Join(resolved, rest))
		}
		parent := filepath.Dir(current)
		if parent == current {
			// 一路到根都无法解析(整条路径都不存在),保持原样。
			return abs
		}
		rest = filepath.Join(filepath.Base(current), rest)
		current = parent
	}
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
