# Session-Scoped Model Binding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind provider+model per session so switching session tabs restores that session's model in the dropdown (instead of using a single global value).

**Architecture:** Backend `Session` already stores `Model`/`Provider`; add a `SetSessionModel` RPC and persist the used model in `SendMessage`. Frontend introduces `sessionBindings` (per-session source of truth) + `defaultProvider`/`defaultModel` (global, for new sessions). `selectedProvider`/`selectedModel` become a derived mirror of the active session's binding. ModelPicker disables itself while the active session is generating.

**Tech Stack:** Go (Wails v3 services, `testing`), React 18 + TypeScript + Zustand (no frontend test runner — verified via `npm run build`).

**Spec:** `docs/superpowers/specs/2026-06-17-session-model-binding-design.md`

**Testing note:** This codebase has **no frontend test runner** (no jest/vitest — confirmed). Backend tasks use TDD with `go test`. Frontend tasks verify via `npm run build` (tsc typecheck) + manual behavior checks. This follows the project's established conventions (`internal/api/app_test.go` is empty; only `SessionManager` is unit-tested).

---

## File Structure

| File | Change | Responsibility |
|------|--------|----------------|
| `internal/api/app.go` | Modify | Add `SetSessionModel` RPC (~line 545, near `SetSessionPinned`); persist model/provider in `SendMessage` save block (~line 752) |
| `internal/api/session_manager_test.go` | Modify | Add test for model/provider mutation round-trip |
| `frontend/bindings/monika/...` | Regenerate | Pick up `SetSessionModel` binding |
| `frontend/src/store/index.ts` | Modify | Add `sessionBindings`, `defaultProvider`, `defaultModel` state; `applySessionBinding`, `setActiveSessionModel`, `setDefaultModelGlobal` actions; wire binding population into 4 load paths; restore `selected*` in `switchSessionTab`; set `default*` in `loadProviders` |
| `frontend/src/components/Chat/ModelPicker.tsx` | Modify | Repoint `handleSelect` to `setActiveSessionModel`; disable while generating |
| `frontend/src/components/Sidebar/SessionList.tsx` | Modify | `handleNewSession` uses `defaultProvider`/`defaultModel` |
| `frontend/src/components/Settings/ModelsTab.tsx` | Modify | Manage `default*` via `setDefaultModelGlobal` instead of `selected*` |

---

## Task 1: Backend — `SetSessionModel` RPC + `SendMessage` persistence

**Files:**
- Modify: `internal/api/app.go`
- Test: `internal/api/session_manager_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/api/session_manager_test.go`:

```go
func TestSessionModelProviderMutationRoundTrip(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, dir)

	s, err := sm.New("gpt-4", "openai")
	if err != nil {
		t.Fatal(err)
	}
	if err := sm.Save(s); err != nil {
		t.Fatal(err)
	}

	// Simulate what SetSessionModel does: load, mutate provider+model, save.
	loaded, err := sm.Load(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Provider = "deepseek"
	loaded.Model = "deepseek-chat"
	if err := sm.Save(loaded); err != nil {
		t.Fatal(err)
	}

	again, err := sm.Load(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Provider != "deepseek" {
		t.Errorf("provider did not persist: got %q, want deepseek", again.Provider)
	}
	if again.Model != "deepseek-chat" {
		t.Errorf("model did not persist: got %q, want deepseek-chat", again.Model)
	}
}
```

- [ ] **Step 2: Run test to verify it passes (validates persistence contract)**

Run: `go test ./internal/api/ -run TestSessionModelProviderMutationRoundTrip -v`
Expected: PASS (this confirms `SessionManager` persists model/provider mutations — the contract `SetSessionModel` relies on).

- [ ] **Step 3: Add `SetSessionModel` RPC**

In `internal/api/app.go`, add this method immediately after `SetSessionPinned` (after line ~555, before `RenameSession`):

```go
func (a *App) SetSessionModel(projectPath, sessionID, providerID, model string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	defer sm.Unlock()
	s, err := sm.Load(sessionID)
	if err != nil {
		return err
	}
	s.Provider = providerID
	s.Model = model
	return sm.Save(s)
}
```

