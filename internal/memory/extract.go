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
