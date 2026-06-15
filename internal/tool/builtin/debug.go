package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"monika/internal/dap"
	"monika/internal/tool"
)

// ---------------------------------------------------------------------------
// debugTool
// ---------------------------------------------------------------------------

type debugTool struct {
	manager *dap.DapManager
}

// NewDebugTool creates a new LLM-callable debug tool.
func NewDebugTool(manager *dap.DapManager) tool.Tool {
	return &debugTool{manager: manager}
}

func (d *debugTool) Name() string { return "debug" }

func (d *debugTool) Description() string {
	return `Control debug sessions via the Debug Adapter Protocol (DAP).

WHEN TO DEBUG:
- A bug is suspected at runtime — code inspection alone can't determine the root cause
- You need to verify that the actual execution path matches your analysis
- A variable or return value has unexpected content
- An exception or crash occurs at a specific location

THE DEBUG CYCLE (suggested workflow):
  1. launch → start the program under the debugger
  2. set_breakpoint → place breakpoints at suspicious code paths
  3. continue → let program run until it hits a breakpoint
  4. stack_trace → when stopped, inspect the call stack (levels=N to control depth)
  5. scopes → see which variables are available in current frame
  6. variables → expand variable references to inspect contents
  7. evaluate("expr") → check specific expressions or conditions
  8. step_over / step_in / step_out → navigate through code one step at a time
  9. Repeat 2-8 as needed, adjusting breakpoints and navigation
  10. terminate → clean up when done

TIPS:
- Set multiple breakpoints at once before continuing — saves round trips
- Use condition on breakpoints for precise stopping (e.g. condition: "i > 10")
- After step_over/step_in, check stack_trace to confirm where you landed
- Use evaluate to test hypotheses without editing the code
- If the program exits unexpectedly, check output and the exit code via sessions

ACTIONS:

# Session Management
- launch  — launch a program under the debugger (returns immediately; use finish_launch to complete)
- finish_launch — send configurationDone and wait for launch to finish (call after setting breakpoints)
- attach  — attach to an existing process (by pid or host:port)
- terminate — terminate a debug session
- sessions — list all sessions

# Breakpoints
- set_breakpoint         — set a source breakpoint at file:line (supports condition)
- remove_breakpoint      — remove a source breakpoint at file:line
- set_function_breakpoint   — set a breakpoint on a function name
- remove_function_breakpoint — remove a function breakpoint
- set_instruction_breakpoint — (not yet implemented)
- remove_instruction_breakpoint — (not yet implemented)
- data_breakpoint_info  — (not yet implemented)
- set_data_breakpoint   — (not yet implemented)
- remove_data_breakpoint — (not yet implemented)

# Execution Control
- continue  — resume execution until next breakpoint
- step_over — step over current line
- step_in   — step into function call
- step_out  — step out of current function
- pause     — pause the debuggee

# State Inspection
- stack_trace — fetch stack frames for the stopped thread
- threads     — list all threads
- scopes      — fetch variable scopes for a stack frame
- variables   — fetch children of a variable reference
- evaluate    — evaluate an expression in a stack frame context
- output      — get accumulated output from the debuggee
- modules     — (not yet implemented)
- loaded_sources — (not yet implemented)

# Low-level
- disassemble    — (not yet implemented)
- read_memory    — (not yet implemented)
- write_memory   — (not yet implemented)
- custom_request — send an arbitrary DAP request (not yet implemented)`
}

