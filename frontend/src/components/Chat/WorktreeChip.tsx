import { useStore } from '../../store'

interface WorktreeChipProps {
    sessionId: string
    onClick: () => void
}

export default function WorktreeChip({ sessionId, onClick }: WorktreeChipProps) {
    const worktreePath = useStore((s) => s.sessionWorktrees[sessionId])

    if (!worktreePath) return null

    // Extract branch name from the worktree path
    // Path format: .../.worktrees/<branch>-<sessionHash> or .../<branch>
    const parts = worktreePath.replace(/\\/g, '/').split('/')
    const dirName = parts[parts.length - 1]

    // Try to strip trailing session hash (e.g., "feature-x-a1b2c3d4" → "feature-x")
    const branchDisplay = sessionId.length >= 8 && dirName.endsWith('-' + sessionId.slice(0, 8))
        ? dirName.slice(0, -(sessionId.slice(0, 8).length + 1))
        : dirName

    return (
        <button
            onClick={onClick}
            className="text-[11px] px-2 py-0.5 rounded cursor-pointer flex items-center gap-1"
            style={{
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border)',
                color: 'var(--text-primary)',
                fontFamily: 'inherit',
            }}
            title={`Worktree: ${worktreePath}`}
        >
            <span>Worktree: {branchDisplay}</span>
        </button>
    )
}
