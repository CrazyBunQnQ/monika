# File Editing Experience Optimization — Interactive Editor

**Date**: 2026-06-13
**Status**: designing

## Overview

Upgrade the PreviewPanel from a read-only file viewer into a fully interactive code editor. The design philosophy: Monika is AI-first, human-second — the editor exists for reviewing AI output and making targeted corrections, not for writing code from scratch.

## Design Principles

1. **Always editable** — open a file, start editing immediately. No view/edit mode toggle.
2. **CodeMirror 6 as foundation** — keep existing tree-sitter highlighting and LSP diagnostics. Extend with CM6's modular extension system rather than replacing with Monaco.
3. **AI conflict is a first-class concern** — when the user has unsaved edits and AI tries to modify the same file, don't silently overwrite or reject. Show a diff and let the user decide.
4. **Reuse existing UI patterns** — conflict resolution uses the existing Preview diff view, not modal dialogs.
5. **Incremental scope** — multi-cursor, auto-completion, snippets, refactoring are intentionally excluded (they serve human-first workflows).

## Architecture

```
PreviewPanel.tsx (modified)
├── EditorView: editable: true  (was: false)
├── + history()                 // undo/redo
├── + standardKeymap            // Ctrl+Z/Y/A/C/V/S/X
├── + bracketMatching()         // highlight matching brackets
├── + closeBrackets()           // auto-close () {} []
├── + indentOnInput()           // auto-indent on Enter
├── + foldGutter()              // tree-sitter driven code folding
├── + search()                  // Ctrl+F search panel
├── + hoverTooltip()            // bridge LSP hover → CM6 tooltip
├── + definitionHandler         // Ctrl+Click / F12 go-to-definition
├── + saveKeybinding            // Ctrl+S → Go WriteFile
└── (existing) tree-sitter highlighting, LSP diagnostics, minimap

lspService.ts (extended)
├── requestDiagnostics()        // existing, stays active in edit mode
├── + requestHover()            // NEW: LSP hover info
└── + requestDefinition()       // NEW: LSP go-to-definition

store/index.ts (extended)
├── + dirtyFiles: Set<string>   // files with unsaved user edits
├── + markFileDirty(path)
├── + markFileClean(path)
└── + handleToolConflict(event) // route conflict to preview diff

Go backend
├── internal/tool/builtin/file_edit.go  // + conflict detection
├── internal/tool/builtin/file_patch.go // + conflict detection
├── internal/tool/builtin/file_write.go // + conflict detection
├── internal/tool/context.go            // + dirty file tracking
├── internal/api/                        // + WriteFile, + SetFileDirty Wails bindings
└── internal/tool/tool.go               // ExecutionResult +Conflict field
```

## Detailed Changes

### 1. PreviewPanel — Editable Core

**File**: `frontend/src/components/Preview/PreviewPanel.tsx`

Remove the Cursor class hack (lines ~479-492) that fakes a "cursor" inside readonly code. Replace with native CM6 editing.

New extensions added to the existing `EditorState.create()`:
- `history()` + `historyKeymap` — undo/redo stack
- `standardKeymap` — includes all standard keybindings
- `bracketMatching()` — bracket pair highlighting
- `closeBrackets()` — auto-close parentheses, braces, brackets
- `indentOnInput()` — auto-indent on newline
- `foldGutter()` — code folding gutter (driven by tree-sitter if available, fallback to indent-based)
- `search()` — `@codemirror/search` with search panel
- `hoverTooltip()` — custom tooltip that calls `lspService.requestHover()`
- Custom keybinding: `Ctrl+S` → call `MonikaApp.WriteFile(path, content)`

Keep existing:
- `syntaxHighlighting(oneDarkHighlightStyle)` — theme
- tree-sitter highlighting extension
- LSP diagnostic decorations
- Minimap

**Save flow**:
1. User presses Ctrl+S
2. EditorView.content → `MonikaApp.WriteFile(filePath, content)`
3. On success → `markFileClean(filePath)`, dirty indicator disappears
4. On error → toast notification

**Dirty indicator**:
- Tab title shows a yellow dot (●) next to filename when unsaved
- Status bar shows "Unsaved" in yellow
- On Ctrl+S success, both clear

### 2. LSP Hover and Go-to-Definition

**File**: `frontend/src/lib/lspService.ts`

Two new methods:
- `requestHover(filePath, line, character): Promise<{contents, range} | null>`
- `requestDefinition(filePath, line, character): Promise<{uri, range} | null>`

**Hover tooltip** (`PreviewPanel.tsx`):
- CM6 `hoverTooltip()` extension calls `lspService.requestHover()` on debounced mouse-hover
- Displays type signature, documentation, source file
- Styled to match Monika's dark theme

**Go-to-definition** (`PreviewPanel.tsx`):
- `F12` or `Ctrl+Click` triggers `lspService.requestDefinition()`
- If same file: scroll to definition line
- If different file: open a new dockview tab and scroll to definition

### 3. AI Edit Conflict Handling

This is the core differentiator from VS Code. The user's manual edits and AI tool edits can target the same file simultaneously.

**Frontend — dirty tracking**:

