package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/tool"
)

type backgroundTaskTool struct {
	bgMgr BgManager
}

func NewBackgroundTask() (tool.Tool, error) {
	return &backgroundTaskTool{}, nil
}

func (t *backgroundTaskTool) SetBgManager(mgr BgManager) {
	t.bgMgr = mgr
}

func (t *backgroundTaskTool) Name() string { return "background_task" }

func (t *backgroundTaskTool) Description() string {
	return `Manage background tasks started via the bash tool (action='background').

Actions:
- **list**: Show all background tasks with their status (running/stopped/exited), command, and task ID.
- **logs**: Retrieve recent output from a background task. Use the 'lines' parameter to control how many lines to retrieve (default 50).
- **stop**: Stop a running background task by its task_id.

Use 'list' first if you don't know the task ID. Use 'logs' to check on dev servers, watchers, or long-running builds.`
}

func (t *backgroundTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "logs", "stop"},
				"description": "Action to perform: 'list' (show all tasks), 'logs' (get task output), 'stop' (stop a task).",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Background task ID. Required when action is 'logs' or 'stop'.",
			},
			"lines": map[string]any{
				"type":        "integer",
				"description": "Number of log lines to retrieve. Default 50. Only used with action='logs'.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *backgroundTaskTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Action string `json:"action"`
		TaskID string `json:"task_id"`
		Lines  int    `json:"lines"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if t.bgMgr == nil {
		return tool.ExecutionResult{Content: "background tasks not available", IsError: true}, nil
	}

	switch params.Action {
	case "list":
		tasks := t.bgMgr.List()
		if len(tasks) == 0 {
			return tool.ExecutionResult{Content: "No background tasks."}, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Background tasks (%d):\n\n", len(tasks)))
		for _, task := range tasks {
			exitInfo := ""
			if task.Status == BgTaskExited {
				exitInfo = fmt.Sprintf(" (exit %d)", task.ExitCode)
			}
			sb.WriteString(fmt.Sprintf("  %s  [%s%s]  %s\n", task.ID, task.Status, exitInfo, task.Command))
		}
		return tool.ExecutionResult{Content: strings.TrimSpace(sb.String())}, nil

	case "logs":
		if params.TaskID == "" {
			return tool.ExecutionResult{Content: "task_id is required for logs action", IsError: true}, nil
		}
		lines := params.Lines
		if lines <= 0 {
			lines = 50
		}
		logLines, err := t.bgMgr.Logs(params.TaskID, lines)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		out := strings.Join(logLines, "\n")
		return tool.ExecutionResult{Content: out}, nil

	case "stop":
		if params.TaskID == "" {
			return tool.ExecutionResult{Content: "task_id is required for stop action", IsError: true}, nil
		}
		if err := t.bgMgr.Stop(params.TaskID); err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: "stopped task " + params.TaskID}, nil

	default:
		return tool.ExecutionResult{
			Content: fmt.Sprintf("unknown action %q: use 'list', 'logs', or 'stop'", params.Action),
			IsError: true,
		}, nil
	}
}
