# LSP File Preview Design

Date: 2026-06-07

## Goal

Add full Language Server Protocol capabilities to the file preview panel, enabling users to navigate, understand, and edit code with IDE-like features: go to definition, find references, hover info, document symbols, diagnostics, rename, and code actions.

## Architecture

```
Frontend (CodeMirror 6)          Go Backend (Wails IPC)          LSP Server
+---------------------+     +-------------------------+     +-----------+
| EditorView          |     | App struct              |     | gopls     |
|  +- LspExtension    |---->|  +- LspGoToDefinition() |---->| tsserver  |
|  +- HoverTooltip    |     |  +- LspReferences()     |<----| rust-analyzer
|  +- DiagDecorations |<----|  +- LspHover()          |     | ...       |
|  +- SymbolSidebar   |     |  +- LspDocumentSymbols()|     +-----------+
|  +- Breadcrumbs     |     |  +- LspRename()         |
|                     |     |  +- LspCodeActions()     |
| Right-click Menu    |     |  +- LspDiagnostics()    |
| Ctrl+Click          |     |  +- LspApplyEdit()      |
+---------------------+     |                         |
                            | FileService             |
                            |  +- ReadFile()          |
                            |  +- WriteFile()         |
                            +-------------------------+
```

Reuse existing `internal/lsp/Manager`. Add LSP IPC methods on `App` struct. User edits write directly to disk via `App.WriteFile()` and notify LSP via `App.LspDidChange()`. No coupling with AI agent's tool pipeline.

## Backend: Wails IPC Methods

Add to `App` struct (`internal/api/app.go`), following existing `ReadFile`/`WriteFile` patterns:

```go
// File lifecycle
LspOpenFile(path string) error
LspCloseFile(path string) error
LspDidChange(path string, content string, version int) error

// Read operations
LspGoToDefinition(path string, line int, col int) ([]Location, error)
LspTypeDefinition(path string, line int, col int) ([]Location, error)
LspImplementation(path string, line int, col int) ([]Location, error)
LspReferences(path string, line int, col int) ([]Location, error)
LspHover(path string, line int, col int) (*HoverResult, error)
LspDocumentSymbols(path string) ([]Symbol, error)

// Write operations
LspRename(path string, line int, col int, newName string) (*WorkspaceEdit, error)
LspCodeActions(path string, line int, col int) ([]CodeAction, error)
LspExecuteCodeAction(action CodeAction) (*WorkspaceEdit, error)

// Diagnostics
LspDiagnostics(path string) ([]Diagnostic, error)
```

Types defined in `internal/api/types.go`, re-exporting from `internal/lsp/protocol.go` with Wails-compatible json tags. `Location` uses `{path, line, col}` format; frontend never handles raw LSP URIs.

## Frontend: CodeMirror Upgrade

### Editable Mode

Current: `EditorView.editable.of(false)` (read-only).
New: default remains read-only; toolbar button or typing activates edit mode.

### Edit Sync Flow

```
User types -> CodeMirror onChange
  -> debounce 300ms
  -> App.WriteFile(path, content)
  -> App.LspDidChange(path, content, version++)
```

Frontend maintains version counter, incremented on each change.

### File Lifecycle

- Open preview -> `App.LspOpenFile(path)`
- Close preview / switch file -> `App.LspCloseFile(path)`
- Switch file: close old, then open new

## Read Operations

### Go to Definition (Ctrl+Click + Right-click Menu)

```
Ctrl+Click (line, col)
  -> App.LspGoToDefinition(path, line, col) -> []Location
    -> Single result: jump directly
       Same file: EditorView.dispatch({ scrollIntoView })
       Cross file: store.setPreviewFile(targetPath)
    -> Multiple results: Peek Panel at click position
```

Right-click menu: "Go to Definition", "Go to Type Definition", "Go to Implementation".

### Find References

```
Right-click -> "Find All References"
  -> App.LspReferences(path, line, col)
  -> Peek Panel below editor
    -> Reference list (filename:line + code snippet)
    -> Click entry -> jump to location
```

### Hover Tooltip

```
Mouse hover 500ms on token
  -> App.LspHover(path, line, col) -> HoverResult { contents, range }
  -> CodeMirror tooltip extension renders Markdown
  -> Code blocks highlighted (reuse highlight.js or tree-sitter)
  -> Symbol links in tooltip are clickable
```

Implemented via `@codemirror/tooltip`, compatible with existing `oneDarkHighlightStyle`.

## Document Symbols: Breadcrumbs + Sidebar

### Data Source

```
File open -> App.LspDocumentSymbols(path)
  -> Tree of Symbol[]: { name, kind, range, children[] }
  -> Frontend caches, re-fetches on edit (debounced)
```

