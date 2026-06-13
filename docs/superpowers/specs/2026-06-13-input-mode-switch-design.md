# Input Mode Switch Design

## Summary

Add an input mode toggle before the permission mode picker in the chat input footer. Two modes: **Normal** (existing behavior — AI chat with `/` commands) and **Shell** (direct command execution, equivalent to the existing `$` prefix but without needing to type `$`). The mode is persisted per session and only exists in memory.

## Approach

Option A (lightweight, in ChatInput): Add `inputModes` to the Zustand store keyed by session ID, a new `InputModePicker` component styled identically to `PermissionModePicker`, and conditional branches in `ChatInput` for submit/autocomplete/history behavior. No new components split — a single editor serves both modes.

Key design decisions:

- **Mode replaces `$` prefix**: In shell mode, typing a command and pressing Enter executes it directly. No `$` needed. The existing `$` prefix continues to work in normal mode.
- **Visual style unchanged**: The editor looks the same in both modes. Only the toggle button indicates the current mode.
- **Per-session persistence in memory**: Each session remembers its own mode while the app is running. On restart, all sessions reset to normal mode.
- **Autocomplete in shell mode**: `@` file path completion retained; `/` command completion disabled (shell mode doesn't need `/init`, `/compact`, `/skill`).
- **Tab inline path completion**: In shell mode, Tab performs terminal-style inline file path completion instead of the dropdown-based autocomplete.

## Data Model

### Store Extension

```typescript
// frontend/src/store/index.ts — AppState interface
inputModes: Record<string, 'normal' | 'shell'>  // key = sessionId
```

- Default: `'normal'` when no entry exists for a session.
- Not persisted to localStorage — memory only.
- Action: `setInputMode(sessionId: string, mode: 'normal' | 'shell')`

## Components

### InputModePicker (new)

`frontend/src/components/Chat/InputModePicker.tsx`

- Follows `PermissionModePicker` pattern exactly: 11px button with dropdown popup.
- Two options: `Normal` and `Shell`.
- Renders in ChatInput footer, left of PermissionModePicker.
- Keyboard: Escape to close, ArrowUp/ArrowDown to navigate, Enter to select.
- Reads/writes `inputModes[sessionId]` from the store.

### ChatInput.tsx (modified)

The existing `ChatInput` component gains mode-aware behavior branches:

#### handleSubmit

When `inputMode === 'shell'`:
- Skip `$` prefix check, `/init`, `/compact`, `/skill` command routing, and quote message wrapping.
- Take the trimmed input text directly as the shell command.
- Save to shell command history (`localStorage` key `monika-cmd-history-{sessionId}`).
- Call `onRunShell(command)`.

When `inputMode === 'normal'`:
- Behavior unchanged.

#### Autocomplete

- Shell mode: `fetchAutocomplete` skips `getQueryAtCursor` results where `prefix === '/'` (no `/` command autocomplete). `@` path completion works normally.
- Normal mode: unchanged.

#### History navigation (ArrowUp/ArrowDown)

- Shell mode: Uses shell command history (same source as `$` prefix mode). The `isShellMode` check in `handleKeyDown` changes from `valueRef.current.startsWith('$')` to reading `inputMode`.
- Normal mode: Uses user message history. Unchanged.

#### Tab key — inline path completion (shell mode)

When `inputMode === 'shell'` and Tab is pressed (no autocomplete dropdown open):
1. Extract the text fragment before the cursor (from the last whitespace to cursor position).
2. If the fragment is empty or doesn't look like a path, do nothing.
3. Call `App.ListFileTree(projectPath, false)` (cached — already fetched for `@` autocomplete).
4. Find all files whose path starts with the fragment.
5. On first Tab: insert the first match (replacing the fragment).
6. On subsequent Tab presses (within the same position): cycle to the next match.
7. If no matches, do nothing.

Cycling state resets when the user moves the cursor or types another character.

## File Changes

| File | Change |
|------|--------|
| `frontend/src/store/index.ts` | Add `inputModes` field + `setInputMode` action |
| `frontend/src/components/Chat/InputModePicker.tsx` | New — Shell/Normal toggle button |
| `frontend/src/components/Chat/ChatInput.tsx` | Import InputModePicker; add mode-aware branches in handleSubmit, handleKeyDown, autocomplete, and Tab handler |

## Behavior Matrix

| Feature | Normal mode | Shell mode |
|---------|-------------|------------|
| Enter submits as | AI chat message | Shell command |
| `$` prefix | Works (runs shell command) | Not needed (everything is shell) |
| `/` commands | `/init`, `/compact`, `/skill` | N/A |
| `@` autocomplete | File path dropdown | File path dropdown |
| Tab with no dropdown | Default browser behavior | Inline path completion |
| ArrowUp/ArrowDown | User message history | Shell command history |
| Mode persistence | Per session, memory only | Per session, memory only |
| Quote wrapping | Wraps quoted messages | N/A |
