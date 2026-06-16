package memory

import (
	"path/filepath"
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