- [ ] **Step 4: Persist model/provider in `SendMessage` save block**

In `internal/api/app.go`, in `SendMessage`'s goroutine save block (~line 752-757), the block currently is:

```go
		s.Messages = conv.Messages
		s.TokenCount = conv.TokenCount
		s.TokenMax = conv.TokenMax
		s.CompactionCount = conv.CompactionCount
		s.CompactionFrom = conv.CompactionFrom
		sm.SetTitle(s)
```

Change it to also persist the model/provider used for this send:

```go
		s.Messages = conv.Messages
		s.TokenCount = conv.TokenCount
		s.TokenMax = conv.TokenMax
		s.CompactionCount = conv.CompactionCount
		s.CompactionFrom = conv.CompactionFrom
		s.Provider = providerID
		s.Model = model
		sm.SetTitle(s)
```

(`providerID` and `model` are already `SendMessage` parameters.)

- [ ] **Step 5: Verify backend compiles + tests pass**

Run: `go vet ./internal/api/ && go test ./internal/api/ -v`
Expected: no vet errors; all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/app.go internal/api/session_manager_test.go
git commit -m "feat(api): add SetSessionModel RPC and persist model/provider in SendMessage"
```

---

## Task 2: Regenerate Wails bindings

**Files:**
- Regenerate: `frontend/bindings/monika/`

- [ ] **Step 1: Regenerate bindings**

Run:
```bash
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"
```
Expected: completes without error.

- [ ] **Step 2: Verify `SetSessionModel` appears in bindings**

Run: search `frontend/bindings/monika/` for `SetSessionModel`
Expected: a new binding file/method `SetSessionModel` exists (e.g. in `frontend/bindings/monika/internal/api/App.js` / `.d.ts`).

- [ ] **Step 3: Commit**

```bash
git add frontend/bindings/
git commit -m "chore(bindings): regenerate for SetSessionModel"
```

---

## Task 3: Frontend store — add binding state + actions

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add new state fields to the `AppState` interface**

In `frontend/src/store/index.ts`, in the `AppState` interface, find the block (around line 196-199):

```ts
    availableProviders: ProviderInfo[]
    selectedProvider: string
    modelsByProvider: Record<string, ModelInfo[]>
    selectedModel: string
```

Add `sessionBindings` and `default*` right after `selectedModel`:

```ts
    availableProviders: ProviderInfo[]
    selectedProvider: string
    modelsByProvider: Record<string, ModelInfo[]>
    selectedModel: string
    sessionBindings: Record<string, { provider: string; model: string }>
    defaultProvider: string
    defaultModel: string
```

- [ ] **Step 2: Add new action signatures to the `AppState` interface**

Find the existing signature (around line 292):

```ts
    setSelectedProvider: (providerId: string) => Promise<void>
```

Add the three new actions right after it:

```ts
    setSelectedProvider: (providerId: string) => Promise<void>
    applySessionBinding: (id: string, provider?: string, model?: string) => void
    setActiveSessionModel: (providerId: string, modelId: string) => Promise<void>
    setDefaultModelGlobal: (providerId: string, modelId: string) => Promise<void>
```

- [ ] **Step 3: Add initial state values**

Find the initial-state block (around line 388-391):

```ts
    availableProviders: [],
    selectedProvider: '',
    modelsByProvider: {},
    selectedModel: '',
```

Add the new fields:

```ts
    availableProviders: [],
    selectedProvider: '',
    modelsByProvider: {},
    selectedModel: '',
    sessionBindings: {} as Record<string, { provider: string; model: string }>,
    defaultProvider: '',
    defaultModel: '',
```

- [ ] **Step 4: Implement `applySessionBinding`**

Find the existing `setSelectedModel` action (around line 618-635). Add `applySessionBinding` immediately **before** `setSelectedModel`:

```ts
    applySessionBinding: (id, provider, model) => {
        if (!provider || !model) return
        set((s) => {
            const bindings = { ...s.sessionBindings, [id]: { provider, model } }
            if (id !== s.activeSessionId) {
                return { sessionBindings: bindings }
            }
            const models = s.modelsByProvider[provider] || []
            const m = models.find((mm: any) => mm.ID === model) as any
            const newMax = m?.ContextLimit ?? 0
            const current = s.sessionTokens[id]
            return {
                sessionBindings: bindings,
                selectedProvider: provider,
                selectedModel: model,
                ...(newMax > 0
                    ? {
                        tokenMax: newMax,
                        sessionTokens: { ...s.sessionTokens, [id]: { count: current?.count ?? 0, max: newMax } },
                    }
                    : {}),
            }
        })
    },