func (d *debugTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"launch", "attach", "terminate", "sessions",
					"set_breakpoint", "remove_breakpoint",
					"set_function_breakpoint", "remove_function_breakpoint",
					"set_instruction_breakpoint", "remove_instruction_breakpoint",
					"data_breakpoint_info", "set_data_breakpoint", "remove_data_breakpoint",
					"continue", "step_over", "step_in", "step_out", "pause",
					"stack_trace", "threads", "scopes", "variables",
					"evaluate", "output", "modules", "loaded_sources",
					"disassemble", "read_memory", "write_memory", "custom_request",
				},
				"description": "The debug action to perform",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session ID (optional; uses active session if empty)",
			},
			"program": map[string]any{
				"type":        "string",
				"description": "Program path (for launch)",
			},
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Program arguments (for launch)",
			},
			"adapter": map[string]any{
				"type":        "string",
				"description": "Debug adapter name/type (e.g. 'dlv', 'node', 'python')",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory for the debug session",
			},
			"file": map[string]any{
				"type":        "string",
				"description": "Source file path (for breakpoints)",
			},
			"line": map[string]any{
				"type":        "number",
				"description": "Line number (for breakpoints)",
			},
			"function": map[string]any{
				"type":        "string",
				"description": "Function name (for function breakpoints)",
			},
			"condition": map[string]any{
				"type":        "string",
				"description": "Breakpoint condition expression",
			},
			"expression": map[string]any{
				"type":        "string",
				"description": "Expression to evaluate",
			},
			"context": map[string]any{
				"type":        "string",
				"enum":        []string{"watch", "repl", "hover", "variables", "clipboard"},
				"description": "Evaluation context hint",
			},
			"frame_id": map[string]any{
				"type":        "number",
				"description": "Stack frame ID (for scopes, evaluate)",
			},
			"variable_ref": map[string]any{
				"type":        "number",
				"description": "Variable reference ID (for variables request)",
			},
			"scope_id": map[string]any{
				"type":        "number",
				"description": "Scope ID (unused; use frame_id for scopes)",
			},
			"levels": map[string]any{
				"type":        "number",
				"description": "Number of stack frames to fetch (default 10)",
			},
			"memory_reference": map[string]any{
				"type":        "string",
				"description": "Memory reference for read/write/disassemble",
			},
			"count": map[string]any{
				"type":        "number",
				"description": "Byte count or instruction count",
			},
			"pid": map[string]any{
				"type":        "number",
				"description": "Process ID (for attach)",
			},
			"port": map[string]any{
				"type":        "number",
				"description": "Port number (for attach)",
			},
			"host": map[string]any{
				"type":        "string",
				"description": "Host address (for attach)",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default 30)",
			},
		},
		"required": []string{"action"},
	}
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

