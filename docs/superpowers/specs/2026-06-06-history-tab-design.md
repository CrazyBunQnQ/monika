# HISTORY Tab Design

## Summary

Add a HISTORY sub-tab to the existing CHANGES panel, turning it into a "Git Panel" with two tabs: CHANGES (existing) and HISTORY (new). The HISTORY tab displays a graphical git log showing both local and remote commits.

## Requirements

- Display git commit history with: short hash, author, date, commit message
- Show graphical branch topology (like `git log --graph`)
- Distinguish local and remote refs visually
- Keep CHANGES functionality unchanged
- No dockview layout changes

## Data Layer (Go Backend)

### New Type: `CommitInfo`

Added to `internal/api/types.go`:

```go
type CommitInfo struct {
    Hash      string `json:"hash"`
    Author    string `json:"author"`
    Date      string `json:"date"`
    Message   string `json:"message"`
    Refs      string `json:"refs"`
    GraphLine string `json:"graph_line"`
}
```

- `Hash`: short commit hash (7 chars)
- `Author`: committer name
- `Date`: relative date like "2 hours ago"
- `Message`: first line of commit message
- `Refs`: ref decorations, e.g. `HEAD -> main, origin/main, tag: v1.0`
- `GraphLine`: the `--graph` ASCII line for this commit, e.g. `| * |`

### New API Method

`GitLog(projectPath string) ([]CommitInfo, error)` on the `App` struct.

Implementation: execute `git log --graph --all --decorate --oneline` with a custom format string to extract hash, author, date, message, and refs per line. Parse the graph prefix and the structured data separately. Return up to 200 most recent commits.

Uses `git` CLI directly (not go-git) because `--graph` output is specific to the CLI and parsing it reliably from go-git would require re-implementing graph rendering.

## Frontend

### Component Restructure

`ChangesList` becomes `GitPanel` — a container with a tab bar:

```
GitPanel
├── Tab bar: [ CHANGES | HISTORY ]
├── CHANGES tab → existing change stats list (unchanged logic)
└── HISTORY tab → commit history list
    └── Per row: graph (monospace) | hash | refs tags | author | date | message
```

### Ref Display Conventions

- `HEAD -> branch-name`: bold text, indicates current local branch
- `origin/xxx`: colored tag, indicates remote ref
- `tag: xxx`: distinct colored tag, indicates annotated/lightweight tag

### Data Flow

1. User switches to HISTORY tab
2. Frontend calls `App.GitLog(projectPath)`
3. Backend executes git log, parses output, returns `CommitInfo[]`
4. Frontend renders the list

### Refresh Strategy

- Load on first tab switch
- Re-load when switching back to HISTORY tab
- Optional: re-load after agent bash execution completes (piggyback on existing `emitBranchChangeIfChanged` mechanism)

### Store Changes

Add to Zustand store (`frontend/src/store/index.ts`):

- `commitHistory`: `{ loading: boolean; commits: CommitInfo[]; error: string | null }`
- `loadCommitHistory()`: action that calls `App.GitLog` and updates state

## Files Changed

| File | Change |
|------|--------|
| `internal/api/types.go` | Add `CommitInfo` struct |
| `internal/api/app.go` | Add `GitLog` method |
| `frontend/bindings/monika/...` | Auto-regenerated via `wails3 generate bindings` |
| `frontend/src/components/ChangesList/ChangesList.tsx` | Refactor to GitPanel with tab switching + HISTORY view |
| `frontend/src/store/index.ts` | Add `commitHistory` state and `loadCommitHistory` action |

No changes to: `App.tsx`, `defaultLayout.ts`, or any dockview configuration.

## Out of Scope (Future)

- Clicking a commit to view its diff
- Right-click context menu (checkout / cherry-pick / revert)
- Search/filter commits by author or message
- Auto-polling for new commits