```

- [ ] **Step 5: Implement `setActiveSessionModel` (used by ModelPicker)**

Add immediately after `setSelectedModel` (after its closing `},` around line 635):

```ts
    setActiveSessionModel: async (providerId, modelId) => {
        const sid = get().activeSessionId
        if (!sid) return
        if (providerId !== get().selectedProvider) {
            await get().loadModelsForProvider(providerId)
        }
        get().applySessionBinding(sid, providerId, modelId)
        const project = get().projectPath
        if (project) {
            App.SetSessionModel(project, sid, providerId, modelId).catch((e: unknown) => {
                console.error('[monika] SetSessionModel failed:', e)
            })
        }
    },
```

- [ ] **Step 6: Implement `setDefaultModelGlobal` (used by ModelsTab)**

Add immediately after `setActiveSessionModel`:

```ts
    setDefaultModelGlobal: async (providerId, modelId) => {
        set({ defaultProvider: providerId, defaultModel: modelId })
        try {
            await Call.ByName('monika/internal/api.App.SetDefaultModel', providerId, modelId)
        } catch (e) {
            console.error('[monika] SetDefaultModel failed:', e)
        }
    },
```

- [ ] **Step 7: Verify typecheck**

Run: `cd frontend && npm run build`
Expected: compiles with no TS errors. (Runtime behavior unchanged yet — actions are defined but not wired into load/switch paths; that's Task 4-5.)

- [ ] **Step 8: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat(store): add sessionBindings + default model state and actions"
```

---

## Task 4: Frontend store — populate `sessionBindings` in all load paths

**Files:**
- Modify: `frontend/src/store/index.ts`

Each of the 4 session-load sites has a `const session = await App.LoadSession(...)` whose result exposes `session.provider` / `session.model`. Add an `applySessionBinding` call at each.

- [ ] **Step 1: Wire `openSessionTab`**

In `openSessionTab` (~line 806-809), find:

```ts
            const project = useStore.getState().projectPath
            const session = await App.LoadSession(project, id)
            const msgs = session?.messages
```

After the `const session = ...` line, the method proceeds into a `set((s) => {...})`. After that `set(...)` call closes (around line 846, right before the `} catch {`), insert the binding call. Find the line:

```ts
        // Mark session as viewed when opened
        const project = useStore.getState().projectPath
```

Insert **before** it:

```ts
        if (session?.provider && session?.model) {
            get().applySessionBinding(id, session.provider, session.model)
        }

        // Mark session as viewed when opened
        const project = useStore.getState().projectPath
```

- [ ] **Step 2: Wire `restoreSessionTabs`**

In `restoreSessionTabs` (~line 1007), find:

```ts
                const session = await App.LoadSession(project, tab.id)
                const msgs = session?.messages
```

Inside the same `try` block, after the `set((s) => ({ ... }))` call that follows (which repairs title + worktree), add (before the closing `} catch {`):

```ts
                if (session?.provider && session?.model) {
                    get().applySessionBinding(tab.id, session.provider, session.model)
                }
```

- [ ] **Step 3: Wire `loadSessionList`**

In `loadSessionList` (~line 1069), find inside the `Promise.allSettled` map:

```ts
                    const session = await App.LoadSession(project, s.id)
                    const msgs = session?.messages
```

After the `set((prev) => ({ ... }))` call that follows (before the `} catch {`), add:

```ts
                    if (session?.provider && session?.model) {
                        get().applySessionBinding(s.id, session.provider, session.model)
                    }
```

- [ ] **Step 4: Wire `pushSubagentOverlay`**

In `pushSubagentOverlay` (~line 958), find:

