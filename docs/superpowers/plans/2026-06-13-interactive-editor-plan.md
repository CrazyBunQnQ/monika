# Interactive Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade PreviewPanel from read-only file viewer to fully interactive code editor with LSP hover/definition, undo/redo, search, code folding, and AI-edit conflict handling.

**Architecture:** CodeMirror 6 remains the editor foundation. Enable `editable`, add 8 CM6 extensions (history, search, bracket matching, closeBrackets, indentOnInput, foldGutter, hoverTooltip, custom save keybinding). Extend lspService with hover/definition. Track unsaved user edits in store via `dirtyFiles: Set<string>`. Go backend detects user-AI edit conflicts in file_edit/file_patch/file_write and returns `conflict: true` with diff — frontend renders in existing Preview diff view with Accept/Keep buttons.

**Tech Stack:** CodeMirror 6, tree-sitter, Wails (Go ↔ TypeScript bindings), Zustand store, React

---

## File Map

| File | Role | Action |
|------|------|--------|
| `internal/agent/event.go:32` | `ToolEvent` struct | Add conflict fields |
| `internal/tool/tool.go:16` | `ExecutionResult` struct | Add Conflict/DiskContent/AiContent |
| `internal/tool/context.go` | Dirty file tracking + IsFileDirty | Add sync.Map, binding |
| `internal/agent/agent_loop.go:1078` | Map ExecutionResult → ToolEvent | Add new field mappings |
| `internal/tool/builtin/file_edit.go` | Conflict detection in Execute | Add check before write |
| `internal/tool/builtin/file_patch.go` | Conflict detection in Execute | Add check before write |
| `internal/tool/builtin/file_write.go` | Conflict detection in Execute | Add check before write |
| `internal/api/app.go` | SetFileDirty Wails binding | Add new method |
| `frontend/package.json` | Add @codemirror/search dep | Add dependency |
| `frontend/src/lib/lspService.ts` | Add hover/definition | Add 2 methods |
| `frontend/src/store/index.ts` | dirtyFiles, conflict handler | Add state + actions |
| `frontend/src/components/Preview/PreviewPanel.tsx` | Enable editing, extensions, save, dirty tracking | Major changes |
| `frontend/bindings/` | Auto-generated from Go changes | Regenerate by Wails |

---

### Task 1: Add conflict fields to Go data structures

**Files:**
- Modify: `internal/tool/tool.go:16-21`
- Modify: `internal/agent/event.go:32-39`

- [ ] **Step 1: Add Conflict/DiskContent/AiContent to ExecutionResult**

Replace `ExecutionResult` in `internal/tool/tool.go`:

```go
type ExecutionResult struct {
	Content     string
	IsError     bool
	DiffLines   []string
	Conflicts   bool   // merge conflict markers in file (existing)
	Conflict    bool   // NEW: user has unsaved edits — AI edit blocked
	DiskContent string `json:"diskContent,omitempty"` // NEW: file on disk (with user edits)
	AiContent   string `json:"aiContent,omitempty"`   // NEW: what AI wants to write
}
```

- [ ] **Step 2: Add Conflict/DiskContent/AiContent to ToolEvent**

Replace `ToolEvent` in `internal/agent/event.go`:

```go
type ToolEvent struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Input       string   `json:"input"`
	Output      string   `json:"output"`
	Status      string   `json:"status"`
	DiffLines   []string `json:"diffLines,omitempty"`
	Conflict    bool     `json:"conflict,omitempty"`    // NEW
	DiskContent string   `json:"diskContent,omitempty"` // NEW
	AiContent   string   `json:"aiContent,omitempty"`   // NEW
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd d:/git/monika && go build ./...`
Expected: PASS (no errors)

- [ ] **Step 4: Commit**

```bash
git add internal/tool/tool.go internal/agent/event.go
git commit -m "feat: add conflict/DiskContent/AiContent to ExecutionResult and ToolEvent"
```

---

### Task 2: Add dirty file tracking in Go backend

**Files:**
- Modify: `internal/tool/context.go`

- [ ] **Step 1: Add dirtyFiles sync.Map and IsFileDirty**

Add to `internal/tool/context.go` after existing context key definitions:

