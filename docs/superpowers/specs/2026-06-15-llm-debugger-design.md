# LLM-Driven Debugger via Debug Adapter Protocol

**Date**: 2026-06-15
**Status**: designing

## Overview

Add DAP (Debug Adapter Protocol) integration to Monika, enabling LLMs to set breakpoints, step through code, inspect variables, read memory, and control program execution — just like a human using a debugger. The feature consists of three subsystems: a pure-Go DAP backend, an LLM-callable `debug` tool, and a real-time debug UI panel in the frontend.

Reference implementation: oh-my-pi (`packages/coding-agent/src/dap/`, `packages/coding-agent/src/tools/debug.ts`).

## Design Principles

1. **LLM-first, human-second** — the debug workflow is driven by LLM tool calls. The frontend panel visualizes state; it does not lead.
2. **Pure Go DAP** — no Node.js sidecar, no external runtime dependency. DAP is a simple protocol (header-delimited JSON over stdio/socket).
3. **Follow existing patterns** — `internal/dap/` mirrors `internal/lsp/`; debug tool follows builtin tool conventions; frontend follows existing component patterns.
4. **Multi-session with explicit IDs** — LLM manages session IDs explicitly. No implicit session resolution that could confuse context.
5. **Read/exec permission split** — read-only debug actions (stack_trace, variables, etc.) default-allow; execution-modifying actions (launch, step, continue, etc.) go through the permission pipeline.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                     Frontend (React)                  │
│                                                       │
│  Monaco Editor         Debug Side Panel               │
│  (breakpoint glyphs,   (call stack, variables,        │
│   current-line          threads, breakpoints,          │
│   highlight)            watch)                        │
│       │                    │                           │
│       └────────────────────┼───────────────────────────│
│                            │ Wails bindings            │
├────────────────────────────┼───────────────────────────│
│                     Backend (Go)                       │
│                            │                           │
│  ┌─────────────────────────┼──────────────────┐       │
│  │ internal/api/           │                   │       │
│  │  eventbus.go  ← DAP events                  │       │
│  │  debug_api.go ← Frontend debug API          │       │
│  └──────────┬──────────────┼──────────────────┘       │
│             │              │                           │
│  ┌──────────▼──────────────▼──────────────────┐       │
│  │ internal/dap/                               │       │
│  │  manager.go    — global DAP manager (singleton)      │
│  │  session.go    — per-debug-session state    │       │
│  │  client.go     — DAP protocol client        │       │
│  │  types.go      — full DAP type definitions  │       │
│  │  config.go     — adapter discovery & select │       │
│  │  defaults.go   — built-in adapter configs   │       │
│  └──────────┬──────────────────────────────────┘       │
│             │                                           │
│  ┌──────────▼──────────────────────────────────┐       │
│  │ internal/tool/builtin/                       │       │
│  │  debug.go    — LLM debug tool (30 actions)   │       │
│  └──────────────────────────────────────────────┘       │
│             │                                           │
│             ▼                                           │
│     ┌──────────────┐                                    │
│     │ Debug Adapter │ (external: dlv, gdb, debugpy...)  │
│     └──────────────┘                                    │
└──────────────────────────────────────────────────────────┘
```

### Data Flow (LLM-driven)

```
LLM → tool.Execute("debug", args) → DapManager → DapSession → DapClient → debug adapter
                                                                        │
                                                    ┌───────────────────┘
                                                    ▼
                                        DAP events (stopped, output, etc.)
                                                    │
                                    ┌───────────────┴───────────────┐
                                    ▼                               ▼
                              Tool result                      EventBus emit
                              (returned to LLM)                (frontend updates)