```ts
            const session = await App.LoadSession(project, subagentId)
            let msgs = session?.messages
```

After the `set((s) => ({ ... }))` call that follows (before `} catch {`), add:

```ts
            if (session?.provider && session?.model) {
                get().applySessionBinding(subagentId, session.provider, session.model)
            }
```

- [ ] **Step 5: Verify typecheck**

Run: `cd frontend && npm run build`
Expected: compiles with no TS errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat(store): populate sessionBindings on session load"
```

---

## Task 5: Frontend store — restore `selected*` on tab switch + set `default*` in `loadProviders`

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Restore model binding in `switchSessionTab`**

In `switchSessionTab` (~line 905-931), find the `updates` object:

```ts
            const updates = {
                activeSessionId: id,
                sessionMessages: currentCache,
                messages: restored,
                displayCounts: { ...s.displayCounts, [id]: INITIAL_DISPLAY_COUNT },
                tokenCount: s.sessionTokens[id]?.count ?? 0,
                tokenMax: s.sessionTokens[id]?.max ?? 0,
                sessionParents: s.sessionParents,
            }
```

Replace it with a version that also restores the bound provider/model (falling back to `default*`):

```ts
            const binding = s.sessionBindings[id]
            const restProvider = binding?.provider || s.defaultProvider
            const restModel = binding?.model || s.defaultModel
            const updates: Record<string, unknown> = {
                activeSessionId: id,
                sessionMessages: currentCache,
                messages: restored,
                displayCounts: { ...s.displayCounts, [id]: INITIAL_DISPLAY_COUNT },
                tokenCount: s.sessionTokens[id]?.count ?? 0,
                tokenMax: s.sessionTokens[id]?.max ?? 0,
                sessionParents: s.sessionParents,
                selectedProvider: restProvider,
                selectedModel: restModel,
            }
```

- [ ] **Step 2: Set `default*` in `loadProviders`**

In `loadProviders` (~line 1199-1203), find:

```ts
        set({
            availableProviders: providers,
            selectedProvider: validProvider,
            ...(persistedModel ? { selectedModel: persistedModel } : {}),
        });
```

Replace to also populate `default*` (keep `selected*` as the initial display value; session activation overrides it):

```ts
        set({
            availableProviders: providers,
            selectedProvider: validProvider,
            defaultProvider: validProvider,
            ...(persistedModel ? { selectedModel: persistedModel, defaultModel: persistedModel } : {}),
        });
```

- [ ] **Step 3: Verify typecheck**

Run: `cd frontend && npm run build`
Expected: compiles with no TS errors.

- [ ] **Step 4: Manual behavior check (core feature)**

Run the app in dev mode: `wails3 dev`
1. Open Session A, send a message with provider GLM.
2. Open Session B, switch the dropdown to DeepSeek, send a message.
3. Switch back to tab A → **dropdown must show GLM again** (not DeepSeek).
4. Switch to tab B → **dropdown must show DeepSeek**.

Expected: dropdown follows the active session's bound model.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat(store): restore session model binding on tab switch; set default model"
```

---

## Task 6: Frontend — ModelPicker per-session binding + disable while generating

**Files:**
- Modify: `frontend/src/components/Chat/ModelPicker.tsx`

- [ ] **Step 1: Read generating state + new action**

In `frontend/src/components/Chat/ModelPicker.tsx`, find the selector block (lines 8-14):

```ts
function ModelPicker() {
  const availableProviders = useStore((s) => s.availableProviders)
  const selectedProvider = useStore((s) => s.selectedProvider)
  const selectedModel = useStore((s) => s.selectedModel)
  const modelsByProvider = useStore((s) => s.modelsByProvider)
  const setSelectedProvider = useStore((s) => s.setSelectedProvider)
  const loadModelsForProvider = useStore((s) => s.loadModelsForProvider)
```

Replace with (swap `setSelectedProvider` for the per-session action; add generating + active session + projectPath):