```go
import "sync"

// dirtyFiles tracks which files have unsaved user edits.
// Key: absolute file path, Value: bool.
var dirtyFiles sync.Map

// IsFileDirty returns true if the file has unsaved user edits.
func IsFileDirty(path string) bool {
	v, ok := dirtyFiles.Load(path)
	return ok && v.(bool)
}

// SetFileDirty marks a file as having (or not having) unsaved user edits.
func SetFileDirty(path string, dirty bool) {
	if dirty {
		dirtyFiles.Store(path, true)
	} else {
		dirtyFiles.Delete(path)
	}
}

// ClearSessionDirty removes all dirty flags (called on session end).
func ClearSessionDirty() {
	dirtyFiles.Range(func(key, _ any) bool {
		dirtyFiles.Delete(key)
		return true
	})
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd d:/git/monika && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/tool/context.go
git commit -m "feat: add dirty file tracking (IsFileDirty, SetFileDirty)"
```

---

### Task 3: Add SetFileDirty Wails binding

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add SetFileDirty method to App**

Add to `internal/api/app.go` (near existing ReadFile/WriteFile methods):

```go
func (a *App) SetFileDirty(projectPath, filePath string, dirty bool) {
	absPath := a.getFileService(projectPath).AbsPath(filePath)
	tool.SetFileDirty(absPath, dirty)
}
```

Note: You need to check the `FileService` type to confirm `AbsPath` method exists. If not, use `filepath.Join(projectPath, filePath)` instead.

- [ ] **Step 2: Verify compilation**

Run: `cd d:/git/monika && go build ./...`
Expected: PASS

- [ ] **Step 3: Regenerate Wails bindings**

Run: `cd d:/git/monika && wails3 task build` (or equivalent Wails regenerate command)

This auto-generates the TS bindings files in `frontend/bindings/` — ToolEvent will now have `conflict`, `diskContent`, `aiContent` fields.

- [ ] **Step 4: Commit**

```bash
git add internal/api/app.go frontend/bindings/
git commit -m "feat: add SetFileDirty Wails binding, regenerate bindings"
```

---

### Task 4: Map new ExecutionResult fields in agent_loop

**Files:**
- Modify: `internal/agent/agent_loop.go:1078-1086`

- [ ] **Step 1: Add Conflict/DiskContent/AiContent to ToolEvent construction**

Modify the ToolEvent literal at `internal/agent/agent_loop.go:1078-1086`:

```go
ch <- Event{
    Type: EventToolOutput,
    Tool: &ToolEvent{
        ID:          tc.ID,
        Name:        tc.Function.Name,
        Input:       tc.Function.Arguments,
        Output:      toolContent,
        Status:      status,
        DiffLines:   execResult.DiffLines,
        Conflict:    execResult.Conflict,
        DiskContent: execResult.DiskContent,
        AiContent:   execResult.AiContent,
    },
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd d:/git/monika && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/agent/agent_loop.go
git commit -m "feat: map conflict/DiskContent/AiContent from ExecutionResult to ToolEvent"
```

---

### Task 5: Add conflict detection to file_edit / file_patch / file_write

**Files:**
- Modify: `internal/tool/builtin/file_edit.go`
- Modify: `internal/tool/builtin/file_patch.go`
- Modify: `internal/tool/builtin/file_write.go`

- [ ] **Step 1: Add conflict detection to file_edit.go Execute()**

In `file_edit.go`, add after path resolution and before the existing edit logic (after line ~87, before `result, err := f.editFile(...)`):

```go
// Check for unsaved user edits
if IsFileDirty(safePath) {
    diskData, _ := os.ReadFile(safePath)
    diskContent := string(diskData)
    // Compute what AI would write: apply edit to disk content
    // Re-use existing editFile logic on disk content
    aiResult, _ := f.editFile(safePath, params.Anchor, params.NewString, lineCount)
    // Read back the hypothetical result by applying edit manually
    // We need to compute diff between diskContent and what AI wants
    lines := strings.Split(strings.ReplaceAll(diskContent, "\r\n", "\n"), "\n")
    // Apply the same edit logic...
    // (For simplicity, compute aiContent via the edit logic directly)
    var aiContent string
    // ... same logic as editFile but applied to diskContent, returning the new content
    aiContent = applyEditToContent(diskContent, params.Anchor, params.NewString, lineCount)
    diff := computeDiff(safePath, diskContent, aiContent)
    return ExecutionResult{
        Content:     fmt.Sprintf("File %s has unsaved user edits. AI edit blocked — showing conflict diff.", safePath),
        Conflict:    true,
        DiskContent: diskContent,
        AiContent:   aiContent,
        DiffLines:   diff,
    }, nil
}
```

