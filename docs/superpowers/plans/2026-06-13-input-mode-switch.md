# Input Mode Switch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Normal/Shell input mode toggle before the Permission mode switcher in the ChatInput footer.

**Architecture:** A new `inputModes` record in the Zustand store (keyed by sessionId), a new `InputModePicker` component mirroring `PermissionModePicker`, and mode-aware conditional branches in `ChatInput` for submit, autocomplete, history, and Tab behavior.

**Tech Stack:** React 18, TypeScript, Zustand, Tailwind CSS v4

---

## File Map

| File | Role |
|------|------|
| `frontend/src/store/index.ts` | Add `inputModes: Record<string, 'normal' \| 'shell'>` + `setInputMode` action |
| `frontend/src/components/Chat/InputModePicker.tsx` | New — Normal/Shell toggle button (mirrors PermissionModePicker) |
| `frontend/src/components/Chat/ChatInput.tsx` | Integrate picker, add mode-branched submit/autocomplete/history/tab logic |

---

### Task 1: Store — add inputModes state

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add `inputModes` field to AppState interface**

In the `AppState` interface, add after `permissionMode` (line 199):

```typescript
inputModes: Record<string, 'normal' | 'shell'>
```

In the action methods interface, add after `setPermissionMode` (line 227):

```typescript
setInputMode: (sessionId: string, mode: 'normal' | 'shell') => void
```

In the `set` call that creates initial state (find the `set` call inside the `create` function, near where `permissionMode: 'auto'` is set), add:

```typescript
inputModes: {},
```

After the `setPermissionMode` implementation (around line 629), add:

```typescript
setInputMode: (sessionId, mode) => set((s) => ({
    inputModes: { ...s.inputModes, [sessionId]: mode },
})),
```

- [ ] **Step 2: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add inputModes state and setInputMode action to store"
```

---

### Task 2: InputModePicker component

**Files:**
- Create: `frontend/src/components/Chat/InputModePicker.tsx`

- [ ] **Step 1: Create InputModePicker.tsx**

```typescript
import { useState, useRef, useEffect } from 'react'
import { useStore } from '../../store'
import { IconChevronDown } from '../Icons'

const MODES: { id: 'normal' | 'shell'; label: string }[] = [
    { id: 'normal', label: 'Normal' },
    { id: 'shell', label: 'Shell' },
]

function InputModePicker() {
    const activeSessionId = useStore((s) => s.activeSessionId)
    const inputMode = useStore((s) => s.inputModes[activeSessionId] || 'normal')
    const setInputMode = useStore((s) => s.setInputMode)

    const [open, setOpen] = useState(false)
    const [focusIdx, setFocusIdx] = useState(0)
    const ref = useRef<HTMLDivElement>(null)

    useEffect(() => {
        if (!open) return
        const handler = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
        }
        document.addEventListener('mousedown', handler)
        return () => document.removeEventListener('mousedown', handler)
    }, [open])

    useEffect(() => {
        if (open) setFocusIdx(0)
    }, [open])

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') { setOpen(false); return }
        if (e.key === 'ArrowDown') {
            e.preventDefault()
            setFocusIdx((prev) => Math.min(prev + 1, MODES.length - 1))
        } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setFocusIdx((prev) => Math.max(prev - 1, 0))
        } else if (e.key === 'Enter') {
            e.preventDefault()
            handleSelect(MODES[focusIdx].id)
        }
    }

    const current = MODES.find((m) => m.id === inputMode) || MODES[0]

    const handleSelect = (id: 'normal' | 'shell') => {
        setInputMode(activeSessionId, id)
        setOpen(false)
    }

    return (
        <div ref={ref} style={{ position: 'relative' }}>
            <button
                onClick={() => setOpen((v) => !v)}
                className="text-[11px] px-2 py-0.5 rounded cursor-pointer flex items-center gap-1"
                style={{
                    background: 'var(--bg-elevated)',
                    border: '1px solid var(--border)',
                    color: 'var(--text-primary)',
                    fontFamily: 'inherit',
                }}
            >
                <span>{current.label}</span>
                <IconChevronDown size={8} />
            </button>
            {open && (
                <div
                    style={{
                        position: 'absolute',
                        bottom: '100%',
                        left: 0,
                        marginBottom: '4px',
                        minWidth: '100%',
                        maxHeight: '240px',
                        overflowY: 'auto',
                        background: 'var(--bg-elevated)',
                        border: '1px solid var(--border-strong)',
                        borderRadius: 'var(--radius-md, 6px)',
                        padding: '4px',
                        zIndex: 1000,
                        boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
                    }}
                    onKeyDown={handleKeyDown}
                >
                    {MODES.map((mode, idx) => {
                        const isSelected = mode.id === inputMode
                        return (
                            <button
                                key={mode.id}
                                onClick={() => handleSelect(mode.id)}
                                onMouseEnter={() => setFocusIdx(idx)}
                                className="text-[11px] w-full text-left px-2 py-1 rounded cursor-pointer"
                                style={{
                                    background:
                                        idx === focusIdx
                                            ? 'var(--bg-hover)'
                                            : isSelected
                                                ? 'var(--accent-muted)'
                                                : 'transparent',
                                    color: isSelected ? 'var(--accent)' : 'var(--text-primary)',
                                    border: 'none',
                                    fontFamily: 'inherit',
                                }}
                            >
                                {mode.label}
                            </button>
                        )
                    })}
                </div>
            )}
        </div>
    )
}

