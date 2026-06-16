package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ReviewResult struct {
	Conflicts      []Conflict
	Deprecated     []string
	UpgradesNeeded []Upgrade
	LinkAdditions  []LinkAddition
}

type Conflict struct {
	FileA string
	FileB string
	Issue string
}

type Upgrade struct {
	LessonPaths []string
	TopicTitle  string
}

type LinkAddition struct {
	Source string
	Target string
}

type ReviewLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

func (s *KBStore) Review(ctx context.Context, llm ReviewLLM, scope string) (*ReviewResult, error) {
	files, err := s.ListFiles(scope, "")
	if err != nil {
		return nil, err
	}

	var recent []KBFile
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	for _, f := range files {
		if f.UpdatedAt.After(weekAgo) && f.Status == "active" {
			recent = append(recent, f)
		}
	}

	if len(recent) < 2 {
		return &ReviewResult{}, nil
	}

	prompt := buildReviewPrompt(recent)
	resp, err := llm.Chat(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("review: %w", err)
	}

	result := parseReviewResponse(resp)
	return result, nil
}

func (s *KBStore) ExecuteReview(ctx context.Context, llm ReviewLLM, scope string) error {
	result, err := s.Review(ctx, llm, scope)
	if err != nil {
		return err
	}

	for _, c := range result.Conflicts {
		older := olderFile(c.FileA, c.FileB, scope, s)
		if older != "" {
			s.SoftDelete(scope, older)
			s.LogEntry(scope, "冲突解决", fmt.Sprintf("%s marked deprecated: %s", older, c.Issue))
		}
	}

	for _, d := range result.Deprecated {
		s.SoftDelete(scope, d)
		s.LogEntry(scope, "淘汰旧记忆", fmt.Sprintf("%s deprecated", d))
	}

	for _, u := range result.UpgradesNeeded {
		var mergedContent string
		for _, lp := range u.LessonPaths {
			content, _ := s.ReadFile(scope, lp)
			mergedContent += "\n\n" + ExtractBody(content)
		}
		s.WriteFile(scope, CategoryTopic, u.TopicTitle, mergedContent, nil, "medium")
		for _, lp := range u.LessonPaths {
			s.archiveFile(scope, lp)
		}
		s.LogEntry(scope, "知识升级", fmt.Sprintf("Lessons %v → topic %s", u.LessonPaths, u.TopicTitle))
	}

	for _, l := range result.LinkAdditions {
		s.addLink(scope, l.Source, l.Target)
	}

	return nil
}

func (s *KBStore) archiveFile(scope, path string) error {
	content, err := s.ReadFile(scope, path)
	if err != nil {
		return err
	}
	updated := strings.Replace(content, "> 状态：active", "> 状态：archived", 1)
	root := s.rootFor(scope)
	return os.WriteFile(filepath.Join(root, path), []byte(updated), 0644)
}

func (s *KBStore) addLink(scope, sourcePath, targetPath string) error {
	content, err := s.ReadFile(scope, sourcePath)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("> 关联：[[%s]]", targetPath)
	if strings.Contains(content, link) {
		return nil
	}
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "> 关联：") {
			lines[i] = strings.TrimSuffix(line, "\n") + " | [[" + targetPath + "]]"
			found = true
			break
		}
	}
	if !found {
		// Insert after the last metadata line ("> ..." lines in frontmatter)
		insertIdx := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "> ") {
				insertIdx = i + 1
			}
		}
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, link)
		newLines = append(newLines, lines[insertIdx:]...)
		lines = newLines
	}
	root := s.rootFor(scope)
	if err := os.WriteFile(filepath.Join(root, sourcePath), []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}
	// 同步 DB linked_to 列，让 memory_search 能直接返回依赖关系，无需 file_read 全文。
	return s.setLinkedTo(scope, sourcePath, parseFMLinks(strings.Join(lines, "\n")))
}

func olderFile(a, b, scope string, s *KBStore) string {
	files, _ := s.ListFiles(scope, "")
	var fa, fb *KBFile
	for _, f := range files {
		if f.Path == a {
			fa = &f
		}
		if f.Path == b {
			fb = &f
		}
	}
	if fa != nil && fb != nil && fa.UpdatedAt.Before(fb.UpdatedAt) {
		return a
	}
	if fb != nil {
		return b
	}
	return a
}

func buildReviewPrompt(files []KBFile) string {
	var list strings.Builder
	for _, f := range files {
		list.WriteString(fmt.Sprintf("- %s (%s) [%s]\n", f.Title, f.Category, f.Path))
	}
	return `Review the following recently modified knowledge entries. Find:
1. Conflicts — contradictory conclusions on the same topic
2. Deprecated — facts no longer applicable
3. Upgrades — multiple lessons revealing a common pattern → consolidate to topic
4. Missing links — related files not cross-linked

Entries:
` + list.String() + `
Respond in JSON format.`
}

func parseReviewResponse(resp string) *ReviewResult {
	jsonStr := strings.TrimSpace(resp)
	if idx := strings.Index(jsonStr, "```json"); idx >= 0 {
		jsonStr = jsonStr[idx+7:]
		if end := strings.Index(jsonStr, "```"); end >= 0 {
			jsonStr = jsonStr[:end]
		}
	} else if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
	}

	var result ReviewResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return &ReviewResult{}
	}
	return &result
}