But we need an `applyEditToContent` helper. Extract the edit logic from `editFile` into a reusable function.

- [ ] **Step 2: Extract a reusable `applyEditToContent` function in file_edit.go**

Extract the line-replacement logic from `editFile` into a standalone function:

```go
// applyEditToContent applies a line-based edit to the given content string
// and returns the new content. It does NOT write to disk.
func applyEditToContent(content, anchor, newString string, lineCount int) (string, error) {
	hadCRLF := strings.Contains(content, "\r\n")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	newString = strings.ReplaceAll(newString, "\r\n", "\n")

	colonIdx := strings.LastIndex(anchor, ":")
	if colonIdx < 0 {
		return "", fmt.Errorf("invalid anchor format")
	}
	lineNum, err := strconv.Atoi(anchor[colonIdx+1:])
	if err != nil || lineNum < 1 {
		return "", fmt.Errorf("invalid anchor line number")
	}

	lines := strings.Split(content, "\n")
	if lineNum > len(lines) {
		return "", fmt.Errorf("anchor line beyond file length")
	}

	var result []string
	if lineCount == 0 {
		before := lines[:lineNum]
		after := lines[lineNum:]
		insertLines := splitLines(newString)
		result = make([]string, 0, len(before)+len(insertLines)+len(after))
		result = append(result, before...)
		result = append(result, insertLines...)
		result = append(result, after...)
	} else {
		before := lines[:lineNum-1]
		after := lines[lineNum+lineCount-1:]
		replacementLines := splitLines(newString)
		result = make([]string, 0, len(before)+len(replacementLines)+len(after))
		result = append(result, before...)
		result = append(result, replacementLines...)
		result = append(result, after...)
	}

	content = strings.Join(result, "\n")
	if hadCRLF {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}
	return content, nil
}
```

Then `editFile` calls `applyEditToContent` then writes to disk. And the conflict check also calls it.

- [ ] **Step 3: Add conflict check to file_edit.go Execute()**

Add after `safePath, err := f.resolvePath(...)`:

```go
if IsFileDirty(safePath) {
    diskData, _ := os.ReadFile(safePath)
    diskContent := string(diskData)
    aiContent, applyErr := applyEditToContent(diskContent, params.Anchor, params.NewString, lineCount)
    if applyErr != nil {
        return ExecutionResult{Content: applyErr.Error(), IsError: true}, nil
    }
    diff := computeDiff(safePath, diskContent, aiContent)
    return ExecutionResult{
        Content:     fmt.Sprintf("⚠ %s has unsaved user edits. Choose Accept AI or Keep Mine in preview.", safePath),
        Conflict:    true,
        DiskContent: diskContent,
        AiContent:   aiContent,
        DiffLines:   diff,
    }, nil
}
```

- [ ] **Step 4: Add conflict check to file_patch.go Execute()**

Add after `safePath, err := resolveToolPath(...)`:

```go
if IsFileDirty(safePath) {
    diskData, _ := os.ReadFile(safePath)
    diskContent := string(diskData)
    // Apply patch to disk content
    normalized := strings.ReplaceAll(diskContent, "\r\n", "\n")
    normalizedSearch := strings.ReplaceAll(params.Search, "\r\n", "\n")
    normalizedReplace := strings.ReplaceAll(params.Replace, "\r\n", "\n")
    aiContent := strings.Replace(normalized, normalizedSearch, normalizedReplace, 1)
    diff := computeDiff(safePath, normalized, aiContent)
    return ExecutionResult{
        Content:     fmt.Sprintf("⚠ %s has unsaved user edits. Choose Accept AI or Keep Mine in preview.", safePath),
        Conflict:    true,
        DiskContent: diskContent,
        AiContent:   aiContent,
        DiffLines:   diff,
    }, nil
}
```

- [ ] **Step 5: Add conflict check to file_write.go Execute()**

Add after `safePath, err := f.resolvePath(...)`:

