package memory

import (
	"context"
	"fmt"
	"strings"
)

const (
	MaxKnowledgeChars = 3000
	MaxProfileChars   = 1500
)

type CompactionLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

func (s *KBStore) CompactKnowledge(ctx context.Context, llm CompactionLLM, scope string) error {
	content, err := s.ReadFile(scope, "wiki/knowledge.md")
	if err != nil || content == "" {
		return nil
	}

	if len([]rune(content)) <= MaxKnowledgeChars {
		return nil
	}

	prompt := `你是一个知识压缩器。将以下 knowledge.md 压缩到 ` +
		fmt.Sprintf("%d 字符以内，保留高置信度的事实，淘汰低置信度或过时的信息。\n\n", MaxKnowledgeChars) +
		`规则：
- 保留所有用户偏好和硬约束
- 保留最近的、高频使用的知识
- 合并相关内容，删除重复
- 保留原始 markdown 结构

原始内容：
` + content

	compressed, err := llm.Chat(ctx, prompt, "")
	if err != nil {
		return fmt.Errorf("compact: %w", err)
	}

	runes := []rune(compressed)
	if len(runes) > MaxKnowledgeChars {
		compressed = string(runes[:MaxKnowledgeChars])
	}

	return s.WriteFile(scope, CategoryKnowledge, "Core Knowledge", compressed, nil, "high")
}

func (s *KBStore) CompactProfile(ctx context.Context, llm CompactionLLM, scope string) error {
	content, err := s.ReadFile(scope, "wiki/profile.md")
	if err != nil || content == "" {
		return nil
	}

	if len([]rune(content)) <= MaxProfileChars {
		return nil
	}

	prompt := `压缩以下 user profile 到 ` + fmt.Sprintf("%d", MaxProfileChars) +
		` 字符以内，保留最重要的偏好和事实：\n\n` + content

	compressed, err := llm.Chat(ctx, prompt, "")
	if err != nil {
		return fmt.Errorf("compact profile: %w", err)
	}

	runes := []rune(compressed)
	if len(runes) > MaxProfileChars {
		compressed = string(runes[:MaxProfileChars])
	}

	return s.WriteFile(scope, CategoryProfile, "User Profile", compressed, nil, "high")
}

func ExtractBody(content string) string {
	lines := strings.Split(content, "\n")
	var body []string
	fmDone := false
	for _, line := range lines {
		if !fmDone && strings.HasPrefix(line, "> ") {
			continue
		}
		if !fmDone && strings.HasPrefix(line, "# ") {
			fmDone = true
			continue
		}
		fmDone = true
		body = append(body, line)
	}
	return strings.Join(body, "\n")
}