```

### Data Flow (optional human click on UI buttons)

```
User clicks step/continue → Wails binding → DapManager → DapSession → DapClient → adapter
```

Frontend buttons are convenience, not primary control. The LLM drives the debugging loop.

## Detailed Changes

### 1. DAP Backend (`internal/dap/`)

#### 1.1 Types (`types.go`)

Full DAP type definitions modeled after the DAP specification and oh-my-pi's type coverage:

- **Protocol messages**: `DapRequestMessage`, `DapResponseMessage`, `DapEventMessage`
- **Session lifecycle**: `DapLaunchArguments`, `DapAttachArguments`, `DapCapabilities`, `DapInitializeArguments`
- **Breakpoints**: `DapSourceBreakpoint`, `DapFunctionBreakpoint`, `DapInstructionBreakpoint`, `DapDataBreakpoint`, and corresponding response/record types
- **State queries**: `DapStackFrame`, `DapScope`, `DapVariable`, `DapThread`
- **Memory & disassembly**: `DapDisassembledInstruction`, `DapReadMemoryResponse`, `DapWriteMemoryArguments`
- **Modules & sources**: `DapModule`, `DapSource`

Internal record types (not DAP spec but used by the session manager):
- `DapSessionSummary` — serializable session state for LLM and frontend
- `DapContinueOutcome` — result of continue/step operations
- `DapBreakpointRecord` — tracked breakpoint state
- `DapAdapterConfig` / `DapResolvedAdapter` — adapter configuration

#### 1.2 Protocol Client (`client.go`)

```
DapClient:
  Spawn(adapter DapResolvedAdapter, cwd string) → *DapClient
  Initialize(args DapInitializeArguments) → DapCapabilities
  SendRequest(command string, args any) → response body
  WaitForEvent(event string) → event body
  OnEvent(event string, handler) → unsubscribe func
  OnReverseRequest(command string, handler)
  Dispose()
  IsAlive() bool
```

- Communication: `Content-Length: <N>\r\n\r\n<JSON>` framing over child process stdin/stdout
- Socket mode support for adapters that require it (e.g., dlv via `--listen=unix:<path>`)
- Message reader loop with buffered parsing, header detection, and content-length extraction
- Request/response matching by `seq`/`request_seq`
- Pending request timeout (default 30s, configurable)
- Reverse request handling (e.g., `runInTerminal` from adapter)

#### 1.3 Session Manager (`session.go`)

```
DapSession:
  // Identity
  id, adapter, cwd, program

  // Lifecycle
  status: launching | configuring | stopped | running | terminated
  initializedSeen, needsConfigurationDone, configurationDoneSent

  // State
  breakpoints: map[filePath][]DapBreakpointRecord
  functionBreakpoints, instructionBreakpoints, dataBreakpoints
  stop: DapStopLocation (threadId, frameId, reason, source, line, column)
  threads: []DapThread
  lastStackFrames: []DapStackFrame

  // Output
  output: string (capped at 128KB), outputTruncated: bool

  // Capabilities
  capabilities: DapCapabilities

  Methods:
  Launch() / Attach()       — spawn adapter, initialize, configure, capture initial stop
  SetBreakpoint() / RemoveBreakpoint()
  Continue() / StepOver() / StepIn() / StepOut() / Pause()
  Evaluate() / StackTrace() / Scopes() / Variables()
  Threads() / Disassemble() / ReadMemory() / WriteMemory()
  Modules() / LoadedSources()
  Terminate()
```

Key behaviors:
- Launch flow: spawn adapter → initialize → (wait for initialized event) → configurationDone → launch request → capture initial stop
- Breakpoint management: deduplication by file+line, sorting by line number, server-side verification
- `Continue`/step returns `DapContinueOutcome` with state: `stopped` | `running` | `terminated`, plus `timedOut` flag
- Output buffer auto-truncates at 128KB
- Idle session cleanup after 10 minutes
- Heartbeat checks adapter liveness every 5 seconds

#### 1.4 Global Manager (`manager.go`)

```
DapManager (singleton):
  sessions: map[string]*DapSession
  activeSessionID: string

  Launch(program, args, adapter, cwd) → DapSessionSummary
  Attach(pid, port, host, adapter, cwd) → DapSessionSummary
  GetSession(id) → *DapSession
  ListSessions() → []DapSessionSummary
  TerminateSession(id)