func (d *debugTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Action          string   `json:"action"`
		SessionID       string   `json:"session_id"`
		Program         string   `json:"program"`
		Args            []string `json:"args"`
		Adapter         string   `json:"adapter"`
		Cwd             string   `json:"cwd"`
		File            string   `json:"file"`
		Line            int      `json:"line"`
		Function        string   `json:"function"`
		Condition       string   `json:"condition"`
		Expression      string   `json:"expression"`
		Context         string   `json:"context"`
		FrameID         int      `json:"frame_id"`
		VariableRef     int      `json:"variable_ref"`
		ScopeID         int      `json:"scope_id"`
		Levels          int      `json:"levels"`
		MemoryReference string   `json:"memory_reference"`
		Count           int      `json:"count"`
		PID             int      `json:"pid"`
		Port            int      `json:"port"`
		Host            string   `json:"host"`
		Timeout         float64  `json:"timeout"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if d.manager == nil {
		return tool.ExecutionResult{Content: "debug manager not available", IsError: true}, nil
	}

	timeout := durationOrDefault(params.Timeout, 30*time.Second)

	switch params.Action {
	// ---- Session ----
	case "launch":
		return d.launch(ctx, params, timeout)
	case "attach":
		return d.attach(params, timeout)
	case "finish_launch":
		return d.finishLaunch(params, timeout)
	case "terminate":
		return d.terminateSession(params)
	case "sessions":
		return d.listSessions()

	// ---- Breakpoints ----
	case "set_breakpoint":
		return d.setBreakpoint(ctx, params, timeout)
	case "remove_breakpoint":
		return d.removeBreakpoint(ctx, params, timeout)
	case "set_function_breakpoint":
		return d.setFunctionBreakpoint(params, timeout)
	case "remove_function_breakpoint":
		return d.removeFunctionBreakpoint(params, timeout)
	case "set_instruction_breakpoint":
		return notImplemented("set_instruction_breakpoint")
	case "remove_instruction_breakpoint":
		return notImplemented("remove_instruction_breakpoint")
	case "data_breakpoint_info":
		return notImplemented("data_breakpoint_info")
	case "set_data_breakpoint":
		return notImplemented("set_data_breakpoint")
	case "remove_data_breakpoint":
		return notImplemented("remove_data_breakpoint")

	// ---- Execution ----
	case "continue":
		return d.continueExec(params, timeout)
	case "step_over":
		return d.stepOver(params, timeout)
	case "step_in":
		return d.stepIn(params, timeout)
	case "step_out":
		return d.stepOut(params, timeout)
	case "pause":
		return d.pause(params, timeout)

	// ---- State ----
	case "stack_trace":
		return d.stackTrace(params, timeout)
	case "threads":
		return d.threads(params, timeout)
	case "scopes":
		return d.scopes(params, timeout)
	case "variables":
		return d.variables(params, timeout)
	case "evaluate":
		return d.evaluate(params, timeout)
	case "output":
		return d.output(params)
	case "modules":
		return notImplemented("modules")
	case "loaded_sources":
		return notImplemented("loaded_sources")

	// ---- Low-level ----
	case "disassemble":
		return notImplemented("disassemble")
	case "read_memory":
		return notImplemented("read_memory")
	case "write_memory":
		return notImplemented("write_memory")
	case "custom_request":
		return notImplemented("custom_request")

	default:
		return tool.ExecutionResult{
			Content: fmt.Sprintf("unknown debug action %q", params.Action),
			IsError: true,
		}, nil
	}
}

// ---------------------------------------------------------------------------
// Session helpers
// ---------------------------------------------------------------------------

func (d *debugTool) getSession(id string) *dap.DapSession {
	if id != "" {
		return d.manager.GetSession(id)
	}
	return d.manager.ActiveSession()
}

func resolvePath(file string, ctx context.Context) string {
	if filepath.IsAbs(file) {
		return file
	}
	dir := tool.ProjectDirFromContext(ctx)
	if dir != "" {
		return filepath.Join(dir, file)
	}
	return file
}

func durationOrDefault(seconds float64, def time.Duration) time.Duration {
	if seconds > 0 {
		return time.Duration(seconds * float64(time.Second))
	}
	return def
}

func notImplemented(action string) (tool.ExecutionResult, error) {
	return tool.ExecutionResult{
		Content: fmt.Sprintf("Action %q not yet implemented in this version", action),
	}, nil
}

// ---------------------------------------------------------------------------
// Session actions
// ---------------------------------------------------------------------------

func (d *debugTool) launch(ctx context.Context, params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	if params.Program == "" {
		return tool.ExecutionResult{Content: "program is required for launch", IsError: true}, nil
	}

	cwd := params.Cwd
	if cwd == "" {
		cwd = tool.ProjectDirFromContext(ctx)
	}

	summary, err := d.manager.Launch(params.Program, params.Args, params.Adapter, cwd)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("launch failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatSessionSummary(summary)}, nil
}

func (d *debugTool) finishLaunch(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active debug session", IsError: true}, nil
	}
	if err := session.CompleteLaunch(timeout); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("finish launch failed: %v", err), IsError: true}, nil
	}
	s := session.Summary()
	return tool.ExecutionResult{Content: fmt.Sprintf("Launch completed. Status: %s\n%s", s.Status, formatSessionSummary(&s))}, nil
}

func (d *debugTool) attach(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	host := params.Host
	if host == "" {
		host = "127.0.0.1"
	}

	cwd := params.Cwd
	summary, err := d.manager.Attach(params.PID, params.Port, host, params.Adapter, cwd)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("attach failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatSessionSummary(summary)}, nil
}

func (d *debugTool) terminateSession(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	id := session.Summary().ID
	d.manager.TerminateSession(id)
	return tool.ExecutionResult{Content: fmt.Sprintf("session %q terminated", id)}, nil
}

func (d *debugTool) listSessions() (tool.ExecutionResult, error) {
	summaries := d.manager.ListSessions()
	if len(summaries) == 0 {
		return tool.ExecutionResult{Content: "no debug sessions"}, nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Debug sessions (%d):\n", len(summaries)))
	for _, s := range summaries {
		b.WriteString(formatSessionSummary(&s))
		b.WriteByte('\n')
	}
	return tool.ExecutionResult{Content: b.String()}, nil
}

// ---------------------------------------------------------------------------
// Breakpoint actions
// ---------------------------------------------------------------------------

func (d *debugTool) setBreakpoint(ctx context.Context, params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.File == "" {
		return tool.ExecutionResult{Content: "file is required for set_breakpoint", IsError: true}, nil
	}
	if params.Line <= 0 {
		return tool.ExecutionResult{Content: "line is required for set_breakpoint", IsError: true}, nil
	}

	file := resolvePath(params.File, ctx)
	records, err := session.SetBreakpoint(file, params.Line, params.Condition, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("set breakpoint failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatBreakpoints(file, records)}, nil
}

func (d *debugTool) removeBreakpoint(ctx context.Context, params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.File == "" {
		return tool.ExecutionResult{Content: "file is required for remove_breakpoint", IsError: true}, nil
	}
	if params.Line <= 0 {
		return tool.ExecutionResult{Content: "line is required for remove_breakpoint", IsError: true}, nil
	}

	file := resolvePath(params.File, ctx)
	records, err := session.RemoveBreakpoint(file, params.Line, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("remove breakpoint failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatBreakpoints(file, records)}, nil
}

func (d *debugTool) setFunctionBreakpoint(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.Function == "" {
		return tool.ExecutionResult{Content: "function is required for set_function_breakpoint", IsError: true}, nil
	}

	records, err := session.SetFunctionBreakpoint(params.Function, params.Condition, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("set function breakpoint failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatFunctionBreakpoints(records)}, nil
}

func (d *debugTool) removeFunctionBreakpoint(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.Function == "" {
		return tool.ExecutionResult{Content: "function is required for remove_function_breakpoint", IsError: true}, nil
	}

	records, err := session.RemoveFunctionBreakpoint(params.Function, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("remove function breakpoint failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatFunctionBreakpoints(records)}, nil
}

// ---------------------------------------------------------------------------
// Execution actions
// ---------------------------------------------------------------------------

func (d *debugTool) continueExec(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	outcome, err := session.Continue(timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("continue failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatContinueOutcome(outcome)}, nil
}

func (d *debugTool) stepOver(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	outcome, err := session.StepOver(timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("step_over failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatContinueOutcome(outcome)}, nil
}

func (d *debugTool) stepIn(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	outcome, err := session.StepIn(timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("step_in failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatContinueOutcome(outcome)}, nil
}

func (d *debugTool) stepOut(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	outcome, err := session.StepOut(timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("step_out failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatContinueOutcome(outcome)}, nil
}

func (d *debugTool) pause(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	summary, err := session.Pause(timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("pause failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatSessionSummary(summary)}, nil
}

// ---------------------------------------------------------------------------
// State actions
// ---------------------------------------------------------------------------

func (d *debugTool) stackTrace(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	levels := params.Levels
	if levels <= 0 {
		levels = 10
	}

	frames, err := session.StackTrace(levels, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("stack_trace failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatStackFrames(frames)}, nil
}

func (d *debugTool) threads(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	threadList, err := session.Threads(timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("threads failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatThreads(threadList)}, nil
}

func (d *debugTool) scopes(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.FrameID <= 0 {
		return tool.ExecutionResult{Content: "frame_id is required for scopes", IsError: true}, nil
	}

	scopeList, err := session.Scopes(params.FrameID, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("scopes failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatScopes(scopeList)}, nil
}

func (d *debugTool) variables(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.VariableRef <= 0 {
		return tool.ExecutionResult{Content: "variable_ref is required for variables", IsError: true}, nil
	}

	vars, err := session.Variables(params.VariableRef, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("variables failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatVariables(vars)}, nil
}

func (d *debugTool) evaluate(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}, timeout time.Duration) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	if params.Expression == "" {
		return tool.ExecutionResult{Content: "expression is required for evaluate", IsError: true}, nil
	}

	ctx := params.Context
	if ctx == "" {
		ctx = "repl"
	}

	eval, err := session.Evaluate(params.Expression, ctx, params.FrameID, timeout)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("evaluate failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: formatEvaluate(eval)}, nil
}

func (d *debugTool) output(params struct {
	Action          string   `json:"action"`
	SessionID       string   `json:"session_id"`
	Program         string   `json:"program"`
	Args            []string `json:"args"`
	Adapter         string   `json:"adapter"`
	Cwd             string   `json:"cwd"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Function        string   `json:"function"`
	Condition       string   `json:"condition"`
	Expression      string   `json:"expression"`
	Context         string   `json:"context"`
	FrameID         int      `json:"frame_id"`
	VariableRef     int      `json:"variable_ref"`
	ScopeID         int      `json:"scope_id"`
	Levels          int      `json:"levels"`
	MemoryReference string   `json:"memory_reference"`
	Count           int      `json:"count"`
	PID             int      `json:"pid"`
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	Timeout         float64  `json:"timeout"`
}) (tool.ExecutionResult, error) {
	session := d.getSession(params.SessionID)
	if session == nil {
		return tool.ExecutionResult{Content: "no active session found", IsError: true}, nil
	}

	out := session.GetOutput()
	if out == "" {
		return tool.ExecutionResult{Content: "(no output)"}, nil
	}

	return tool.ExecutionResult{Content: out}, nil
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

func formatSessionSummary(s *dap.DapSessionSummary) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Session: %s  Status: %s", s.ID, s.Status))
	if s.Adapter != "" {
		b.WriteString(fmt.Sprintf("  Adapter: %s", s.Adapter))
	}
	if s.ProcessName != "" {
		b.WriteString(fmt.Sprintf("  Program: %s", s.ProcessName))
	}
	if s.StopLocation != nil {
		loc := s.StopLocation
		b.WriteString(fmt.Sprintf("  Stopped: reason=%s", loc.Reason))
		if loc.Source != nil && loc.Source.Path != "" {
			b.WriteString(fmt.Sprintf("  at %s:%d", loc.Source.Path, loc.Line))
		} else if loc.Line > 0 {
			b.WriteString(fmt.Sprintf("  at line %d", loc.Line))
		}
		if loc.Text != "" {
			b.WriteString(fmt.Sprintf(" (%s)", loc.Text))
		}
	}
	if s.HasExited {
		b.WriteString(" [exited]")
	}
	return b.String()
}

