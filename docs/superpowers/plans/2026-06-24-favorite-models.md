# 偏好模型分组 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "preferred models" group to ModelPicker dropdown, allowing users to star favorite models stored in localStorage.

**Architecture:** Add `favoriteModels` string array to the Zustand store with `toggleFavoriteModel` method. Persist to localStorage key `monika:favorite_models`. In ModelPicker, flatten favorites into a `favorite-header` + model rows at the top of `flatItems`, with star buttons on hover.

**Tech Stack:** React 18, TypeScript, Tailwind CSS v4, Zustand

---

## File Structure

| File | Change | Responsibility |
|------|--------|---------------|
| `frontend/src/store/index.ts` | Modify | Add `favoriteModels` state, type, init, localStorage loader, and `toggleFavoriteModel` |
| `frontend/src/components/Chat/ModelPicker.tsx` | Modify | Add `favorite-header` FlatItem type, favorites section to flatItems, star button, hover CSS |

---

### Task 1: Add favoriteModels to store type and initial state

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add `favoriteModels` to AppState interface**

Add after `selectedModel` on line 213:

```typescript
favoriteModels: string[]
```

- [ ] **Step 2: Add method declarations to AppState interface**

Add after `setActiveSessionModel` on line 314:

```typescript
toggleFavoriteModel: (providerId: string, modelId: string) => void
```

- [ ] **Step 3: Add localStorage loader function**

Add before `export const useStore = create<AppState>((set, get) => ({` (line 389):

```typescript
function loadFavoriteModels(): string[] {
    try {
        const raw = localStorage.getItem('monika:favorite_models')
        if (!raw) return []
        const parsed = JSON.parse(raw)
        if (!Array.isArray(parsed)) return []
        return parsed.filter((item: unknown): item is string => typeof item === 'string')
    } catch {
        return []
    }
}
```

- [ ] **Step 4: Add `favoriteModels` to initial state**

Replace `favoriteModels: [] as string[],` (added in Step 1 placeholder) with:

```typescript
favoriteModels: loadFavoriteModels(),
```

(Add after `selectedModel: '',` on line 429)

