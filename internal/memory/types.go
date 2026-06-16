package memory

import (
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

const (
	CategoryKnowledge = "wiki/knowledge"
	CategoryProfile   = "wiki/profile"
	CategoryLesson    = "wiki/lesson"
	CategoryTopic     = "wiki/topic"
	CategoryRawDoc    = "raw/doc"
	CategoryRawCode   = "raw/code"
)

type KBFile struct {
	ID         int64     `json:"id"`
	Path       string    `json:"path"`
	Scope      string    `json:"scope"`
	Category   string    `json:"category"`
	Title      string    `json:"title"`
	Tags       []string  `json:"tags"`
	Confidence string    `json:"confidence"`
	Status     string    `json:"status"`
	CharCount  int       `json:"char_count"`
	LinkedTo   []string  `json:"linked_to,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func GlobalKBPath(homeDir string) string {
	return filepath.Join(homeDir, ".monika", "kb")
}

func ProjectKBPath(projectDir string) string {
	return filepath.Join(projectDir, ".monika", "kb")
}

func KBSubdirs() []string {
	return []string{
		"raw/docs",
		"raw/code",
		"wiki/topics",
		"wiki/lessons",
		"wiki/.trash",
		".index",
		".trash",
	}
}

// ResolveWorkspaceRoot 从给定目录出发，找到 git 仓库的根目录。
// 对于 git worktree，返回主仓库根目录而非 worktree 子目录。
// 如果不是 git 仓库，返回原始目录。
func ResolveWorkspaceRoot(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return dir
	}
	commonDir := strings.TrimSpace(string(out))
	if commonDir == "" {
		return dir
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(dir, commonDir)
	}
	// commonDir 类似 /repo/.git，仓库根 = 其父目录
	root := filepath.Dir(commonDir)
	if root == "" || root == "." {
		return dir
	}
	return root
}