func formatBreakpoints(file string, breakpoints []dap.DapBreakpointRecord) string {
	if len(breakpoints) == 0 {
		return fmt.Sprintf("No breakpoints in %s", file)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Breakpoints in %s (%d):\n", file, len(breakpoints)))
	for _, bp := range breakpoints {
		line := bp.Breakpoint.Line
		if bp.ActualLine > 0 {
			line = bp.ActualLine
		}
		b.WriteString(fmt.Sprintf("  #%d  line %d", bp.ID, line))
		if !bp.Verified {
			b.WriteString(" [unverified]")
		}
		if bp.Breakpoint.Condition != "" {
			b.WriteString(fmt.Sprintf("  if %s", bp.Breakpoint.Condition))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatFunctionBreakpoints(bps []dap.DapFunctionBreakpointRecord) string {
	if len(bps) == 0 {
		return "No function breakpoints"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Function breakpoints (%d):\n", len(bps)))
	for _, fb := range bps {
		b.WriteString(fmt.Sprintf("  #%d  %s", fb.ID, fb.Breakpoint.Name))
		if !fb.Verified {
			b.WriteString(" [unverified]")
		}
		if fb.Breakpoint.Condition != "" {
			b.WriteString(fmt.Sprintf("  if %s", fb.Breakpoint.Condition))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatContinueOutcome(outcome *dap.DapContinueOutcome) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("State: %s", outcome.State))
	if outcome.TimedOut {
		b.WriteString(" (timed out)")
	}
	b.WriteString("\n")
	b.WriteString(formatSessionSummary(&outcome.Snapshot))
	return b.String()
}

func formatStackFrames(frames []dap.DapStackFrame) string {
	if len(frames) == 0 {
		return "No stack frames"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Stack frames (%d):\n", len(frames)))
	for i, f := range frames {
		b.WriteString(fmt.Sprintf("  #%d  0x%x  %s", i, f.Id, f.Name))
		if f.Source != nil {
			path := f.Source.Path
			if path == "" {
				path = f.Source.Name
			}
			if path != "" {
				b.WriteString(fmt.Sprintf(" at %s:%d", path, f.Line))
			} else {
				b.WriteString(fmt.Sprintf(" at line %d", f.Line))
			}
		} else if f.Line > 0 {
			b.WriteString(fmt.Sprintf(" at line %d", f.Line))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatThreads(threads []dap.DapThread) string {
	if len(threads) == 0 {
		return "No threads"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Threads (%d):\n", len(threads)))
	for _, t := range threads {
		b.WriteString(fmt.Sprintf("  #%d  %s\n", t.Id, t.Name))
	}
	return b.String()
}

func formatScopes(scopes []dap.DapScope) string {
	if len(scopes) == 0 {
		return "No scopes"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Scopes (%d):\n", len(scopes)))
	for _, s := range scopes {
		b.WriteString(fmt.Sprintf("  %s (vars: %d, expensive: %v)\n", s.Name, s.VariablesReference, s.Expensive))
	}
	return b.String()
}

func formatVariables(vars []dap.DapVariable) string {
	if len(vars) == 0 {
		return "No variables"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Variables (%d):\n", len(vars)))
	for _, v := range vars {
		b.WriteString(fmt.Sprintf("  %s = %s", v.Name, v.Value))
		if v.Type != "" {
			b.WriteString(fmt.Sprintf("  (%s)", v.Type))
		}
		if v.VariablesReference > 0 {
			b.WriteString(fmt.Sprintf("  [ref: %d]", v.VariablesReference))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatEvaluate(eval *dap.DapEvaluateResponse) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Result: %s", eval.Result))
	if eval.Type != "" {
		b.WriteString(fmt.Sprintf("  Type: %s", eval.Type))
	}
	if eval.VariablesReference > 0 {
		b.WriteString(fmt.Sprintf("  [ref: %d]", eval.VariablesReference))
	}
	b.WriteByte('\n')
	return b.String()
}

// Ensure strconv import isn't flagged as unused (used via int-to-string conversions).
var _ = strconv.Itoa
