package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"monika/internal/tool"
)

type BgManager interface {
	Start(command, workdir string) (string, error)
	Stop(taskID string) error
	Logs(taskID string, lines int) ([]string, error)
}

type bashTool struct {
	projectDir string
	bgMgr      BgManager
	engine     *ShellEngine
}

func NewBash(projectDir string) (tool.Tool, error) {
	return &bashTool{
		projectDir: projectDir,
		engine:     NewShellEngine(),
	}, nil
}

func (b *bashTool) SetBgManager(mgr BgManager) {
	b.bgMgr = mgr
}

// ResolveShell returns the system shell path and argument.
// Kept for informational purposes (e.g. system prompt display).
// The shell engine no longer depends on a system shell, but this
// is still used to display the shell name in the system prompt.
func ResolveShell() (string, string) {
	return resolveShell()
}

func resolveShell() (string, string) {
	if path, err := exec.LookPath("sh"); err == nil {
		return path, "-c"
	}
	if path, err := exec.LookPath("bash"); err == nil {
		return path, "-c"
	}
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path, "-Command"
	}
	if path, err := exec.LookPath("powershell"); err == nil {
		return path, "-Command"
	}
	if path, err := exec.LookPath("cmd"); err == nil {
		return path, "/C"
	}
	return "", ""
}

func (b *bashTool) Name() string { return "bash" }
func (b *bashTool) Description() string {
	return `Execute a shell command. Use bash ONLY when no dedicated tool covers the operation.

IMPORTANT: Do NOT use bash for file operations:
- File search: Use glob (NOT find/ls)
- Content search: Use grep (NOT grep/rg)
- Read files: Use file_read (NOT cat/head/tail)
- Edit files: Use file_edit or patch (NOT sed/awk)
- Write files: Use file_write (NOT echo/cat)
- Communication: Output text directly (NOT echo/printf)

When running multiple independent commands, make multiple bash calls in parallel.
Commands timeout after 120 seconds by default. Use the "timeout" parameter (max 600 seconds) for long-running commands like npm install, docker build, etc. Output exceeding 30000 characters is truncated.

CRITICAL: Execute ONE command per call. Do NOT use &&, ||, ; or command substitution ($()).
If multiple commands are needed, issue separate bash calls.

# Committing changes with git

Only create commits when requested by the user.

Git Safety Protocol:
- NEVER update git config
- NEVER run destructive/irreversible git commands (push --force, reset --hard) unless explicitly requested
- NEVER skip hooks (--no-verify) unless explicitly requested
- NEVER force push to main/master — warn the user if they request it
- Avoid git commit --amend. ONLY use --amend when ALL conditions met:
  (1) User explicitly requested amend, OR commit succeeded but pre-commit hooks auto-modified files
  (2) HEAD commit was created by you in this conversation
  (3) Commit has NOT been pushed to remote
- If commit FAILED or was REJECTED by hook, NEVER amend — fix the issue and create a NEW commit
- If already pushed to remote, NEVER amend unless user explicitly requests it

When creating a commit:
1. Run in parallel: git status, git diff, git log --oneline -10
2. Analyze changes, draft commit message matching repo style
3. Stage files and commit
4. Run git status to verify
Do NOT push unless explicitly asked.
Do NOT use git commands with -i flag (requires interactive input).

# Background Tasks

When a command is expected to run for a long time (dev servers, watchers, file watchers, build watchers, etc.), use action='background' instead of blocking. The command runs in the background and you get a task_id back. You can check logs with action='logs' and stop with action='stop'. Do NOT block waiting for long-running commands.`
}

func (b *bashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to execute",
			},
			"workdir": map[string]any{
				"type":        "string",
				"description": "The working directory. Defaults to the project directory.",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"run", "background", "stop", "logs"},
				"description": "Action mode: 'run' (default, wait for completion), 'background' (run in background, return task_id), 'stop' (stop a background task), 'logs' (get recent logs of a background task).",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Background task ID. Required when action is 'stop' or 'logs'.",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds for the command. Default 120, max 600. Use higher values for long-running commands like npm install, docker build, etc.",
			},
		},
		"required": []string{},
	}
}

func (b *bashTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Command string `json:"command"`
		Workdir string `json:"workdir"`
		Action  string `json:"action"`
		TaskID  string `json:"task_id"`
		Timeout int    `json:"timeout"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	switch params.Action {
	case "stop":
		if b.bgMgr == nil {
			return tool.ExecutionResult{Content: "background tasks not available", IsError: true}, nil
		}
		if params.TaskID == "" {
			return tool.ExecutionResult{Content: "task_id is required for stop action", IsError: true}, nil
		}
		if err := b.bgMgr.Stop(params.TaskID); err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: "stopped task " + params.TaskID}, nil

	case "logs":
		if b.bgMgr == nil {
			return tool.ExecutionResult{Content: "background tasks not available", IsError: true}, nil
		}
		if params.TaskID == "" {
			return tool.ExecutionResult{Content: "task_id is required for logs action", IsError: true}, nil
		}
		lines, err := b.bgMgr.Logs(params.TaskID, 50)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		out := strings.Join(lines, "\n")
		return tool.ExecutionResult{Content: out}, nil

	case "background":
		if b.bgMgr == nil {
			return tool.ExecutionResult{Content: "background tasks not available", IsError: true}, nil
		}
		workdir, err := b.resolveWorkdir(ctx, params.Workdir)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		taskID, err := b.bgMgr.Start(params.Command, workdir)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: "background task started with id: " + taskID}, nil

	default:
		return b.executeRun(ctx, params.Command, params.Workdir, params.Timeout)
	}
}

func (b *bashTool) executeRun(ctx context.Context, command, workdirParam string, timeoutSec int) (tool.ExecutionResult, error) {
	workdir, err := b.resolveWorkdir(ctx, workdirParam)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	const defaultTimeout = 120
	const maxTimeout = 600

	timeout := defaultTimeout
	if timeoutSec > 0 {
		timeout = timeoutSec
	}
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	result := b.engine.Run(timeoutCtx, command, workdir, os.Environ())

	out := result.Stdout
	if result.Stderr != "" {
		if out != "" {
			out += "\n"
		}
		out += result.Stderr
	}
	if out == "" && result.ExitCode != 0 {
		out = fmt.Sprintf("exit code %d", result.ExitCode)
	}

	isError := result.ExitCode != 0
	out = strings.TrimSpace(out)

	const maxOutputChars = 30000
	if len(out) > maxOutputChars {
		headLen := maxOutputChars / 2
		tailLen := maxOutputChars / 2
		head := out[:headLen]
		tail := out[len(out)-tailLen:]
		omitted := len(out) - headLen - tailLen
		out = head + fmt.Sprintf("\n\n... [%d characters truncated] ...\n\n", omitted) + tail
	}

	return tool.ExecutionResult{Content: out, IsError: isError}, nil
}

func (b *bashTool) resolveWorkdir(ctx context.Context, workdirParam string) (string, error) {
	workdir := tool.ProjectDirOrDefault(ctx, b.projectDir)
	if workdirParam == "" {
		return workdir, nil
	}
	if !filepath.IsAbs(workdirParam) {
		return "", fmt.Errorf("workdir must be absolute")
	}
	absProject, err := filepath.Abs(tool.ProjectDirOrDefault(ctx, b.projectDir))
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(absProject); err == nil {
		absProject = real
	}
	absWD, err := filepath.Abs(workdirParam)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(absWD); err == nil {
		absWD = real
	}
	rel, err := filepath.Rel(absProject, absWD)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("workdir is outside project directory")
	}
	return absWD, nil
}