export default InputModePicker
```

- [ ] **Step 2: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/InputModePicker.tsx
git commit -m "feat: add InputModePicker component for shell/normal toggle"
```

---

### Task 3: Integrate InputModePicker into ChatInput footer

**Files:**
- Modify: `frontend/src/components/Chat/ChatInput.tsx`

- [ ] **Step 1: Add import and wire into footer**

Add import at the top (after the PermissionModePicker import, around line 7):

```typescript
import InputModePicker from './InputModePicker'
```

In the JSX return, in the footer div (around line 759), insert `<InputModePicker />` before `<PermissionModePicker />`:

```tsx
<InputModePicker />
<PermissionModePicker />
```

- [ ] **Step 2: Read inputMode from store**

In the component body, add after the `activeSessionId` selector (around line 231):

```typescript
const inputMode = useStore((s) => s.inputModes[activeSessionId] || 'normal')
```

- [ ] **Step 3: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Chat/ChatInput.tsx
git commit -m "feat: integrate InputModePicker into ChatInput footer"
```

---

### Task 4: Shell mode handleSubmit

**Files:**
- Modify: `frontend/src/components/Chat/ChatInput.tsx`

- [ ] **Step 1: Modify handleSubmit to branch on inputMode**

In `handleSubmit` (around line 581), wrap the existing `$` prefix check to also trigger for shell mode. Replace the existing `$` prefix block and the `/init`, `/compact`, `/skill` blocks with the following structure:

At the start of `handleSubmit`, after `if (!trimmed || disabled) return`, add:

```typescript
if (inputMode === 'shell') {
    if (resolved.startsWith('$')) {
        resolved = resolved.slice(1).trim()
    }
    if (!resolved) { onSend(resolved); setValue(''); return }
    const h = historyRef.current.filter(c => c !== resolved)
    const updated = [resolved, ...h].slice(0, 50)
    historyRef.current = updated
    saveHistory(sessionIdRef.current, updated)
    onRunShell(resolved)
    setValue('')
    return
}
```

The existing `$` prefix, `/init`, `/compact`, `/skill` handling (lines 591–633) remains unchanged for normal mode.

- [ ] **Step 2: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/ChatInput.tsx
git commit -m "feat: shell mode handleSubmit — direct command execution"
```

---

### Task 5: Shell mode autocomplete — disable `/` commands

**Files:**
- Modify: `frontend/src/components/Chat/ChatInput.tsx`

- [ ] **Step 1: Filter out `/` prefix matches in shell mode**

In `getQueryAtCursor` (line 464), the function returns `{ prefix, query, cursor }` matches. Rather than modifying `getQueryAtCursor`, modify the caller `fetchAutocomplete` to skip `/` prefix when in shell mode.

In `updateAutocomplete` (around line 541), after the initial match check, add a guard:

```typescript
const updateAutocomplete = useCallback(() => {
    const match = getQueryAtCursor()
    if (!match) {
        setAc(s => s.open ? { ...s, open: false } : s)
        return
    }
    // In shell mode, skip / command autocomplete
    if (inputMode === 'shell' && match.prefix === '/') {
        setAc(s => s.open ? { ...s, open: false } : s)
        return
    }
    if (acDebounceRef.current) clearTimeout(acDebounceRef.current)
    acDebounceRef.current = setTimeout(() => {
        fetchAutocomplete(match.prefix, match.query)
    }, 300)
}, [value, fetchAutocomplete, inputMode])
```

- [ ] **Step 2: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/ChatInput.tsx
git commit -m "feat: disable / command autocomplete in shell mode"
```

---

### Task 6: Shell mode history navigation

**Files:**
- Modify: `frontend/src/components/Chat/ChatInput.tsx`

- [ ] **Step 1: Change history source detection**

In `handleKeyDown` (around line 681), the check `const isShellMode = valueRef.current.startsWith('$')` determines which history source to use. Change it to:

```typescript
const isShellMode = inputMode === 'shell'
```

On line 709, the fallback when navigating past the end of history sets `setValue(isShellMode ? '$ ' : '')`. In shell mode, the fallback should just be an empty string (no `$` prefix needed), so change to:

```typescript
setValue(isShellMode ? '' : '')
```

Actually, since both are empty string now, simplify to:

```typescript
setValue('')
```

- [ ] **Step 2: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/ChatInput.tsx
git commit -m "feat: use inputMode for shell history navigation source"
```

---

### Task 7: Tab inline path completion (shell mode)

**Files:**
- Modify: `frontend/src/components/Chat/ChatInput.tsx`

- [ ] **Step 1: Add tab completion state refs**

In the component body, near other refs (around line 244), add:

```typescript
const tabCycleRef = useRef<{ wordStart: number; matches: string[]; index: number } | null>(null)
```

- [ ] **Step 2: Add tab completion handler and wire into handleKeyDown**

Add a helper function inside the component before `handleKeyDown`:

```typescript
const handleTabComplete = useCallback(() => {
    const el = editorRef.current
    if (!el) return
    const cursor = getTextOffset(el)
    const textBefore = value.slice(0, cursor)
    const wordStart = Math.max(0, ...[' ', '\n', '\t'].map(c => textBefore.lastIndexOf(c))) + 1
    const fragment = textBefore.slice(wordStart)
    if (!fragment) return

    // If cycling through existing matches
    const cycle = tabCycleRef.current
    if (cycle && cycle.wordStart === wordStart && cycle.matches.length > 0) {
        const nextIdx = (cycle.index + 1) % cycle.matches.length
        tabCycleRef.current = { ...cycle, index: nextIdx }
        const replaceText = cycle.matches[nextIdx]
        const newValue = value.slice(0, wordStart) + replaceText + value.slice(cursor)
        setValue(newValue)
        pendingCursorRef.current = wordStart + replaceText.length
        return
    }

    // Fetch file list for matching
    App.ListFileTree(projectPath, false)
        .then(r => flattenFiles(r as FileEntry[]))
        .then(files => {
            const matches = files
                .filter(f => f.path.startsWith(fragment))
                .map(f => f.is_dir ? f.path + '/' : f.path)
                .slice(0, 20)
            if (matches.length === 0) return
            tabCycleRef.current = { wordStart, matches, index: 0 }
            const newValue = value.slice(0, wordStart) + matches[0] + value.slice(cursor)
            setValue(newValue)
            pendingCursorRef.current = wordStart + matches[0].length
        })
        .catch(() => { /* ignore */ })
}, [value, projectPath])

// Reset tab cycle when user moves cursor or types
useEffect(() => {
    tabCycleRef.current = null
}, [value])
```

In `handleKeyDown`, after the autocomplete Tab handling (around line 652), add before the normal `Enter` handling:

```typescript
if (inputMode === 'shell' && e.key === 'Tab' && !ac.open) {
    e.preventDefault()
    handleTabComplete()
    return
}
```

- [ ] **Step 2: Build check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/ChatInput.tsx
git commit -m "feat: Tab inline path completion in shell mode"
```

---

### Task 8: Full verification build

- [ ] **Step 1: Full build**

```bash
cd frontend && npm run build
```

Expected: Build succeeds with no errors.

- [ ] **Step 2: Commit (if any fixes were needed)**

If anything was fixed during build, commit the fixes.