```go
if IsFileDirty(safePath) {
    diskData, _ := os.ReadFile(safePath)
    diskContent := string(diskData)
    aiContent := params.Content
    diff := computeDiff(safePath, diskContent, aiContent)
    return ExecutionResult{
        Content:     fmt.Sprintf("⚠ %s has unsaved user edits. Choose Accept AI or Keep Mine in preview.", safePath),
        Conflict:    true,
        DiskContent: diskContent,
        AiContent:   aiContent,
        DiffLines:   diff,
    }, nil
}
```

- [ ] **Step 6: Verify compilation**

Run: `cd d:/git/monika && go build ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/tool/builtin/file_edit.go internal/tool/builtin/file_patch.go internal/tool/builtin/file_write.go
git commit -m "feat: add conflict detection to file_edit, file_patch, file_write"
```

---

### Task 6: Install new CodeMirror dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install @codemirror/search and @codemirror/autocomplete**

Run: `cd d:/git/monika/frontend && npm install @codemirror/search @codemirror/autocomplete`

(`@codemirror/search` provides the search panel; `@codemirror/autocomplete` provides `closeBrackets`/`closeBracketsKeymap`)

- [ ] **Step 2: Commit**

```bash
git add frontend/package.json frontend/package-lock.json
git commit -m "chore: add @codemirror/search and @codemirror/autocomplete deps"
```

---

### Task 7: Add LSP hover and definition to lspService

**Files:**
- Modify: `frontend/src/lib/lspService.ts`

- [ ] **Step 1: Add requestHover method**

Add to `lspService` class after existing `requestDiagnostics` method:

```typescript
async requestHover(filePath: string, line: number, character: number): Promise<{ contents: string; range: { start: { line: number; character: number }; end: { line: number; character: number } } } | null> {
    const server = this.getServerForFile(filePath)
    if (!server) return null
    const uri = this.toUri(filePath)
    try {
        const result = await server.sendRequest('textDocument/hover', {
            textDocument: { uri },
            position: { line, character }
        })
        if (!result || !result.contents) return null
        return {
            contents: typeof result.contents === 'string' ? result.contents
                : result.contents.value || result.contents,
            range: result.range || null
        }
    } catch {
        return null
    }
}
```

- [ ] **Step 2: Add requestDefinition method**

```typescript
async requestDefinition(filePath: string, line: number, character: number): Promise<{ uri: string; range: { start: { line: number; character: number }; end: { line: number; character: number } } } | null> {
    const server = this.getServerForFile(filePath)
    if (!server) return null
    const uri = this.toUri(filePath)
    try {
        const result = await server.sendRequest('textDocument/definition', {
            textDocument: { uri },
            position: { line, character }
        })
        if (!result || (Array.isArray(result) && result.length === 0)) return null
        const loc = Array.isArray(result) ? result[0] : result
        return {
            uri: loc.uri,
            range: loc.range
        }
    } catch {
        return null
    }
}
```

- [ ] **Step 3: Verify TypeScript compilation**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Expected: PASS (no new errors)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/lspService.ts
git commit -m "feat: add requestHover and requestDefinition to lspService"
```

---

### Task 8: Add dirtyFiles state and conflict handler to store

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add dirtyFiles to AppState**

In the `AppState` interface, add after `preview`:

```typescript
dirtyFiles: Set<string>  // paths of files with unsaved user edits
```

- [ ] **Step 2: Add initial value**

In the initial state object (the `create` call), add:

```typescript
dirtyFiles: new Set<string>(),
```

- [ ] **Step 3: Add actions to AppActions**

In the `AppActions` interface, add:

```typescript
markFileDirty: (path: string) => void
markFileClean: (path: string) => void
handleToolConflict: (toolEvent: { filePath: string; name: string; diffLines: string[]; diskContent: string; aiContent: string }) => void
```

- [ ] **Step 4: Implement markFileDirty**

In the `create` callback's returned actions:

```typescript
markFileDirty: (path: string) => {
    set((s) => ({
        dirtyFiles: new Set([...s.dirtyFiles, path])
    }))
    Call.ByName('monika/internal/api.App.SetFileDirty', {
        projectPath: get().projectPath,
        filePath: path,
        dirty: true
    }).catch(() => {})
},