```ts
function ModelPicker() {
  const availableProviders = useStore((s) => s.availableProviders)
  const selectedProvider = useStore((s) => s.selectedProvider)
  const selectedModel = useStore((s) => s.selectedModel)
  const modelsByProvider = useStore((s) => s.modelsByProvider)
  const setActiveSessionModel = useStore((s) => s.setActiveSessionModel)
  const loadModelsForProvider = useStore((s) => s.loadModelsForProvider)
  const generatingSessionIds = useStore((s) => s.generatingSessionIds)
  const activeSessionId = useStore((s) => s.activeSessionId)
  const isGenerating = generatingSessionIds.includes(activeSessionId)
```

- [ ] **Step 2: Repoint `handleSelect` to per-session binding**

Find `handleSelect` (lines 108-118):

```ts
  const handleSelect = useCallback(
    async (providerId: string, modelId: string) => {
      if (providerId !== selectedProvider) {
        await setSelectedProvider(providerId)
      }
      useStore.getState().setSelectedModel(modelId)
      setOpen(false)
      App.PersistSelection(providerId, modelId).catch((e: unknown) => { logger.error('PersistSelection failed:', e) })
    },
    [selectedProvider, setSelectedProvider],
  )
```

Replace with:

```ts
  const handleSelect = useCallback(
    async (providerId: string, modelId: string) => {
      if (isGenerating) return
      await setActiveSessionModel(providerId, modelId)
      setOpen(false)
    },
    [setActiveSessionModel, isGenerating],
  )
```

- [ ] **Step 3: Disable the trigger button while generating**

Find the trigger button (lines 134-147):

```tsx
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
```

Add `disabled` + dimmed style when generating:

```tsx
      <button
        onClick={() => setOpen((v) => !v)}
        disabled={isGenerating}
        className="text-[11px] px-2 py-0.5 rounded cursor-pointer flex items-center gap-1"
        style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border)',
          color: 'var(--text-primary)',
          fontFamily: 'inherit',
          opacity: isGenerating ? 0.5 : 1,
          cursor: isGenerating ? 'not-allowed' : 'pointer',
        }}
      >
```

- [ ] **Step 4: Verify typecheck**

Run: `cd frontend && npm run build`
Expected: compiles with no TS errors. (Note: `App` import may now be unused in this file — if so, remove the `import { App } from '../../../bindings/monika'` line. `logger` may also become unused — remove its import too. The build will flag these as errors.)

- [ ] **Step 5: Manual behavior check**

Run: `wails3 dev`
1. Send a message in a session (generation starts).
2. While generating, click the model dropdown → **button must be disabled / non-interactive**.
3. After generation completes → dropdown is interactive again; changing it binds to the active session only.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/Chat/ModelPicker.tsx
git commit -m "feat(model-picker): bind to active session; disable while generating"
```

---

## Task 7: Frontend — SessionList + ModelsTab use the right model source

**Files:**
- Modify: `frontend/src/components/Sidebar/SessionList.tsx`
- Modify: `frontend/src/components/Settings/ModelsTab.tsx`

- [ ] **Step 1: SessionList — New Session uses global default**

In `frontend/src/components/Sidebar/SessionList.tsx`, find the selectors (lines 38-39):

```ts
    const selectedModel = useStore((s) => s.selectedModel)
    const selectedProvider = useStore((s) => s.selectedProvider)
```

Replace with the global-default selectors:

```ts
    const defaultModel = useStore((s) => s.defaultModel)
    const defaultProvider = useStore((s) => s.defaultProvider)
```

Then find `handleNewSession` (lines 77-88) and update the `NewSession` call (line 80):

```ts
            const info = await App.NewSession(projectPath, selectedProvider, selectedModel)
```

becomes:

```ts
            const info = await App.NewSession(projectPath, defaultProvider, defaultModel)
```

- [ ] **Step 2: ModelsTab — manage global default via `setDefaultModelGlobal`**

In `frontend/src/components/Settings/ModelsTab.tsx`, find the selectors (lines 115-118):

```ts
  const selectedProvider = useStore((s) => s.selectedProvider)
  const selectedModel = useStore((s) => s.selectedModel)
  const setSelectedProvider = useStore((s) => s.setSelectedProvider)
  const setSelectedModel = useStore((s) => s.setSelectedModel)