```

- Only one launch/attach at a time (serialize via mutex)
- LLM explicitly passes session_id for multi-session scenarios
- If session_id is omitted and only one session exists, use that

#### 1.5 Adapter Configuration (`config.go` + `defaults.go`)

Built-in adapter defaults (based on oh-my-pi's `defaults.json`):

| Adapter | Command | Languages | File Types |
|---------|---------|-----------|------------|
| dlv | `dlv dap` | Go | `.go` |
| debugpy | `python -m debugpy.adapter` | Python | `.py` |
| gdb | `gdb --interpreter=dap` | C/C++/Rust | `.c`, `.cpp`, `.rs` |
| lldb-dap | `lldb-dap` | C/C++/Rust/Swift | `.c`, `.cpp`, `.rs`, `.swift` |
| node | `js-debug` | JavaScript/TypeScript | `.js`, `.ts`, `.mjs` |

Auto-selection logic:
1. If LLM specifies `adapter`, use that directly
2. Otherwise, match by file extension
3. Fall back to root markers (`go.mod`, `Cargo.toml`, `package.json`)
4. For extensionless binaries, prefer gdb → lldb-dap

### 2. LLM Debug Tool (`internal/tool/builtin/debug.go`)

Single `debug` tool with `action` discriminator. Follows the existing tool pattern in `internal/tool/builtin/`.

#### Schema (key parameters)

```json
{
  "action": "enum of 26 actions",
  "session_id": "string (optional)",
  "program": "string",
  "args": ["string"],
  "adapter": "string (optional)",
  "cwd": "string (optional)",
  "file": "string",
  "line": "number",
  "function": "string",
  "condition": "string",
  "expression": "string",
  "frame_id": "number",
  "variable_ref": "number",
  "scope_id": "number",
  "memory_reference": "string",
  "count": "number",
  "...": "..."
}
```

#### Actions

**Session (4)**: `launch`, `attach`, `terminate`, `sessions`

**Breakpoints (8)**: `set_breakpoint`, `remove_breakpoint`, `set_function_breakpoint`, `remove_function_breakpoint`, `set_instruction_breakpoint`, `remove_instruction_breakpoint`, `set_data_breakpoint`, `remove_data_breakpoint`, `data_breakpoint_info`

**Execution control (5)**: `continue`, `step_over`, `step_in`, `step_out`, `pause`

**State queries (7)**: `stack_trace`, `threads`, `scopes`, `variables`, `evaluate`, `output`, `modules`, `loaded_sources`

**Low-level (3)**: `disassemble`, `read_memory`, `write_memory`, `custom_request`

#### Permission Model

```
Read actions (default allow):
  stack_trace, threads, scopes, variables, evaluate,
  output, modules, loaded_sources, disassemble, read_memory, sessions

Exec actions (default ask):
  launch, attach, terminate,
  set_breakpoint, remove_breakpoint, set_function_breakpoint, ...,
  continue, step_over, step_in, step_out, pause,
  write_memory, custom_request
```

Implemented via Monika's existing `permission.Pipeline` in `internal/permission/`.

#### Return Format

Human-readable text output, same style as other Monika tools:

```
# set_breakpoint result
Breakpoints for main.go:
- line 42: verified

# continue result
Session debug-1 continued.
State: stopped
Stop reason: breakpoint
Location: main.go:42:5
Frame: main.handleRequest

# stack_trace result
Stack trace:
- #0 main.handleRequest @ main.go:42:5
- #1 main.processQueue @ main.go:87:12
- #2 main.main @ main.go:15:3