- [ ] **Step 5: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS (methods not yet implemented, but declared — may have error about missing method implementations; that's expected)

- [ ] **Step 6: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add favoriteModels type, initial state, and localStorage loader to store"
```

---

### Task 2: Implement toggleFavoriteModel

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Implement toggleFavoriteModel**

Add after `setActiveSessionModel` implementation (after line 699):

```typescript
toggleFavoriteModel: (providerId, modelId) => {
    const key = `${providerId}:${modelId}`
    set((s) => {
        const exists = s.favoriteModels.includes(key)
        const next = exists
            ? s.favoriteModels.filter((k) => k !== key)
            : [...s.favoriteModels, key]
        try {
            localStorage.setItem('monika:favorite_models', JSON.stringify(next))
        } catch { /* ignore quota or disabled */ }
        return { favoriteModels: next }
    })
},
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: implement toggleFavoriteModel with localStorage persistence"
```

---

### Task 3: Add favorite group to ModelPicker flatItems

**Files:**
- Modify: `frontend/src/components/Chat/ModelPicker.tsx`

- [ ] **Step 1: Subscribe to favoriteModels and toggleFavoriteModel**

Add after `const modelsByProvider = useStore((s) => s.modelsByProvider)` on line 10:

```typescript
const favoriteModels = useStore((s) => s.favoriteModels)
const toggleFavoriteModel = useStore((s) => s.toggleFavoriteModel)
```

- [ ] **Step 2: Extend FlatItem type**

Replace lines 64-66:

```typescript
type FlatItem =
    | { type: 'provider'; provider: ProviderInfo }
    | { type: 'model'; provider: ProviderInfo; model: ModelInfo }
```

With:

```typescript
type FlatItem =
    | { type: 'favorite-header' }
    | { type: 'provider'; provider: ProviderInfo }
    | { type: 'model'; provider: ProviderInfo; model: ModelInfo; isFavorite?: boolean }
```

- [ ] **Step 3: Rewrite flatItems useMemo with favorites logic**

Replace lines 68-90 (the entire `useMemo` call) with:

```typescript
const flatItems = useMemo((): FlatItem[] => {
    const items: FlatItem[] = []
    const searchLower = search.toLowerCase()

    // Build lookup: "providerId:modelId" -> { provider, model }
    const allModelsLookup = new Map<string, { provider: ProviderInfo; model: ModelInfo }>()
    for (const p of availableProviders) {
        const models = modelsByProvider[p.id] || []
        for (const m of models) {
            allModelsLookup.set(`${p.id}:${m.ID}`, { provider: p, model: m })
        }
    }

    // Favorite group
    const validFavorites: { provider: ProviderInfo; model: ModelInfo }[] = []
    for (const key of favoriteModels) {
        const entry = allModelsLookup.get(key)
        if (!entry) continue
        if (searchLower) {
            if (!entry.model.DisplayName.toLowerCase().includes(searchLower)
                && !entry.model.ID.toLowerCase().includes(searchLower)) continue
        }
        validFavorites.push(entry)
    }
    if (validFavorites.length > 0) {
        items.push({ type: 'favorite-header' })
        for (const { provider, model } of validFavorites) {
            items.push({ type: 'model', provider, model, isFavorite: true })
        }
    }

    // Provider groups
    for (const p of availableProviders) {
        const models = modelsByProvider[p.id] || []
        const filtered = searchLower
            ? models.filter((m) =>
                m.DisplayName.toLowerCase().includes(searchLower) ||
                m.ID.toLowerCase().includes(searchLower)
            )
            : models

        if (filtered.length === 0) continue
        if (availableProviders.length > 1) {
            items.push({ type: 'provider', provider: p })
        }
        for (const m of filtered) {
            items.push({
                type: 'model',
                provider: p,
                model: m,
                isFavorite: favoriteModels.includes(`${p.id}:${m.ID}`)
            })
        }
    }
    return items
}, [availableProviders, modelsByProvider, search, favoriteModels])
```

- [ ] **Step 4: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Chat/ModelPicker.tsx
git commit -m "feat: add favorites group to ModelPicker flatItems"
```

---

### Task 4: Add favorite-header and star button rendering

**Files:**
- Modify: `frontend/src/components/Chat/ModelPicker.tsx`

- [ ] **Step 1: Add `inFavoriteGroup` tracker and `favorite-header` render branch**

Before the `flatItems.map` call (~line 201), add:

```typescript
let inFavoriteGroup = false
```

Then in the map callback, add BEFORE the existing `if (item.type === 'provider')` check:

```typescript
if (item.type === 'favorite-header') {
    inFavoriteGroup = true
    return (
        <div
            key="favorite-header"
            className="text-[10px] font-semibold uppercase tracking-[0.05em] px-2 pt-2 pb-0.5"
            style={{ color: '#ffd700' }}
        >
            ★ Favorite models
        </div>
    )
}
```

And in the existing `if (item.type === 'provider')` branch, add:
```typescript
if (item.type === 'provider') {
    inFavoriteGroup = false
    // ... existing provider header rendering
}
```

- [ ] **Step 2: Add provider label and star button to model rows**

Replace the model row button content (lines 234-237):

```tsx
<span>{m.DisplayName}</span>
{isSelected && (
    <IconCheck size={12} />
)}
```

With:

```tsx
<span>
    {m.DisplayName}
    {inFavoriteGroup && (
        <span className="text-[9px]" style={{ color: 'var(--text-dim)', marginLeft: 4 }}>
            ({item.provider.display_name})
        </span>
    )}
</span>
<div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
    <span
        onClick={(e) => {
            e.stopPropagation()
            toggleFavoriteModel(item.provider.id, m.ID)
        }}
        className="fav-star"
        style={{
            fontSize: 14,
            cursor: 'pointer',
            color: item.isFavorite ? '#ffd700' : 'var(--text-dim)',
            opacity: item.isFavorite ? 1 : 0,
            transition: 'opacity 0.15s',
        }}
        title={item.isFavorite ? 'Remove from favorites' : 'Add to favorites'}
    >
        {item.isFavorite ? '★' : '☆'}
    </span>
    {isSelected && <IconCheck size={12} />}
</div>
```

- [ ] **Step 3: Update model button className for hover**

Add `fav-row` class to the model `<button>` className:

Current (line 221):
```tsx
className="text-[11px] w-full text-left px-2 py-1 rounded cursor-pointer flex items-center justify-between"
```

New:
```tsx
className="text-[11px] w-full text-left px-2 py-1 rounded cursor-pointer flex items-center justify-between fav-row"
```

- [ ] **Step 4: Add hover CSS style tag**

At the top of the return block, inside the first `<div ref={ref}>`, add:

```tsx
<style>{`
.fav-row:hover .fav-star { opacity: 1 !important; }
`}</style>
```

- [ ] **Step 5: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/Chat/ModelPicker.tsx
git commit -m "feat: add favorite-header, star button, and hover CSS to ModelPicker"
```

---

### Task 5: End-to-end verification

**Files:**
- No file changes, verification only

- [ ] **Step 1: Full TypeScript check**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS with no errors

- [ ] **Step 2: Review feature completeness**

- [ ] `favoriteModels` loaded from localStorage on store init
- [ ] `toggleFavoriteModel` adds/removes `"providerId:modelId"` and writes to localStorage
- [ ] `flatItems` includes `favorite-header` when favorites exist
- [ ] Favorite model rows appear at top with provider label
- [ ] Favorite models also appear in their original provider group
- [ ] Star button shows on hover for unfavorited, always visible for favorited
- [ ] Clicking star toggles favorite without selecting model
- [ ] Search filtering applies to favorites group
- [ ] Empty favorites renders no header

- [ ] **Step 3: Commit if any final tweaks were needed**

```bash
git status
```

