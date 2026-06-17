# Session-Scoped Model Binding

**Date:** 2026-06-17
**Status:** Approved (design)

## Problem

The model/provider selector is **global**. When a user opens multiple sessions and switches between them, the model dropdown does not reflect the session that was (or should be) used:

- Session A was created with GLM, Session B with DeepSeek.
- Switching to B and selecting DeepSeek changes the **global** selector.
- Switching back to A leaves the dropdown on DeepSeek — sending a message in A now uses DeepSeek instead of GLM.

The backend `Session` struct already persists `Model` + `Provider` per session (`internal/api/session_manager.go:30-31`), and `NewSession` sets them at creation. The defect is entirely on the frontend: `selectedProvider`/`selectedModel` are global single values, and `switchSessionTab` restores messages/tokens but **not** the model.

## Goal

Each session binds to its own provider + model. Switching the active session restores that session's bound model in the dropdown. Changing the dropdown only affects the active session, never the global default.

## Decisions

| Decision | Choice |
|----------|--------|
| Dropdown change scope | Binds to the **active session only**; global default untouched |
| New session default model | Uses the **global default** (`~/.monika/config.json` `model_provider`/`model`) |
| Architecture | **Single source of truth** — `selected*` is a derived view of the active session's binding, not independent global state |

## Architecture (Approach B)

### Backend

No schema change — `Session.Model` / `Session.Provider` already exist.

**New RPC — `SetSessionModel`** (`internal/api/app.go`):
```go
func (a *App) SetSessionModel(projectPath, sessionID, providerID, model string) error
```
- Loads the session, sets `s.Provider` / `s.Model`, saves. Reuses the existing `getSessionManager` + lock pattern used by `SetSessionPinned` / `RenameSession`.

**`SendMessage` persists the used model** (`internal/api/app.go`, ~line 752):
- The save block currently sets `s.Messages`, `s.TokenCount`, … but **omits** `s.Model`/`s.Provider`. Add:
  ```go
  s.Provider = providerID
  s.Model = model
  ```
  (the values are already function parameters). This keeps the on-disk session in sync with whatever model was used, including mid-session model changes that happen via the dropdown before the next send.

No other backend changes. `GetDefaultModel` / `SetDefaultModel` / `PersistSelection` remain the global-default mechanism, unchanged.

### Frontend

**Split the two concepts the global `selected*` currently conflates:**

| State | Meaning | Consumers |
|-------|---------|-----------|
| `sessionBindings: Record<sessionId, { provider: string; model: string }>` | **Source of truth** — each session's bound model. Populated when a session loads. | — |
| `selectedProvider` / `selectedModel` | **Derived** — equals `sessionBindings[activeSessionId]`, falling back to `default*` when missing. | ModelPicker, ChatArea, ChatInput (send/compact) |
| `defaultProvider` / `defaultModel` | Global default (for new sessions). Loaded once from `GetDefaultModel`. | SessionList (New Session), ModelsTab (settings) |

`selected*` is no longer independent global state — it is always a reflection of the active session's binding (or the global default when no binding exists).

**Populate `sessionBindings` on session load.** Wherever a session is loaded from the backend and `session.model` / `session.provider` are available, write them into `sessionBindings[id]`:
- `openSessionTab` (`store/index.ts:777`)
- `restoreSessionTabs` (`store/index.ts:998`)
- `loadSessionList` (`store/index.ts:1054`)
- `pushSubagentOverlay` (`store/index.ts:943`)

A helper keeps this DRY:
```ts
const applySessionBinding = (id: string, provider?: string, model?: string) => { ... }
```

**Restore on tab switch.** `switchSessionTab` (`store/index.ts:905`) already restores `messages`, `tokenCount`, `tokenMax`. Extend it to restore `selectedProvider`/`selectedModel` from `sessionBindings[id]` (fallback `default*`). The token-max restoration already present in `setSelectedModel` must run here too so the token bar matches the restored model's context limit.

**Dropdown change no longer touches global default.** `ModelPicker.onSelect` (`ModelPicker.tsx:110`) currently calls `setSelectedProvider` / `setSelectedModel`. Repoint it to a new action that:
1. Updates `sessionBindings[activeSessionId]` and `selected*`.
2. Calls `App.SetSessionModel(projectPath, activeSessionId, provider, model)`.
3. Does **not** call `SetDefaultModel`.

**Disable dropdown while generating.** `ModelPicker` reads `generatingSessionIds` and disables the provider/model selector when the active session is generating. The model for the in-flight turn is fixed at send time (`WithModel` is baked into the running loop), so a mid-generation change cannot affect the current turn anyway — disabling it avoids confusing the user. The dropdown re-enables when generation completes.

**New Session keeps using global default.** `SessionList.handleNewSession` (`SessionList.tsx:80`) switches from `selectedProvider`/`selectedModel` → `defaultProvider`/`defaultModel`.

**Settings keeps managing global default.** `ModelsTab` (`ModelsTab.tsx:215`) unchanged — still `SetDefaultModel`, now reads/writes `default*`.

### Bindings regeneration

Because `SetSessionModel` is only needed for the Wails Go API, regenerate bindings after adding it:
```bash
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"
```

## Behavior Flows

| Flow | Behavior |
|------|----------|
| Switch to tab B | `selected*` ← `sessionBindings[B]` (or `default*`); token bar updates to B's model context limit |
| Open/load session | `sessionBindings[id]` ← `session.{provider,model}` |
| Change dropdown in active session | Update `sessionBindings[active]` + `selected*`; persist via `SetSessionModel`; global default untouched |
| Model change while generating | Dropdown disabled for the active session while it is in `generatingSessionIds`; re-enabled on completion |
| New session | Created with `default*`; after creation its binding = default |
| Send message | Uses `selected*` (= active session binding); `SendMessage` also writes model/provider back to the session JSON |
| Settings → change default | Updates `default*` only (`SetDefaultModel`) |

## Edge Cases

- **Historical sessions**: created with `NewSession` which always set model/provider → load with correct binding, no migration needed.
- **Session with empty model/provider** (legacy/edge): `selected*` falls back to `default*`; the first send backfills the model via the `SendMessage` persistence fix.
- **Child sessions** (`sub_` / `call_compact_`): they carry their own model. Binding restore uses their stored model, falling back to `default*` if absent. No change to how they inherit at spawn time.
- **Provider/model becomes invalid** (e.g. user deleted a provider): existing fallback logic in `loadProviders` / `loadModelsForProvider` already picks a valid first option; this is unchanged.

## Out of Scope

- Changing how child/sub-agent sessions inherit models at spawn time.
- Per-session temperature or other agent parameters.
- A visual model badge on the session tab/list (could be a follow-up).