# variables result
Variables:
- req = *http.Request{...} (*http.Request)
- err = nil (error)
```

### 3. Frontend Debug UI

#### 3.1 Layout

**Side panel** (right side of editor), containing collapsible accordion sections:

1. **VARIABLES** — tree view of local/global variables, expandable by reference
2. **WATCH** — user/LLM-added watch expressions with live evaluation
3. **CALL STACK** — clickable stack frames that navigate the editor
4. **BREAKPOINTS** — list of all breakpoints with file:line, verified status
5. **THREADS** — thread list with current thread highlighted

**Floating debug toolbar** (top of editor, appears when debug session active):
- Continue (F5) / Step Over (F10) / Step In (F11) / Step Out (Shift+F11)
- Stop (Shift+F5)
- Session label (e.g., "debug-1 · dlv")

#### 3.2 Monaco Integration

- **Breakpoint glyphs**: red circle in glyph margin at breakpoint lines, gray for unverified
- **Current-line highlight**: yellow background on the line where execution is paused
- **Hover**: variable values on hover during debug session (bridge DAP evaluate to Monaco hover provider)

#### 3.3 Components

```
frontend/src/components/debug/
  DebugPanel.tsx           — side panel container
  VariablesView.tsx        — tree view of scoped variables
  WatchView.tsx            — watch expressions
  CallStackView.tsx        — stack frames list
  BreakpointsView.tsx      — breakpoint list
  ThreadsView.tsx          — thread list
  DebugToolbar.tsx         — floating toolbar
  useDebugState.ts         — hook: subscribe to EventBus DAP events
  breakpointDecorations.ts — Monaco decoration helpers
```

### 4. Frontend-Backend Bridge

#### 4.1 EventBus Extensions (`internal/api/eventbus.go`)

New DAP event types:

```
"debug.session.created"     — new session spawned
"debug.session.terminated"  — session ended
"debug.stopped"             — execution paused (breakpoint/step/exception)
"debug.continued"           — execution resumed
"debug.output"              — stdout/stderr captured
"debug.breakpoints.changed" — breakpoints added/removed
```

Event payload includes session summary (id, adapter, status, source, line, column, frameName, stopReason). Frontend subscribes via existing EventBus mechanism.

#### 4.2 Debug API (`internal/api/debug_api.go`)

Wails bindings for frontend-initiated operations:

```
Launch(program, args, adapter, cwd) → sessionSummary
Attach(pid, port, host, adapter, cwd) → sessionSummary
Continue(sessionId) / StepOver(sessionId) / StepIn(sessionId) / StepOut(sessionId)
Pause(sessionId)
Stop(sessionId)
GetState(sessionId) → full debug state
SetBreakpoint(sessionId, file, line, condition)
RemoveBreakpoint(sessionId, file, line)
GetOutput(sessionId)
ListSessions()
```

These are convenience APIs for the optional human-operated UI buttons. The LLM uses the `debug` tool directly.

### 5. Error Handling

| Scenario | Behavior |
|----------|----------|
| Adapter not installed | Tool returns clear install instructions (e.g., `dlv: go install github.com/go-delve/delve/cmd/dlv@latest`) |
| Debuggee crashes | Session marked `terminated`, exit code captured, event emitted |
| Request timeout | 30s default per-request timeout; `continue` timeout returns `{state: "running", timedOut: true}` |
| No active session | Tool returns "No active debug session. Launch or attach first." |
| Breakpoint on invalid line | Adapter returns `verified: false`; tool reports "unverified" to LLM |
| Source file modified | `sourceModified` flag sent on next breakpoint update |
| Output overflow | Buffer capped at 128KB, `outputTruncated: true` when exceeded |
| Adapter process dies | Session auto-marked `terminated` via heartbeat check |

## Scope

This is a single spec covering three subsystems. Implementation order:

1. **DAP backend** (`internal/dap/` + debug tool) — core enabling work, LLM can debug
2. **EventBus + frontend API** — state flows to frontend
3. **Frontend debug panel** — visual UI

Each subsystem is independently testable. Subsystem 1 can ship without 2 and 3 (LLM gets text-only debug output, which is already useful).
