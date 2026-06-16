package memory

import (
	"context"
	"fmt"
)

type ArchiveHook struct {
	Store          *KBStore
	LLM            ExtractionLLM
	CompactionLLM  CompactionLLM
	OnStatusChange func(status string)
}

func (h *ArchiveHook) OnArchive(ctx context.Context, scope, sessionID, compactionSummary string) {
	if h.OnStatusChange != nil {
		h.OnStatusChange("归纳中...")
	}

	if compactionSummary == "" {
		if h.OnStatusChange != nil {
			h.OnStatusChange("记忆已更新 ✓")
		}
		return
	}

	result, err := ExtractMemories(ctx, h.LLM, scope, sessionID, compactionSummary)
	if err != nil {
		fmt.Printf("[memory] extraction failed: %v\n", err)
		if h.OnStatusChange != nil {
			h.OnStatusChange("归纳失败")
		}
		return
	}

	written := 0
	for _, c := range result.Candidates {
		sims, _ := h.Store.ComputeSimilarity(c)
		action, target := h.Store.Consolidate(c, sims)

		var cat string
		switch c.Category {
		case "lesson":
			cat = CategoryLesson
		case "topic":
			cat = CategoryTopic
		case "knowledge_update":
			cat = CategoryKnowledge
		default:
			cat = CategoryLesson
		}

		switch action {
		case "update":
			if target != nil {
				existing, _ := h.Store.ReadFile(target.Scope, target.Path)
				body := ExtractBody(existing)
				merged := MergeFileContent(body, c.Content, c.Title)
				h.Store.WriteFile(target.Scope, cat, target.Title, merged, c.Tags, c.Confidence)
				h.Store.LogEntry(c.Scope, "合并记忆", fmt.Sprintf("更新 %s", target.Path))
			}
		case "new_linked", "new":
			h.Store.WriteFile(c.Scope, cat, c.Title, c.Content, c.Tags, c.Confidence)
			h.Store.LogEntry(c.Scope, "新建记忆", fmt.Sprintf("新建 %s", c.Title))
		}
		written++
	}

	if result.ProfileDelta != "" {
		existing, _ := h.Store.ReadFile(ScopeGlobal, "wiki/profile.md")
		if existing == "" {
			existing = "# User Profile\n\n"
		}
		updated := existing + "\n## 更新\n" + result.ProfileDelta
		h.Store.WriteFile(ScopeGlobal, CategoryProfile, "User Profile", updated, nil, "medium")
		h.Store.LogEntry(ScopeGlobal, "更新画像", "profile.md 已更新")
	}

	if h.CompactionLLM != nil {
		h.Store.CompactKnowledge(ctx, h.CompactionLLM, scope)
		h.Store.CompactKnowledge(ctx, h.CompactionLLM, ScopeGlobal)
	}

	h.Store.LogEntry(scope, "Session 归档", fmt.Sprintf("Session %s 归档完成，写入 %d 条记忆", sessionID, written))

	if h.OnStatusChange != nil {
		h.OnStatusChange("记忆已更新 ✓")
	}
}