```
// store/index.ts additions
dirtyFiles: Set<string>   // paths of files with unsaved user edits

markFileDirty(path: string):
  dirtyFiles.add(path)
  MonikaApp.SetFileDirty(path, true)  // notify Go backend

markFileClean(path: string):
  dirtyFiles.delete(path)
  MonikaApp.SetFileDirty(path, false)
```

When `EditorView.content` changes (via `EditorView.updateListener`), if the change is a user-initiated doc change, call `markFileDirty(filePath)`.

**Go backend — conflict detection**:

Add to `internal/tool/context.go`:
```go
// dirtyFiles is a set maintained by SetFileDirty Wails binding
var dirtyFiles sync.Map // map[string]bool

func IsFileDirty(path string) bool {
    v, ok := dirtyFiles.Load(path)
    return ok && v.(bool)
}
```

Add to `file_edit.go` / `file_patch.go` / `file_write.go` `Execute()`:
```go
if IsFileDirty(safePath) {
    diskContent, _ := os.ReadFile(safePath)
    // Compute what AI wants to write
    // Compute what AI wants to write — tool-specific:
    // file_edit: same as existing editFile() logic, applied to diskContent
    // file_patch: same as existing patchFile() logic, applied to diskContent
    // file_write: aiContent = params.Content
    aiContent := toolSpecificApply(diskContent, params)
    diff := computeDiff(safePath, string(diskContent), aiContent)
    return ExecutionResult{
        Content:    "File has unsaved user edits. Showing conflict.",
        Conflict:   true,
        DiskContent: string(diskContent),
        AiContent:   aiContent,
        DiffLines:   diff,
    }, nil
}
```

**Add to `ExecutionResult`** in `tool/tool.go`:
```go
type ExecutionResult struct {
    Content     string
    IsError     bool
    DiffLines   []string
    Conflicts   bool   // merge conflicts in file (existing)
    Conflict    bool   // NEW: user-AI edit conflict
    DiskContent string // NEW: file content on disk (with user edits) 
    AiContent   string // NEW: what AI wants to write
}
```

**Frontend — conflict display**:

The existing `tool_output` handler in store/index.ts already routes `DiffLines` to `setPreviewDiff`. Extend it:

1. Tool result has `conflict: true` and `diffLines` populated
2. Preview panel switches to diff mode showing **Disk version vs AI version**
3. Diff header shows extra buttons: **[Accept AI]** and **[Keep Mine]**
4. **Accept AI**: frontend calls `WriteFile(path, aiContent)`, clears dirtyFiles, refreshes editor view
5. **Keep Mine**: dirtyFiles unchanged, tool returns error to AI, AI retries

This reuses the existing `DiffView` component and Preview panel diff mode — no modal dialog.

### 4. Go Backend New APIs

**WriteFile** (Wails binding):
```
WriteFile(path string, content string) error
```
- Resolves path within project directory (same safety as existing tools)
- Writes content to disk with 0o644 permissions
- Creates parent directories if needed (`os.MkdirAll`)

**SetFileDirty** (Wails binding):
```
SetFileDirty(path string, dirty bool)
```
- Updates the in-memory `dirtyFiles` sync.Map
- Used by frontend to notify backend of dirty state changes
- On frontend disconnect or session end, the session's dirty file entries are cleared via the existing session cleanup path

### 5. Removed Items

### 5. Removed Items

- `editorMode` / `setFileEditable` — not needed; always editable
- Cursor simulation hack in PreviewPanel (the readonly "cursor" span)
- All `@codemirror/lang-*` packages — already planned for removal in 2026-06-07-file-preview-design.md

## Constraint: File Size Limit

Keep the existing `MAX_FILE_LINES = 5000` limit. Files exceeding this remain read-only with a banner: "File too large for editing (5000+ lines). Use external editor."

## What We Are NOT Doing

- Multi-cursor editing
- Snippets
- Auto-completion popup
- Refactoring / rename
- Format on save
- Split editor / multi-pane
- Edit history timeline / blame

These are intentionally excluded as they serve human-first workflows. Monika's AI writes the code; the human editor is for review and targeted corrections.

## File Change Summary

| File | Change |
|------|--------|
| `frontend/src/components/Preview/PreviewPanel.tsx` | Enable editing, add CM6 extensions, hover/go-to-def, save binding, dirty tracking |
| `frontend/src/lib/lspService.ts` | Add `requestHover()`, `requestDefinition()` |
| `frontend/src/store/index.ts` | Add `dirtyFiles`, `markFileDirty`, `markFileClean`, conflict handler |
| `internal/tool/builtin/file_edit.go` | Add conflict detection branch in `Execute()` |
| `internal/tool/builtin/file_patch.go` | Same |
| `internal/tool/builtin/file_write.go` | Same |
| `internal/tool/tool.go` | Add `Conflict`, `DiskContent`, `AiContent` to `ExecutionResult` |
| `internal/tool/context.go` | Add `dirtyFiles` sync.Map, `IsFileDirty()`, new Wails bindings |
| `main.go` or `internal/api/` | Register `WriteFile` and `SetFileDirty` bindings |
