package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SkillCandidate struct {
	PatternName string
	LessonPaths []string
	Description string
}

const skillTemplate = `# %s

> 来源：经验库自动生成
> 触发次数：%d

## 模式描述
%s

## 来源教训
%s

## 使用指南
当遇到类似情况时，参考以上教训。具体步骤由 Agent 根据上下文判断。
`

func (s *KBStore) FindSkillCandidates(scope string) ([]SkillCandidate, error) {
	files, err := s.ListFiles(scope, CategoryLesson)
	if err != nil {
		return nil, err
	}

	groups := clusterByTags(files)
	var candidates []SkillCandidate
	for _, group := range groups {
		if len(group) >= 3 {
			var paths []string
			var contentParts []string
			for _, f := range group {
				paths = append(paths, f.Path)
				body, _ := s.ReadFile(scope, f.Path)
				contentParts = append(contentParts, ExtractBody(body))
			}
			candidates = append(candidates, SkillCandidate{
				PatternName: deriveSkillName(group),
				LessonPaths: paths,
				Description: strings.Join(contentParts, "\n\n"),
			})
		}
	}
	return candidates, nil
}

func GenerateSkill(candidate SkillCandidate, skillsDir string) error {
	slug := titleToSlug(candidate.PatternName)
	skillDir := filepath.Join(skillsDir, slug)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	var sourceList strings.Builder
	for _, p := range candidate.LessonPaths {
		sourceList.WriteString(fmt.Sprintf("- %s\n", p))
	}

	content := fmt.Sprintf(skillTemplate,
		candidate.PatternName,
		len(candidate.LessonPaths),
		firstParagraph(candidate.Description),
		sourceList.String(),
	)

	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
}

func (s *KBStore) BackgroundSkillGen(ctx context.Context, llm interface{}, skillsDir string) error {
	candidates, err := s.FindSkillCandidates(ScopeProject)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if err := GenerateSkill(c, skillsDir); err != nil {
			fmt.Printf("[memory] skill gen failed for %s: %v\n", c.PatternName, err)
			continue
		}
		s.LogEntry(ScopeProject, "生成技能", fmt.Sprintf("检测到重复模式，已生成 Skill: %s", c.PatternName))
	}
	return nil
}

func clusterByTags(files []KBFile) [][]KBFile {
	tagIndex := make(map[string][]*KBFile)
	for i := range files {
		f := &files[i]
		for _, tag := range f.Tags {
			tagIndex[tag] = append(tagIndex[tag], f)
		}
	}
	seen := make(map[string]bool)
	var groups [][]KBFile
	for _, fgroup := range tagIndex {
		var group []KBFile
		for _, f := range fgroup {
			if !seen[f.Path] {
				seen[f.Path] = true
				group = append(group, *f)
			}
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func deriveSkillName(group []KBFile) string {
	if len(group) == 0 {
		return "untitled"
	}
	return group[0].Title
}

func firstParagraph(text string) string {
	if idx := strings.Index(text, "\n\n"); idx >= 0 {
		return text[:idx]
	}
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}