### Breadcrumbs (Bottom Status Bar, Merged)

```
+--------------------------------------------------------------+
| src/components/App.tsx                                        |  file path
+--------------------------------------------------------------+
| 1  function handleSave() {                                    |
| 2    const input = validateInput();                           |  editor
+--------------------------------------------------------------+
| MyComponent > handleSave > validateInput  | Ln 5, Col 12 | TS |  breadcrumbs + status
+--------------------------------------------------------------+
```

Single line: breadcrumbs on left, cursor position + language on right. Breadcrumbs truncate from left when space is insufficient. Click breadcrumb level -> dropdown showing sibling symbols -> click to jump.

Symbol kind icons: function f, class diamond, variable x, interface triangle.

### Symbol Sidebar (Collapsible, Right Side)

```
+----------+------------------------------+
|          |  file path                    |
| > types  |------------------------------|
|   IFoo   |  editor                      |
| v funcs  |                              |
|   handle |                              |
|   valid  |                              |
| > vars   |                              |
+----------+------------------------------+
```

Tree rendering matching `Symbol.children` recursion. Click symbol -> scroll to range and highlight. Current cursor symbol highlighted. Toggle via toolbar button or shortcut.

## Diagnostics

### Acquisition

- File open -> `App.LspDiagnostics(path)` initial pull
- After edit (debounced) -> `App.LspDiagnostics(path)` refresh
- Backend push: `Manager` receives LSP diagnostics -> `EventBus` emits `"lsp_diagnostics"` -> frontend `OnEvent` listener updates in real-time

### Rendering

```
Inline decorations:
  Error:   red wavy underline
  Warning: yellow wavy underline
  Info:    blue dashed underline
  Hint:    gray dashed underline

Line gutter marks:
  Error line: red dot
  Warning line: yellow dot

Hover on diagnostic:
  Show diagnostic message first, then LSP hover below
```

Implemented via CodeMirror `Decoration.widget` + `Decoration.line`.

### Diagnostics + Code Actions

```
Cursor on diagnostic -> right-click "Quick Fix..."
  -> App.LspCodeActions(path, line, col)
  -> Show available actions
  -> Select -> App.LspExecuteCodeAction(action)
  -> WorkspaceEdit returned -> apply and save
```

## Write Operations

### Rename

```
Right-click symbol -> "Rename Symbol"
  -> Inline input replaces symbol text in editor
  -> Enter confirms, Esc cancels
  -> App.LspRename(path, line, col, newName) -> WorkspaceEdit
  -> Apply edits: current file via CodeMirror dispatch, other files via ReadFile + apply + WriteFile
```

### Code Actions (General)

```
Right-click -> "Code Actions..."
  -> App.LspCodeActions(path, line, col) -> CodeAction[]
  -> List grouped by kind:
      quickfix -- Quick Fix
      refactor -- Refactor
      source   -- Source (organize imports, etc.)
  -> Select -> App.LspExecuteCodeAction(action)
  -> WorkspaceEdit -> apply
```

### WorkspaceEdit Application

Unified `applyWorkspaceEdit(edit)` function:
1. Collect all affected files
2. Current file: CodeMirror `dispatch` directly
3. Other files: `App.ReadFile` -> apply TextEdit -> `App.WriteFile`
4. Trigger `LspDidChange` for all changed files
5. Emit `file_changed` event for file tree refresh

## Edge Cases

### LSP Server Unavailable

```
Open file -> App.LspOpenFile(path)
  -> Manager checks for running LSP server for language
  -> Not found -> attempt launch via defaults.go config
    -> Launch fails -> return error, frontend degrades to plain editor
    -> No LSP options in right-click menu / Ctrl+Click
    -> Status bar shows "LSP: N/A" instead of language name
```

### Large Files

```
Existing: >5000 lines truncated for preview
LSP operates on full files -> truncated files skip LspOpenFile
-> Large files: no LSP features, status bar shows "File too large for LSP"
```

### AI Agent Concurrent Edits

```
AI agent edits file -> file_changed event
Frontend receives event -> check if current preview file
  -> Yes -> prompt "File modified by AI, refresh?"
    -> Refresh: re-ReadFile + LspOpenFile
    -> Skip: keep current content (user continues editing)
  -> First save wins, later operation overwrites
```

No complex conflict resolution. Simple last-write-wins.

### Navigation History

```
Definition/reference jumps maintain a navigation stack:
  [
    { path: "App.tsx", line: 10, col: 5 },
    { path: "utils.ts", line: 42, col: 0 },  <- current
  ]
Alt+Left: go back, Alt+Right: go forward
```
