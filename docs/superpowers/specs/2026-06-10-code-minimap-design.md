# Code Minimap — Design Spec

## Overview

Add a VS Code-style minimap (code overview with syntax highlighting) that overlays the editor's scrollbar area. The minimap renders a miniature view of the source code on a `<canvas>` element, with syntax colors matching the One Dark palette. Users can click or drag on the minimap to scroll the editor.

## Approach: Canvas Rendering

Render the minimap on an HTML5 Canvas. Each line is drawn as colored horizontal spans (1-2px height), with character-level color from tree-sitter syntax highlighting. The viewport indicator is painted as a semi-transparent overlay showing the currently visible range. Interaction (click/drag/scroll) uses canvas event handlers.

### Why Canvas (not DOM)

- Performance: 5000 lines × ~60px canvas is lightweight vs. 5000 DOM elements
- Matches VS Code's implementation
- Pixel-level control for character coloring

## Component Architecture

```
PreviewPanel.tsx
  └─ editor container (<div>)
       ├─ CodeMirror EditorView <div ref={containerRef}>
       └─ <CodeMinimap>  ← new component, absolute positioned right
```

### CodeMinimap Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| content | string | — | File content to render |
| totalLines | number | — | Total lines in the file |
| editorViewRef | RefObject\<EditorView \| null\> | — | Reference to the CodeMirror editor view |
| width | number | 60 | Canvas width in pixels |

### New File

`frontend/src/components/Preview/CodeMinimap.tsx`

## Rendering Pipeline

### Phase 1 — Token Colors (content change only)

1. Parse content with `treeSitterInitPromise` + `getLanguageForFile`
2. Walk syntax tree, for each token assign a color from the One Dark palette:
   - Keywords = `#c678dd` (purple)
   - Strings = `#98c379` (green)
    - Comments = `#5c6370` (gray)
   - Functions = `#61afef` (blue)
   - Types = `#56b6c2` (cyan)
   - Numbers = `#d19a66` (orange)
   - Variables = `#e06c75` (pink)
   - Other/unknown = `#abb2bf` (base text, warm gray)
3. For each line, produce `colors: { from: number, to: number, color: string }[]`
4. If tree-sitter fails or language has no grammar, fall back to all-`#abb2bf`
5. Parsed color data is cached and only regenerated when content changes (not on scroll)

### Phase 2 — Canvas Paint (scroll + content change)

1. Canvas size: `width = 60px`, `height = container.clientHeight`
2. Line height: `canvasHeight / totalLines` (minimum 2px)
3. For each line:
   - For each color span: `ctx.fillStyle = span.color`; draw horizontal rect
   - Character width ≈ `60px / line.length`, capped at 1px
4. Viewport indicator:
   - Semi-transparent rect (`rgba(255,255,255,0.08)`) covering visible line range
   - Right edge: 2px bright border (`rgba(255,255,255,0.3)`)

## Placement in PreviewPanel

```
<div style="flex flex-col flex-1 min-h-0; position: relative">
  <FilePreviewHeader />
  <div ref={containerRef} style="flex:1; overflow:hidden">
    <!-- CodeMirror editor -->
  </div>
  <CodeMinimap
    content={displayContent}
    totalLines={totalLines}
    editorViewRef={editorRef}
    style={{ position: 'absolute', top: 0, right: 0, height: '100%' }}
  />
</div>
```

- Minimap overlays the scrollbar area of the editor
- When `showSymbols` is open on the right, the editor container shrinks and minimap stays inside

## Interaction

| Action | Implementation |
|--------|---------------|
| Click | Compute target line from y coordinate → `editorView.dispatch(EditorView.scrollIntoView(line, { y: 'center' }))` |
| Drag | `onmousedown` + `onmousemove` → map y to line → `editorView.scrollDOM.scrollTop = (line/totalLines) * scrollHeight` |
| Wheel | `onwheel` → `preventDefault()` → forward `deltaY` to `editorView.scrollDOM.scrollTop` |
| Viewport sync | Listen to `editorView.scrollDOM.onscroll` → recalculate visible line range → repaint indicator |

## Scroll Sync

- Minimap subscribes to `editorView.scrollDOM.scroll` event when mounted
- Only repaints the viewport indicator (not the full content) on scroll
- Full repaint (Phase 1 + 2) only on content change

## Error Handling

- tree-sitter parse failure → fallback to monochrome rendering (all `#abb2bf`)
- Canvas not supported → render nothing (minimap is decorative/navigational, not essential)
- Editor not yet initialized → defer mounting until `editorRef.current` is set

## Edge Cases

- **Empty file (0 lines)**: Hide minimap entirely
- **Single line**: Minimap shows one thick line; viewport indicator covers full height
- **Very long lines (>200 chars)**: Character width compressed; canvas may show solid-color band
- **Binary files**: Not shown (minimap only renders when `showFile` is true)