markFileClean: (path: string) => {
    set((s) => {
        const next = new Set(s.dirtyFiles)
        next.delete(path)
        return { dirtyFiles: next }
    })
    Call.ByName('monika/internal/api.App.SetFileDirty', {
        projectPath: get().projectPath,
        filePath: path,
        dirty: false
    }).catch(() => {})
},
```

- [ ] **Step 5: Implement handleToolConflict**

```typescript
handleToolConflict: (toolEvent) => {
    const name = toolEvent.filePath.split('/').pop() || toolEvent.filePath.split('\\').pop() || toolEvent.filePath
    // Set preview to diff mode with conflict actions
    set((s) => ({
        preview: {
            mode: 'diff',
            filePath: toolEvent.filePath,
            fileName: name,
            fileContent: toolEvent.diskContent,
            diffLines: toolEvent.diffLines,
            conflictAiContent: toolEvent.aiContent, // stored for Accept action
            conflictActive: true,
        }
    }))
},
```

Note: `conflictAiContent` and `conflictActive` are new fields we'll add to `PreviewState` in `store/index.ts`:

```typescript
interface PreviewState {
    mode: 'file' | 'diff' | 'task' | null
    filePath: string | null
    fileName: string | null
    fileContent: string | null
    diffLines: string[] | null
    conflictAiContent?: string | null    // NEW
    conflictActive?: boolean              // NEW
}
```

- [ ] **Step 6: Modify tool_output handler to route conflicts**

In the `case 'tool_output'` handler (around line 1688), after the existing `setPreviewDiff` call, add conflict handling:

```typescript
// Existing diffLines check...
if (data.tool.diffLines && data.tool.diffLines.length > 0 && data.tool.input) {
    try {
        const parsed = JSON.parse(data.tool.input)
        if (parsed.filePath) {
            const name = parsed.filePath.split('/').pop() || parsed.filePath.split('\\').pop() || parsed.filePath
            if (data.tool.conflict && data.tool.diskContent && data.tool.aiContent) {
                // Conflict: show diff with Accept/Keep buttons
                useStore.getState().preview = {
                    mode: 'diff',
                    filePath: parsed.filePath,
                    fileName: name,
                    fileContent: data.tool.diskContent,
                    diffLines: data.tool.diffLines,
                    conflictAiContent: data.tool.aiContent,
                    conflictActive: true,
                }
            } else {
                useStore.getState().setPreviewDiff(parsed.filePath, name, data.tool.diffLines)
            }
        }
    } catch { }
}
```

- [ ] **Step 7: Reset conflictActive and conflictAiContent in initial preview state**

```typescript
preview: {
    mode: null,
    filePath: null,
    fileName: null,
    fileContent: null,
    diffLines: null,
    conflictAiContent: null,
    conflictActive: false,
},
```

- [ ] **Step 8: Verify TypeScript compilation**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add dirtyFiles tracking and conflict handler to store"
```

---

### Task 9: Enable editing in PreviewPanel — CM6 extensions and save

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

- [ ] **Step 1: Add imports for new CM6 extensions**

Add to the imports at top of PreviewPanel.tsx:

```typescript
import { history, historyKeymap } from '@codemirror/commands'
import { bracketMatching, indentOnInput, foldGutter, syntaxTree } from '@codemirror/language'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { search, searchKeymap } from '@codemirror/search'
import { keymap } from '@codemirror/view'
import { Call } from '@wailsio/runtime'
```

Note: `closeBrackets` and `closeBracketsKeymap` may already be imported or need the `@codemirror/autocomplete` package. Check if it's already in package.json:

Run: `cd d:/git/monika/frontend && grep "@codemirror/autocomplete" package.json`

If not present, install it: `cd d:/git/monika/frontend && npm install @codemirror/autocomplete`

- [ ] **Step 2: Build the complete set of extensions for edit mode**

Find the existing `extensions` array in PreviewPanel.tsx (used in `EditorState.create({extensions: ...})`). Replace/modify it:

