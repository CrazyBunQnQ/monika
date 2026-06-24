package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ExtractCandidate struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Category   string   `json:"category"`
	Scope      string   `json:"scope"`
	Tags       []string `json:"tags"`
	Confidence string   `json:"confidence"`
}

type ExtractResult struct {
	Candidates   []ExtractCandidate `json:"candidates"`
	ProfileDelta string             `json:"profile_delta,omitempty"`
}

type ExtractionLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

func ExtractMemories(ctx context.Context, llm ExtractionLLM, scope, sessionID, compactionSummary string) (*ExtractResult, error) {
	result, err := extractStage1(ctx, llm, scope, sessionID, compactionSummary)
	if err != nil {
		return nil, err
	}

	// Stage 2: self-questioning pass (ProMem pattern). Best-effort —
	// failures do not invalidate Stage 1 results.
	gapFacts, gapErr := extractStage2SelfQuestion(ctx, llm, compactionSummary, result)
	if gapErr == nil && len(gapFacts) > 0 {
		result.Candidates = append(result.Candidates, gapFacts...)
	}

	return result, nil
}

func extractStage1(ctx context.Context, llm ExtractionLLM, scope, sessionID, compactionSummary string) (*ExtractResult, error) {
	systemPrompt := `你是一个知识提取器。从以下 session 总结中提取值得长期保留的知识。

类型定义：
- "lesson": 具体经验教训（问题→根因→解决方案→泛化教训）
- "topic": 技术主题知识点（架构、模式、约定、API 说明）
- "knowledge_update": 需要更新到核心知识库的事实（用户偏好、项目约束、常驻事实）

范围判断：
- "global": 跨项目通用的知识（语言特性、设计模式、通用工具）
- "project": 本项目特有的知识（项目架构、约定、依赖、bug fix）

返回 JSON 格式：
{
  "candidates": [
    {
      "title": "...",
      "content": "markdown 格式正文...",
      "category": "lesson | topic | knowledge_update",
      "scope": "global | project",
      "tags": ["tag1", "tag2"],
      "confidence": "high | medium | low"
    }
  ],
  "profile_delta": "如果从对话中发现用户偏好/风格变化，写简短摘要"
}

规则：
- 只提取有长期价值的知识，不提取一次性操作
- 置信度 high: 明确的教训或事实；medium: 可能有用的发现；low: 不确定但值得记录
- content 用 markdown，包含必要的上下文、代码片段、链接
- 不要重复已有的知识`

	userMsg := fmt.Sprintf("Session ID: %s\nScope: %s\n\n--- Compaction Summary ---\n%s",
		sessionID, scope, compactionSummary)

	resp, err := llm.Chat(ctx, systemPrompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("extraction llm: %w", err)
	}

	jsonStr := strings.TrimSpace(resp)
	if idx := strings.Index(jsonStr, "```json"); idx >= 0 {
		jsonStr = jsonStr[idx+7:]
		if end := strings.Index(jsonStr, "```"); end >= 0 {
			jsonStr = jsonStr[:end]
		}
	} else if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
	}

	var result ExtractResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse extraction: %w\nresponse: %s", err, resp)
	}
	return &result, nil
}

// extractStage2SelfQuestion asks the LLM to identify knowledge that Stage 1
// missed. Returns additional candidates (may be empty). Errors are non-fatal;
// the caller treats them as "no gaps found".
func extractStage2SelfQuestion(ctx context.Context, llm ExtractionLLM, summary string, stage1 *ExtractResult) ([]ExtractCandidate, error) {
	var extractedSB strings.Builder
	for i, c := range stage1.Candidates {
		fmt.Fprintf(&extractedSB, "%d. [%s] %s\n", i+1, c.Category, c.Title)
	}

	prompt := `You are reviewing a knowledge extraction for gaps. Based on the conversation summary and already-extracted knowledge below, identify important facts or lessons that were MISSED.

Rules:
- Only identify genuinely missing items, not refinements of existing ones
- Focus on actionable knowledge: root causes, patterns, preferences, constraints
- If nothing important was missed, return an empty array

Conversation summary:
` + summary + `

Already extracted:
` + extractedSB.String() + `

Return JSON with the same format as the extraction (candidates array). If nothing is missing, return {"candidates":[]}.`

	resp, err := llm.Chat(ctx, "", prompt)
	if err != nil {
		return nil, err
	}

	jsonStr := strings.TrimSpace(resp)
	if idx := strings.Index(jsonStr, "```json"); idx >= 0 {
		jsonStr = jsonStr[idx+7:]
		if end := strings.Index(jsonStr, "```"); end >= 0 {
			jsonStr = jsonStr[:end]
		}
	} else if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
	}

	var result ExtractResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse stage2: %w", err)
	}
	return result.Candidates, nil
}
