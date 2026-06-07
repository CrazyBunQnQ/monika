# Code Minimap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a VS Code-style canvas-based code minimap with syntax highlighting, viewport indicator, and click/drag scroll overlaid on the editor's scrollbar area.

**Architecture:** New `CodeMinimap` React component renders file content as colored lines on `<canvas>` using tree-sitter token colors. A shared utility `treeSitterColors.ts` provides per-line color span extraction. `PreviewPanel.tsx` adds the component and wires scroll sync.

**Tech Stack:** React, CodeMirror 6, tree-sitter (web-tree-sitter), Canvas 2D API

---

### Task 1: Create tree-sitter color extraction utility

**Files:**
- Create: `frontend/src/lib/treeSitterColors.ts`

- [ ] **Step 1: Write the utility file**

Create a utility that takes file content + file path, parses it with tree-sitter, and returns per-line color spans. It has its own tag-to-hex color map matching the One Dark palette used in `treeSitterHighlight.ts`.

```typescript
import { tags, Tag } from '@lezer/highlight'
import { getLanguageForFile, treeSitterInitPromise, Language, Parser } from './treeSitter'
import { Tree } from 'web-tree-sitter'

// ── Tag name → hex color (One Dark palette) ──────────────────────────────

const TAG_TO_HEX: Record<string, string> = {
  comment: '#5c6370',
  docComment: '#5c6370',
  string: '#98c379',
  escape: '#98c379',
  meta: '#c678dd',
  number: '#d19a66',
  literal: '#d19a66',
  bool: '#d19a66',
  color: '#d19a66',
  unit: '#d19a66',
  keyword: '#c678dd',
  moduleKeyword: '#c678dd',
  controlKeyword: '#c678dd',
  definitionKeyword: '#c678dd',
  self: '#c678dd',
  typeName: '#56b6c2',
  className: '#56b6c2',
  namespace: '#56b6c2',
  variableName: '#e06c75',
  propertyName: '#e06c75',
  function: '#61afef',
  standard: '#61afef',
  attributeName: '#61afef',
  labelName: '#61afef',
}

// ── Node type → tag (simplified from treeSitterHighlight.resolveTag) ─────

type TagEntry = [string, Tag]

const tagToName = new Map<Tag, string>()
for (const [k, v] of Object.entries(tags)) {
  tagToName.set(v as Tag, k)
}

function tagToHex(tag: Tag | null): string | null {
  if (!tag) return null
  // Walk the tag hierarchy: check tag and its parent tags
  let t: Tag | null = tag
  while (t) {
    const name = tagToName.get(t)
    if (name && TAG_TO_HEX[name]) return TAG_TO_HEX[name]
    t = t.parent ?? null
  }
  return null
}

// We need a minimal matchTag/resolveTag — simplified from treeSitterHighlight
const patternCache = new Map<string, { re: RegExp; isExact: boolean }>()

function compilePattern(p: string): { re: RegExp; isExact: boolean } {
  const cached = patternCache.get(p)
  if (cached) return cached
  let isExact = false
  let re: RegExp
  if (p.startsWith('*') && p.endsWith('*')) {
    re = new RegExp(p.slice(1, -1))
  } else if (p.endsWith('*')) {
    re = new RegExp('^' + p.slice(0, -1))
  } else if (p.startsWith('*')) {
    re = new RegExp(p.slice(1) + '$')
  } else {
    re = new RegExp('^' + p + '$')
    isExact = true
  }
  const entry = { re, isExact }
  patternCache.set(p, entry)
  return entry
}

const tagMap: TagEntry[] = [
  ['comment', tags.comment],
  ['*comment', tags.comment],
  ['*doc_comment*', tags.docComment],
  ['string', tags.string],
  ['interpreted_string_literal', tags.string],
  ['string_literal', tags.string],
  ['string_fragment', tags.string],
  ['string_*', tags.string],
  ['*_string', tags.string],
  ['escape_sequence', tags.escape],
  ['interpolation', tags.meta],
  ['number', tags.number],
  ['int_literal', tags.number],
  ['float_literal', tags.number],
  ['*_number', tags.number],
  ['boolean', tags.bool],
  ['keyword', tags.keyword],
  ['keyword_*', tags.keyword],
  ['*keyword', tags.keyword],
  ['if', tags.controlKeyword],
  ['else', tags.controlKeyword],
  ['for', tags.controlKeyword],
  ['while', tags.controlKeyword],
  ['do', tags.controlKeyword],
  ['switch', tags.controlKeyword],
  ['case', tags.controlKeyword],
  ['default', tags.keyword],
  ['break', tags.controlKeyword],
  ['continue', tags.controlKeyword],
  ['return', tags.controlKeyword],
  ['match', tags.controlKeyword],
  ['let', tags.definitionKeyword],
  ['var', tags.definitionKeyword],
  ['const', tags.definitionKeyword],
  ['fn', tags.definitionKeyword],
  ['function_definition', tags.definitionKeyword],
  ['func', tags.definitionKeyword],
  ['def', tags.definitionKeyword],
  ['type_definition', tags.definitionKeyword],
  ['type', tags.definitionKeyword],
  ['struct', tags.definitionKeyword],
  ['enum', tags.definitionKeyword],
  ['trait', tags.definitionKeyword],
  ['interface', tags.definitionKeyword],
  ['impl', tags.definitionKeyword],
  ['self', tags.self],
  ['type_identifier', tags.typeName],
  ['type_*', tags.typeName],
  ['class_*', tags.className],
  ['namespace_*', tags.namespace],
  ['identifier', tags.variableName],
  ['field_identifier', tags.propertyName],
  ['property_identifier', tags.propertyName],
  ['call_expression', tags.function],
  ['method_*', tags.function],
  ['function_*', tags.function],
  ['*_expression', tags.standard],
  ['ERROR', tags.invalid],
  ['(', tags.paren],
  [')', tags.paren],
  ['{', tags.brace],
  ['}', tags.brace],
  ['[', tags.bracket],
  [']', tags.bracket],
  [';', tags.semicolon],
  [',', tags.comma],
  ['.', tags.delimiter],
  ['->', tags.arrow],
  ['=>', tags.arrow],
  ['operator', tags.operator],
  ['*_operator', tags.operator],
  ['attribute', tags.attributeName],
  ['*_attribute', tags.attributeName],
]

function matchTag(nt: string): Tag | null {
  const exact: TagEntry[] = []
  const wild: TagEntry[] = []
  for (const [p, tag] of tagMap) {
    const { isExact } = compilePattern(p)
    ;(isExact ? exact : wild).push([p, tag])
  }
  for (const [p, tag] of exact) {
    const { re } = patternCache.get(p)!
    if (re.test(nt)) return tag
  }
  for (const [p, tag] of wild) {
    const { re } = patternCache.get(p)!
    if (re.test(nt)) return tag
  }
  return null
}

function resolveTag(nt: string, parentStack: string[]): Tag | null {
  const pLen = parentStack.length
  const parent = pLen > 0 ? parentStack[pLen - 1] : ''
  const grandparent = pLen > 1 ? parentStack[pLen - 2] : ''

  if (nt === 'identifier') {
    if (parent === 'call_expression') return tags.function(tags.variableName)
    return null
  }

  if (nt === 'field_identifier' || nt === 'property_identifier') {
    if (parent === 'selector_expression' && grandparent === 'call_expression') {
      return tags.function(tags.variableName)
    }
  }

  return matchTag(nt)
}

// ── Public API ───────────────────────────────────────────────────────────

export interface ColorSpan {
  from: number // byte offset relative to the file start
  to: number
  color: string // hex color
}

export interface LineColors {
  line: number // 0-based line number
  spans: ColorSpan[]
}

export async function getLineColors(
  content: string,
  filePath: string
): Promise<LineColors[]> {
  const result: LineColors[] = []
  const langName = getLanguageForFile(filePath)
  if (!langName) {
    // No language — all base color
    const lines = content.split('\n')
    for (let i = 0; i < lines.length; i++) {
      result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#abb2bf' }] })
    }
    return result
  }

  let parser: Parser
  try {
    const init = await treeSitterInitPromise
    parser = init
  } catch {
    // Fallback to monochrome
    const lines = content.split('\n')
    for (let i = 0; i < lines.length; i++) {
      result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#abb2bf' }] })
    }
    return result
  }

  const lang = getLanguage(langName)
  if (!lang) {
    const lines = content.split('\n')
    for (let i = 0; i < lines.length; i++) {
      result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#abb2bf' }] })
    }
    return result
  }

  parser.setLanguage(lang)
  const tree = parser.parse(content)
  const root = tree.rootNode

  // Walk the syntax tree and collect colored spans per line
  const lineColors = new Map<number, { from: number; to: number; color: string }[]>()
  const lines = content.split('\n')
  for (let i = 0; i < lines.length; i++) {
    lineColors.set(i, [])
  }

  function walk(node: any, parents: string[]) {
    const nt = node.type || node.name
    const tag = resolveTag(nt, parents)
    const hex = tagToHex(tag)
    if (hex) {
      const startLine = content.slice(0, node.startIndex).split('\n').length - 1
      const endLine = content.slice(0, node.endIndex).split('\n').length - 1
      if (startLine === endLine) {
        const col = lineColors.get(startLine)
        if (col) {
          col.push({ from: node.startIndex, to: node.endIndex, color: hex })
        }
      }
    }
    parents.push(nt)
    if (node.children) {
      for (const child of node.children) {
        walk(child, parents)
      }
    }
    parents.pop()
  }

  if (root.children) {
    for (const child of root.children) {
      walk(child, [])
    }
  }

  // Build result: for each line, merge overlapping spans and fill gaps with base color
  for (let i = 0; i < lines.length; i++) {
    const spans = lineColors.get(i) || []
    // Sort by from
    spans.sort((a, b) => a.from - b.from)
    // Merge overlapping spans
    const merged: { from: number; to: number; color: string }[] = []
    let prev = spans[0]
    if (prev) {
      merged.push(prev)
      for (let j = 1; j < spans.length; j++) {
        const cur = spans[j]
        if (cur.from <= prev.to && cur.from >= prev.from) {
          // Same color, extend
          if (cur.to > prev.to && cur.color === prev.color) {
            prev.to = cur.to
          }
          // Different color on same range — keep first
        } else {
          merged.push(cur)
          prev = cur
        }
      }
    }
    result.push({ line: i, spans: merged.length > 0 ? merged : [{ from: 0, to: lines[i].length, color: '#abb2bf' }] })
  }

  return result
}
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -20
```
Expected: no errors in treeSitterColors.ts (may have pre-existing errors elsewhere — that's fine)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/treeSitterColors.ts
git commit -m "feat: add tree-sitter color extraction utility for minimap"
```

---

### Task 2: Create CodeMinimap React component

**Files:**
- Create: `frontend/src/components/Preview/CodeMinimap.tsx`

- [ ] **Step 1: Write the component**

This component:
- Receives content, totalLines, editorView, filePath, width props
- Uses a `<canvas>` element for rendering
- Calls `getLineColors()` on content change
- Paints colored lines + viewport indicator
- Handles click, drag, wheel interaction
- Syncs with editor scroll position

```typescript
import React, { useEffect, useRef, useCallback, useState } from 'react'
import { EditorView } from '@codemirror/view'
import { getLineColors, LineColors } from '../../lib/treeSitterColors'

interface CodeMinimapProps {
interface CodeMinimapProps {
  content: string
  totalLines: number
  editorView: EditorView | null  // the editor view instance (triggers re-mount on change)
  filePath: string
  width?: number
}
  width?: number
}

const LINE_HEIGHT_MIN = 2
const VIEWPORT_OVERLAY = 'rgba(255,255,255,0.08)'
const VIEWPORT_BORDER = 'rgba(255,255,255,0.3)'
const BASE_COLOR = '#abb2bf'
const BG_COLOR = '#08090d'

export function CodeMinimap({ content, totalLines, editorView, filePath, width = 60 }: CodeMinimapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const colorsRef = useRef<LineColors[] | null>(null)
  const viewTopRef = useRef(0)
  const viewBottomRef = useRef(0)
  const isDragging = useRef(false)

  // Phase 1: parse content → colors (on content change)
  useEffect(() => {
    let cancelled = false
    getLineColors(content, filePath).then((colors) => {
      if (cancelled) return
      colorsRef.current = colors
      paint()
    })
    return () => { cancelled = true }
  }, [content, filePath])

  // Container height tracking
  const [containerHeight, setContainerHeight] = useState(0)
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerHeight(entry.contentRect.height)
      }
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  // Paint function
  const paint = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || !containerHeight) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    canvas.width = width * dpr
    canvas.height = containerHeight * dpr
    ctx.scale(dpr, dpr)

    // Background
    ctx.fillStyle = BG_COLOR
    ctx.fillRect(0, 0, width, containerHeight)

    if (totalLines === 0) return

    const lineH = Math.max(LINE_HEIGHT_MIN, containerHeight / totalLines)
    const colors = colorsRef.current

    // Draw each line
    for (let i = 0; i < totalLines; i++) {
      const y = i * lineH
      const h = Math.ceil(lineH)

      if (colors && i < colors.length && colors[i].spans.length > 0) {
        const lineColors = colors[i].spans
        const lineLen = content.split('\n')[i]?.length || 1
        const charW = Math.min(1, width / lineLen)

        for (const span of lineColors) {
          const x = Math.round((span.from / Math.max(1, lineLen)) * width)
          const spanW = Math.max(1, Math.round(((span.to - span.from) / Math.max(1, lineLen)) * width))
          ctx.fillStyle = span.color
          ctx.fillRect(x, y, spanW, h)
        }
      } else {
        // Default color for this line
        ctx.fillStyle = BASE_COLOR
        const lineLen = content.split('\n')[i]?.length || 1
        const w = Math.min(width, Math.round((lineLen / Math.max(1, lineLen)) * width))
        ctx.fillRect(0, y, w, h)
      }
    }

    // Viewport indicator
    const viewTop = viewTopRef.current
    const viewBottom = viewBottomRef.current
    if (viewTop >= 0 && viewBottom > viewTop) {
      const vy = viewTop * lineH
      const vh = Math.max(lineH, (viewBottom - viewTop) * lineH)
      ctx.fillStyle = VIEWPORT_OVERLAY
      ctx.fillRect(0, vy, width, vh)
      // Right edge border
      ctx.fillStyle = VIEWPORT_BORDER
      ctx.fillRect(width - 2, vy, 2, vh)
    }
  }, [content, totalLines, containerHeight, width])

  // Repaint when containerHeight changes (resize)
  useEffect(() => {
    paint()
  }, [paint])

  // Scroll sync from editor
  useEffect(() => {
    if (!editorView) return

    const scroller = editorView.scrollDOM
    const onScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = scroller
      const lineH = clientHeight / Math.max(1, totalLines)
      viewTopRef.current = Math.round(scrollTop / lineH)
      viewBottomRef.current = Math.round((scrollTop + clientHeight) / lineH)
      paint()
    }
    scroller.addEventListener('scroll', onScroll)
    // Initial paint with viewport
    onScroll()

    return () => scroller.removeEventListener('scroll', onScroll)
  }, [editorView, totalLines, paint])

  // Click handler
  const handleClick = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!containerHeight || !editorView) return
    const rect = canvasRef.current?.getBoundingClientRect()
    if (!rect) return
    const y = e.clientY - rect.top
    const targetLine = Math.floor((y / containerHeight) * totalLines)
    const clampedLine = Math.max(0, Math.min(targetLine, totalLines - 1))
    const view = editorView
    const line = view.state.doc.line(clampedLine + 1)
    view.dispatch({
      effects: EditorView.scrollIntoView(line.from, { y: 'center' }),
      selection: { anchor: line.from },
    })
  }, [containerHeight, totalLines, editorView])

  // Drag handler
  const handleMouseDown = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    isDragging.current = true
    scrollToY(e.clientY)
  }, [containerHeight, totalLines, editorView])

  useEffect(() => {
    if (!isDragging.current) return
    const onMove = (e: MouseEvent) => {
      if (isDragging.current) scrollToY(e.clientY)
    }
    const onUp = () => { isDragging.current = false }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [containerHeight, totalLines, editorView])

  function scrollToY(clientY: number) {
    if (!containerHeight || !editorView) return
    const rect = canvasRef.current?.getBoundingClientRect()
    if (!rect) return
    const y = clientY - rect.top
    const targetLine = Math.floor((y / containerHeight) * totalLines)
    const clampedLine = Math.max(0, Math.min(targetLine, totalLines - 1))
    const view = editorView
    const scrollH = view.scrollDOM.scrollHeight - view.scrollDOM.clientHeight
    view.scrollDOM.scrollTop = (clampedLine / totalLines) * scrollH
  }

  // Wheel handler
  const handleWheel = useCallback((e: React.WheelEvent<HTMLCanvasElement>) => {
    e.preventDefault()
    if (!editorView) return
    editorView.scrollDOM.scrollTop += e.deltaY
  }, [editorView])

  if (totalLines === 0) return null

  return (
    <div
      ref={containerRef}
      style={{
        position: 'absolute',
        top: 0,
        right: 0,
        height: '100%',
        width: width,
        zIndex: 10,
        pointerEvents: 'auto',
        cursor: 'pointer',
      }}
    >
      <canvas
        ref={canvasRef}
        style={{ width: '100%', height: '100%', display: 'block' }}
        onClick={handleClick}
        onMouseDown={handleMouseDown}
        onWheel={handleWheel}
      />
    </div>
  )
}
```

- [ ] **Step 2: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -30
```
Expected: no errors in CodeMinimap.tsx

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Preview/CodeMinimap.tsx
git commit -m "feat: add CodeMinimap canvas component"
```

---

### Task 3: Integrate into PreviewPanel

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

- [ ] **Step 1: Add the import**

Find the existing import section (around line 15) and add:

```typescript
import { CodeMinimap } from './CodeMinimap'
```

- [ ] **Step 2: Place the CodeMinimap component**

Find the file preview container div (around line 1022, the one with `showFile` display logic) and add `position: 'relative'` to its style, then add CodeMinimap as a child.

The current structure near line 1022:
```tsx
<div style={{ display: showFile ? 'flex' : 'none', flex: 1, overflow: 'hidden', flexDirection: 'column' }}>
  <FilePreviewHeader ... />
  <div ref={containerRef} style={{ flex: 1, overflow: 'hidden' }} onContextMenu={...}>
  </div>