```typescript
const editorExtensions = useMemo(() => [
    lineNumbers(),
    highlightSpecialChars(),
    history(),
    foldGutter(),
    drawSelection(),
    dropCursor(),
    EditorState.allowMultipleSelections.of(true),
    indentOnInput(),
    bracketMatching(),
    closeBrackets(),
    syntaxHighlighting(oneDarkHighlightStyle),
    treeSitterHighlightExtension(lang),
    keymap.of([
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...historyKeymap,
        ...searchKeymap,
    ]),
    search({ top: true }),
    previewTheme,
    // Custom Ctrl+S keybinding
    keymap.of([{
        key: 'Mod-s',
        run: (view) => {
            const content = view.state.doc.toString()
            const state = useStore.getState()
            const filePath = state.preview.filePath
            if (filePath) {
                Call.ByName('monika/internal/api.App.WriteFile', {
                    projectPath: state.projectPath,
                    filePath: filePath,
                    content: content
                }).then(() => {
                    useStore.getState().markFileClean(filePath)
                }).catch((err) => {
                    console.error('Save failed:', err)
                })
            }
            return true
        },
        preventDefault: true,
    }]),
    // Hover tooltip via LSP
    hoverTooltip((view, pos) => {
        const state = useStore.getState()
        if (!state.preview.filePath) return null
        const line = view.state.doc.lineAt(pos)
        const char = pos - line.from
        return new Promise((resolve) => {
            lspService.requestHover(state.preview.filePath!, line.number - 1, char).then((result) => {
                if (!result) { resolve(null); return }
                resolve({
                    pos: pos,
                    end: pos + 50,
                    above: true,
                    create: () => {
                        const dom = document.createElement('div')
                        dom.style.cssText = 'background:#1e2129;border:1px solid rgba(255,255,255,0.1);border-radius:6px;padding:6px 10px;font-size:11px;max-width:400px;font-family:var(--font-mono);'
                        dom.innerHTML = `<span style="color:#abb2bf">${result.contents}</span>`
                        return { dom }
                    }
                })
            }).catch(() => resolve(null))
        })
    }),
    // LSP diagnostics (existing)
    lspDiagnosticField,
    // Dirty tracking: mark file dirty on user edits
    EditorView.updateListener.of((update) => {
        if (update.docChanged && !update.transactions.some(tr => tr.annotation(Transaction.userEvent) === 'undo')) {
            const state = useStore.getState()
            if (state.preview.filePath) {
                state.markFileDirty(state.preview.filePath)
            }
        }
    }),
], [lang])
```

- [ ] **Step 3: Set editable to true**

Find where `EditorView` is created (or `EditorState.create`) and ensure `editable` is not explicitly set to false. By default CM6 View is editable, so remove any `editable: false` configuration.

Find the EditorView construction (search for `new EditorView` or `EditorView` in the file). Remove or change any `editable: false` to omit it (defaults to true).

- [ ] **Step 4: Remove the old cursor simulation hack**

Remove the `Cursor` class and any readonly-cursor-span rendering logic (previously around lines ~479-492). These were workarounds for readonly mode.

- [ ] **Step 5: Remove old @codemirror/lang-* imports**

Remove imports like:
```typescript
import { javascript } from '@codemirror/lang-javascript'
import { python } from '@codemirror/lang-python'
```
These should already be gone per the file-preview-design.md Phase 2 plan. If still present, remove them and their usage.

- [ ] **Step 6: Add Accept AI / Keep Mine buttons to diff view**

In the `DiffView` component (or wherever diff header is rendered), add conflict resolution buttons when `conflictActive` is true:

```typescript
const conflictActive = useStore((s) => s.preview.conflictActive)
const conflictAiContent = useStore((s) => s.preview.conflictAiContent)

// In the diff header div, after the existing file name and +/- counts:
{conflictActive && (
    <div style={{ display: 'flex', gap: 8, marginLeft: 'auto' }}>
        <button
            style={{
                padding: '3px 12px',
                background: 'var(--green)',
                color: '#000',
                border: 'none',
                borderRadius: 4,
                cursor: 'pointer',
                fontSize: 11,
                fontWeight: 600,
            }}
            onClick={async () => {
                const state = useStore.getState()
                if (state.preview.filePath && conflictAiContent) {
                    await Call.ByName('monika/internal/api.App.WriteFile', {
                        projectPath: state.projectPath,
                        filePath: state.preview.filePath,
                        content: conflictAiContent,
                    })
                    useStore.getState().markFileClean(state.preview.filePath!)
                    // Reload file view
                    useStore.getState().setPreview({
                        ...state.preview,
                        conflictActive: false,
                        conflictAiContent: null,
                    })
                }
            }}
        >
            Accept AI
        </button>
        <button
            style={{
                padding: '3px 12px',
                background: 'var(--red)',
                color: '#fff',
                border: 'none',
                borderRadius: 4,
                cursor: 'pointer',
                fontSize: 11,
                fontWeight: 600,
            }}
            onClick={() => {
                const state = useStore.getState()
                useStore.getState().setPreview({
                    ...state.preview,
                    conflictActive: false,
                    conflictAiContent: null,
                })
                useStore.getState().markFileDirty(state.preview.filePath!)
            }}
        >
            Keep Mine
        </button>
    </div>
)}
```

