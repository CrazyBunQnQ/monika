# File Preview Optimization Design

**Date**: 2026-06-07
**Branch**: `feat/optimize-file-preview`
**Status**: designing

## Overview

Optimize the file preview panel by:
1. Removing redundant frontend diff computation (duplicate of backend)
2. Replacing CodeMirror language packages with existing tree-sitter infrastructure for syntax highlighting
3. Adding large file protection and binary file detection
4. Adding line limits to backend `computeDiff`

## Phase 1: Remove Frontend Diff Duplication

### Current State

Backend `computeDiff()` in `diff.go` produces unified diffs for `file_edit`, `file_write`, and `file_patch`. These diffs are streamed via `tool_output` events and the frontend `tool_output` handler (store/index.ts:1580-1588) calls `setPreviewDiff` directly.

Frontend also has a duplicate `simpleDiff()` (~90 lines, O(m*n) LCS DP) in PreviewPanel.tsx, plus a fallback `useEffect` (lines 493-530) that recomputes diffs client-side when the backend diff hasn't arrived yet. This fallback is triggered by `lastEditedFile` / `lastEditedOldContent` / `lastEditVersion` store fields populated by the `tool_start` handler.

Since the backend **always** produces diffs for file-modifying tools, the fallback is dead code under normal operation.

### Changes

**`frontend/src/components/Preview/PreviewPanel.tsx`**:
- Remove `simpleDiff()` function
- Remove `MAX_DIFF_LINES` constant
- Remove diff trigger `useEffect` (lines 493-530)
- Remove store subscriptions: `lastEditedFile`, `lastEditedOldContent`, `lastEditVersion`

**`frontend/src/store/index.ts`**:
- Remove `AppState` fields: `lastEditedFile`, `lastEditedOldContent`, `lastEditVersion`
- Remove `AppActions.setLastEditedFile`
- Remove initial state values
- Remove `tool_start` handler code that reads file before edit (lines 1551-1562)
- Remove `updateToolDone` code that sets `lastEditedFile` and `lastEditVersion` (lines 428-429)
- Remove reset state lines

## Phase 2: Tree-sitter Syntax Highlighting Bridge

### Current State

PreviewPanel uses 4 `@codemirror/lang-*` packages (go, javascript, python, json) for syntax highlighting. Meanwhile, `lib/treeSitter.ts` already supports 19 languages via web-tree-sitter WASM files loaded from `/grammars/tree-sitter-${lang}.wasm`, used only for AST queries (grep `ast_pattern`) and summaries (`file_read summary=true`).

### Design

Create a bridge module `treeSitterHighlight.ts` that converts tree-sitter parse results into CodeMirror `Decoration` sets, replacing all `@codemirror/lang-*` imports.

**Architecture**:

```
FileTree click
  → PreviewPanel useEffect
    → langFromPath(filePath)            // existing in treeSitter.ts
    → getLanguage(langName)             // loads WASM, cached
    → parser.parse(content)             // → Tree
    → buildDecorations(tree, source)    // new: AST → Decorations
    → EditorState.create({
        doc: content,
        extensions: [
          previewTheme,
          syntaxHighlighting(oneDarkHighlightStyle),
          treeSitterHighlightField,     // new: StateField with DecorationSet
          ...
        ]
      })
```

**Node type → Tag mapping** (glob patterns against `node.type`):

| Glob pattern | CodeMirror Tag |
|---|---|
| `*comment*`, `*doc_comment*` | `tags.comment` |
| `*string*`, `string_fragment`, `escape_sequence` | `tags.string` |
| `*number*`, `*int_literal*`, `*float_literal*` | `tags.number` |
| `*true*`, `*false*` (boolean exposed as type) | `tags.bool` |
| `*function*`, `*method*`, `*arrow*` (name child only) | `tags.function(tags.variableName)` |
| `*type*`, `*class*`, `*struct*`, `*interface*`, `*enum*` (name only) | `tags.typeName` |
| `*property*`, `*field*` | `tags.propertyName` |
| `tag_name` (HTML/XML/JSX) | `tags.tagName` |
| `attribute` (HTML/XML) | `tags.attributeName` |
| `heading` (Markdown) | `tags.heading` |
| `*keyword*` | `tags.keyword` |
| `*operator*` | `tags.operator` |
| `*regex*` | `tags.regexp` |

Language-specific overrides can be added for CSS (class_name → `tags.className`, property_name → `tags.propertyName`) and Markdown (emphasis → `tags.emphasis`, link → `tags.url`).

**Files**:

- **`lib/treeSitterHighlight.ts`** (new): `buildDecorations(language, source)`, tag mapping table, `treeSitterHighlightField(langName)` CodeMirror extension
- **`lib/treeSitter.ts`**: Ensure `getLanguage()` and `extToLang()` / `langFromPath()` are exported
- **`PreviewPanel.tsx`**: Remove `getLangExtension()`, remove 4 `@codemirror/lang-*` imports, use `treeSitterHighlightField(langName)` instead
- **`package.json`**: Remove `@codemirror/lang-go`, `@codemirror/lang-javascript`, `@codemirror/lang-python`, `@codemirror/lang-json`

**Supported languages after change**: 19 (up from 4) — go, javascript, typescript, python, rust, java, c, cpp, c_sharp, ruby, php, swift, kotlin, scala, json, yaml, toml, html, css, bash — go, javascript, typescript, python, rust, java, c, cpp, c_sharp, ruby, php, swift, kotlin, scala, json, yaml, toml, html, css, bash

## Phase 3: Large File Protection & Binary Detection

### Large files

Files with > 5000 lines should be truncated in the preview. The preview header shows "X lines (truncated to 5000)". Tree-sitter highlighting is skipped entirely for truncated files (avoids parse overhead).

### Binary files

Detection: read first 8KB of file, check for null bytes (`\x00`) or invalid UTF-8. Files with BOM (UTF-8/UTF-16/UTF-32 byte-order marks) are recognized as text, not binary. If binary, display "[Binary file — preview not available]" placeholder.

**`PreviewPanel.tsx`**: Add binary detection in the file preview `useEffect`.

## Phase 4: Backend Diff Line Limit

Add `MAX_DIFF_LINES = 2000` to `computeDiff()`. If either input exceeds this limit, return `["diff too large (m vs n lines)"]` instead of computing the LCS table.

**`internal/tool/builtin/diff.go`**: Guard at top of `computeDiff`.

## Data Flow (post-optimization)

```
User clicks file in FileTree
  → FileTree calls App.ReadFile() → setPreviewFile(path, name, content)
  → PreviewPanel: binary? → placeholder
  → PreviewPanel: large? → truncate + skip TS parse
  → PreviewPanel: load tree-sitter language → parse → build decorations
  → CodeMirror renders with full syntax highlighting

Agent edits a file (file_edit/file_write/file_patch)
  → Go tool computes computeDiff(old, new) → ExecutionResult.DiffLines
  → agent_loop sends ToolEvent with DiffLines to frontend
  → store tool_output handler calls setPreviewDiff(path, name, diffLines)
  → PreviewPanel DiffView renders parsed hunks
  (No fallback — backend always provides diffs)

User clicks changed file in ChangesList
  → ChangesList calls App.GetFileDiff() → setPreviewDiff(path, name, lines)
  → PreviewPanel DiffView renders parsed hunks
```

## Testing

- Verify all 19 tree-sitter languages produce correct decorations
- Verify binary files show placeholder
- Verify large files are truncated with header indicator
- Verify file edits always trigger diff preview
- Verify `npm run build` / `npm run lint` pass
- Verify removed store fields don't break any code paths