```

Replace with the global-default selectors + action:

```ts
  const selectedProvider = useStore((s) => s.defaultProvider)
  const selectedModel = useStore((s) => s.defaultModel)
  const setDefaultModelGlobal = useStore((s) => s.setDefaultModelGlobal)
```

(The local variable names `selectedProvider`/`selectedModel` are kept intentionally — they're referenced throughout the component's JSX for the "default" highlight, e.g. line 293. They now read from `default*`.)

- [ ] **Step 3: ModelsTab — repoint `setDefaultModel` callback**

Find the `setDefaultModel` callback (lines 212-216):

```ts
  const setDefaultModel = useCallback(async (providerId: string, modelId: string) => {
    setSelectedProvider(providerId)
    setSelectedModel(modelId)
    try { await Call.ByName('monika/internal/api.App.SetDefaultModel', providerId, modelId) } catch { /* best effort */ }
  }, [setSelectedProvider, setSelectedModel])
```

Replace with:

```ts
  const setDefaultModel = useCallback(async (providerId: string, modelId: string) => {
    await setDefaultModelGlobal(providerId, modelId)
  }, [setDefaultModelGlobal])
```

- [ ] **Step 4: Verify typecheck + check for unused imports**

Run: `cd frontend && npm run build`
Expected: compiles with no TS errors. If `Call` is now unused in `ModelsTab.tsx`, remove the `import { Call } from '@wailsio/runtime'` line (the build will flag it).

- [ ] **Step 5: Manual behavior check**

Run: `wails3 dev`
1. Settings → Models tab → set a model as default.
2. Click "New Session" → the new session uses the default model (from step 1), NOT the model of whichever session is currently active.
3. Changing the default in Settings must NOT change the active session's dropdown.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/Sidebar/SessionList.tsx frontend/src/components/Settings/ModelsTab.tsx
git commit -m "feat(ui): new session uses global default; settings manages global default"
```

---

## Task 8: Full build + end-to-end verification

**Files:** none (verification only)

- [ ] **Step 1: Full backend build + tests**

Run:
```bash
go vet ./...
go test ./...
```
Expected: no errors.

- [ ] **Step 2: Full frontend build**

Run: `cd frontend && npm run build`
Expected: compiles with no errors.

- [ ] **Step 3: End-to-end manual verification**

Run: `wails3 dev`

Verify each scenario:
1. **Tab switch restores model** (core): Session A=GLM, Session B=DS → switching tabs shows each session's own model in the dropdown.
2. **Dropdown change is per-session**: change A to DS → B's model stays as its own; global default unchanged.
3. **Disable while generating**: while A is generating, its dropdown is disabled.
4. **New session uses global default**: New Session inherits Settings default, not the active session's model.
5. **Settings default is independent**: changing default in Settings doesn't affect existing sessions' dropdowns.
6. **Persistence across restart**: change a session's model, restart app, reopen the session → dropdown shows the changed model (validated by `SendMessage` writeback + `SetSessionModel`).
7. **Fallback**: a session with empty model (edge) → dropdown falls back to global default.

- [ ] **Step 4: Final commit (if any cleanup)**

Only if verification surfaced fixes:
```bash
git add -A
git commit -m "fix: verification adjustments for session model binding"
```

---

## Self-Review Notes

**Spec coverage:**
- `SetSessionModel` RPC → Task 1
- `SendMessage` persistence → Task 1
- `sessionBindings` + `default*` + `selected*` derived → Task 3
- Populate on load (4 sites) → Task 4
- Restore on switch → Task 5
- Dropdown per-session + disable while generating → Task 6
- New session uses default → Task 7
- Settings manages default → Task 7
- Bindings regeneration → Task 2
- Edge cases (empty model fallback, historical sessions) → covered by fallback logic in Task 5 + manual check 7

**Type consistency:** `applySessionBinding`, `setActiveSessionModel`, `setDefaultModelGlobal` signatures match across interface decl (Task 3) and implementations (Tasks 3/6/7). `sessionBindings` shape `{provider, model}` consistent throughout.

**Known limitation honestly disclosed:** Frontend has no test runner; frontend tasks use `npm run build` + manual verification instead of automated tests (matches project conventions).