- [ ] **Step 7: Add F12 / Ctrl+Click go-to-definition**

Add a custom keybinding for F12 and a click handler for Ctrl+Click:

```typescript
// Add to the keymap array:
{
    key: 'F12',
    run: (view) => {
        const state = useStore.getState()
        if (!state.preview.filePath) return false
        const pos = view.state.selection.main.head
        const line = view.state.doc.lineAt(pos)
        const char = pos - line.from
        lspService.requestDefinition(state.preview.filePath, line.number - 1, char).then((loc) => {
            if (!loc) return
            // Handle navigation — for same file, scroll. For different file, open new tab.
            const absLocPath = loc.uri.replace('file://', '')  // normalize
            if (absLocPath === state.preview.filePath || loc.uri.endsWith(state.preview.filePath!)) {
                // Same file: scroll to definition
                view.dispatch({
                    selection: { anchor: view.state.doc.line(loc.range.start.line + 1).from + loc.range.start.character },
                    scrollIntoView: true,
                })
            } else {
                // Different file: open in preview
                // (This requires dockview API access — see note below)
            }
        })
        return true
    }
},
```

Note: Cross-file go-to-definition requires dockview API to open a new tab. This is a follow-up task. For MVP, same-file definition works.

Also for Ctrl+Click, add an `EditorView.domEventHandlers` or use `EditorView.clickableLinks` pattern. Simplest: add a `click` handler on the editor DOM that checks `event.ctrlKey || event.metaKey`.

- [ ] **Step 8: Handle file_changed event for external modification detection**

In PreviewPanel's `useEffect`, listen for `Events.On('stream', ...)` for `file_changed` events. When the current open file is externally modified (e.g., by AI after conflict resolution):

```typescript
useEffect(() => {
    const unsub = Events.On('stream', (ev: any) => {
        if (ev.data?.file_change?.path) {
            const currentPath = useStore.getState().preview.filePath
            if (currentPath && ev.data.file_change.path === currentPath && ev.data.file_change.status === 'modified') {
                // Reload file content
                Call.ByName('monika/internal/api.App.ReadFile', {
                    projectPath: useStore.getState().projectPath,
                    filePath: currentPath,
                }).then((result) => {
                    // Update editor content...
                })
            }
        }
    })
    return () => unsub?.()
}, [])
```

- [ ] **Step 9: Verify compilation and test**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Fix any type errors. Then run: `cd d:/git/monika/frontend && npm run dev`
Expected: Editor loads, typing works, Ctrl+S saves, Ctrl+F opens search.

- [ ] **Step 10: Commit**

```bash
git add frontend/src/components/Preview/PreviewPanel.tsx
git commit -m "feat: enable interactive editing in PreviewPanel with CM6 extensions and conflict UI"
```

---

### Task 10: End-to-end verification

- [ ] **Step 1: Verify full build**

Run: `cd d:/git/monika && go build ./...`
Expected: PASS

Run: `cd d:/git/monika/frontend && npm run build`
Expected: PASS

- [ ] **Step 2: Manual test checklist**

- [ ] Open a file from FILES panel — editor loads with syntax highlighting
- [ ] Type in the editor — changes appear, dirty indicator (yellow dot) shows on tab
- [ ] Ctrl+Z / Ctrl+Y — undo/redo works
- [ ] Ctrl+F — search panel opens, find text works
- [ ] Ctrl+S — file saves, dirty indicator clears
- [ ] Ctrl+Click a variable — jumps to definition (same file)
- [ ] Hover over a function — tooltip shows type info
- [ ] Open a file, edit without saving. Trigger AI to modify same file.
- [ ] Verify conflict diff appears in preview with [Accept AI] and [Keep Mine] buttons
- [ ] Click Accept AI — file updates to AI version, dirty cleared
- [ ] Re-test: edit, trigger AI conflict, click Keep Mine — user edits preserved

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: complete interactive editor — editing, LSP, conflict handling"
```
