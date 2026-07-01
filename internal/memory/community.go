package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Community struct {
	ID       int
	Tags     []string
	Memories []KBFile
}

type ClusterLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

func (s *KBStore) DetectTagCommunities(scope string) ([]Community, error) {
	files, err := s.ListFiles(scope, "")
	if err != nil {
		return nil, err
	}

	tagToMems := make(map[string][]KBFile)
	for _, f := range files {
		if f.Status != "active" {
			continue
		}
		for _, tag := range f.Tags {
			tagToMems[tag] = append(tagToMems[tag], f)
		}
	}

	coOccur := make(map[string]map[string]int)
	for _, mems := range tagToMems {
		for _, m1 := range mems {
			for _, t1 := range m1.Tags {
				if coOccur[t1] == nil {
					coOccur[t1] = make(map[string]int)
				}
				for _, t2 := range m1.Tags {
					if t1 != t2 {
						coOccur[t1][t2]++
					}
				}
			}
		}
	}

	visited := make(map[string]int)
	var communities []Community
	commID := 0

	for tag := range tagToMems {
		if visited[tag] > 0 {
			continue
		}
		commID++
		visited[tag] = commID

		stack := []string{tag}
		for len(stack) > 0 {
			t := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for neighbor, count := range coOccur[t] {
				if count >= 2 && visited[neighbor] == 0 {
					visited[neighbor] = commID
					stack = append(stack, neighbor)
				}
			}
		}
	}

	grouped := make(map[int][]string)
	for tag, cid := range visited {
		grouped[cid] = append(grouped[cid], tag)
	}

	memSeen := make(map[string]bool)
	for cid, tags := range grouped {
		if len(tags) < 2 {
			continue
		}
		sort.Strings(tags)

		var mems []KBFile
		for _, tag := range tags {
			for _, m := range tagToMems[tag] {
				if !memSeen[m.Path] {
					memSeen[m.Path] = true
					mems = append(mems, m)
				}
			}
		}

		if len(mems) >= 3 {
			communities = append(communities, Community{
				ID:       cid,
				Tags:     tags,
				Memories: mems,
			})
		}
	}

	return communities, nil
}

func (s *KBStore) GenerateTopicSummaries(ctx context.Context, llm ClusterLLM, scope string) error {
	lastFile := filepath.Join(s.rootFor(scope), ".index", "last_community.txt")
	if data, _ := os.ReadFile(lastFile); len(data) > 0 {
		if t, _ := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); time.Since(t) < 7*24*time.Hour {
			return nil
		}
	}

	communities, err := s.DetectTagCommunities(scope)
	if err != nil {
		return fmt.Errorf("detect communities: %w", err)
	}

	for _, comm := range communities {
		if len(comm.Memories) < 3 {
			continue
		}

		var memList strings.Builder
		for i, m := range comm.Memories {
			if i >= 15 {
				fmt.Fprintf(&memList, "... and %d more\n", len(comm.Memories)-15)
				break
			}
			fmt.Fprintf(&memList, "### %s\npath: %s\ntags: %s\n\n%s\n\n",
				m.Title, m.Path, strings.Join(m.Tags, ", "), m.Snippet)
		}

		systemPrompt := fmt.Sprintf(`你是一个知识归纳器。以下记忆属于同一个标签聚类（tags: %s）。
请生成一个简洁的技术主题总结，将分散的知识归纳为结构化的 wiki/topic。

要求：
- 标题：用聚类中最核心的概念命名
- 内容：markdown 格式，按主题分节，保留关键事实、代码引用、因果关系
- 不要丢失任何重要细节
- 标签：使用聚类的核心标签

返回 JSON：
{"title": "...", "content": "markdown...", "tags": ["tag1", "tag2"]}`, strings.Join(comm.Tags, ", "))

		resp, err := llm.Chat(ctx, systemPrompt, memList.String())
		if err != nil {
			continue
		}

		jsonStr := extractJSON(resp)
		var summary struct {
			Title   string   `json:"title"`
			Content string   `json:"content"`
			Tags    []string `json:"tags"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &summary); err != nil {
			continue
		}
		if summary.Title == "" || summary.Content == "" {
			continue
		}

		writeErr := s.writeFileUnchecked(scope, CategoryTopic, summary.Title, summary.Content, summary.Tags, "medium")
		if writeErr == nil {
			s.LogEntry(scope, "社区总结", fmt.Sprintf("聚类 tags=%v → topic '%s' (%d memories)",
				comm.Tags, summary.Title, len(comm.Memories)))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	os.MkdirAll(filepath.Dir(lastFile), 0755)
	os.WriteFile(lastFile, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	return nil
}
