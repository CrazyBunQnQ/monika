package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"monika/internal/tool"
)

type bashTool struct {
	projectDir string
	shell      string
	shellArg   string
}

func NewBash(projectDir string) (tool.Tool, error) {
	shell, shellArg := resolveShell()
	if shell == "" {
		return nil, fmt.Errorf("no shell found on system")
	}
	return &bashTool{
		projectDir: projectDir,
		shell:      shell,
		shellArg:   shellArg,
	}, nil
}

func resolveShell() (string, string) {
	if runtime.GOOS == "windows" {
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
	if path, err := exec.LookPath("sh"); err == nil {
		return path, "-c"
	}
	if path, err := exec.LookPath("bash"); err == nil {
		return path, "-c"
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
Commands timeout after 120 seconds. Output exceeding 30000 characters is truncated.

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
Do NOT use git commands with -i flag (requires interactive input).`
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
		},
		"required": []string{"command"},
	}
}

func (b *bashTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Command string `json:"command"`
		Workdir string `json:"workdir"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	workdir := tool.ProjectDirOrDefault(ctx, b.projectDir)
	if params.Workdir != "" {
		if !filepath.IsAbs(params.Workdir) {
			return tool.ExecutionResult{Content: "workdir must be absolute", IsError: true}, nil
		}
		absProject, err := filepath.Abs(tool.ProjectDirOrDefault(ctx, b.projectDir))
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if real, err := filepath.EvalSymlinks(absProject); err == nil {
			absProject = real
		}
		absWD, err := filepath.Abs(params.Workdir)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if real, err := filepath.EvalSymlinks(absWD); err == nil {
			absWD = real
		}
		rel, err := filepath.Rel(absProject, absWD)
		if err != nil || strings.HasPrefix(rel, "..") {
			return tool.ExecutionResult{Content: "workdir is outside project directory", IsError: true}, nil
		}
		workdir = absWD
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, b.shell, b.shellArg, params.Command)
	cmd.Dir = workdir
	hideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.String()
	if errStderr := stderr.String(); errStderr != "" {
		if out != "" {
			out += "\n"
		}
		out += errStderr
	}
	if out == "" && err != nil {
		out = err.Error()
	}

	isError := err != nil
	out = strings.TrimSpace(out)

	// Truncate very long output, keeping head and tail with a midpoint marker.
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