</div>
```

Change to:
```tsx
<div style={{ display: showFile ? 'flex' : 'none', flex: 1, overflow: 'hidden', flexDirection: 'column', position: 'relative' }}>
  <FilePreviewHeader ... />
  <div ref={containerRef} style={{ flex: 1, overflow: 'hidden' }} onContextMenu={...}>
  </div>
  {editorRef.current && (
    <CodeMinimap
      content={displayContent}
      totalLines={totalLines}
      editorView={editorRef.current}
      filePath={preview.filePath || ''}
      width={60}
    />
  )}
</div>
```

Note: Verify the exact line numbers and content match the current state of PreviewPanel.tsx before editing. The code above is the intended structure — use the actual current content when editing.

- [ ] **Step 3: Verify TypeScript compilation**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -30
```
Expected: zero errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Preview/PreviewPanel.tsx
git commit -m "feat: integrate CodeMinimap into preview panel"
```

---

### Task 4: Manual integration test

- [ ] **Step 1: Build and launch the app**

```bash
cd frontend && npm run build 2>&1 | tail -10
```

Expected: build succeeds (exit code 0)

- [ ] **Step 2: Visual verification**
  1. Open Monika and navigate to a Go/TS/Python file
  2. Confirm a 60px-wide minimap appears on the right edge of the editor
  3. Confirm it shows colored lines matching the syntax
  4. Click on the minimap → editor scrolls to that line
  5. Drag on the minimap → editor follows
  6. Scroll with mouse wheel on minimap → editor scrolls
  7. Resize the window → minimap adjusts height
  8. Open a binary/empty file → minimap is hidden
