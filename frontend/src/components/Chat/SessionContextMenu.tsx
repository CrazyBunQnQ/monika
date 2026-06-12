import { createPortal } from 'react-dom'
import { useStore } from '../../store'

interface SessionContextMenuProps {
    sessionId: string
    x: number
    y: number
    onClose: () => void
    onManageWorktree: () => void
}

export default function SessionContextMenu({ sessionId, x, y, onClose, onManageWorktree }: SessionContextMenuProps) {
    const worktreePath = useStore((s) => s.sessionWorktrees[sessionId])
    const detachSessionWorktree = useStore((s) => s.setSessionWorktree)

    const handleDetach = async () => {
        try {
            const { App } = await import('../../../bindings/monika')
            await App.DetachWorktree(sessionId)
            detachSessionWorktree(sessionId, '')
        } catch { /* ignore */ }
        onClose()
    }

    return createPortal(
        <div
            className="fixed inset-0"
            style={{ zIndex: 2000 }}
            onClick={onClose}
            onContextMenu={(e) => { e.preventDefault(); onClose() }}
        >
            <div
                onClick={(e) => e.stopPropagation()}
                style={{
                    position: 'absolute',
                    left: x,
                    top: y,
                    background: 'var(--bg-elevated)',
                    border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-md)',
                    padding: '4px 0',
                    minWidth: '200px',
                    boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
                    fontSize: '12px',
                    fontFamily: 'var(--font-sans)',
                }}
            >
                <div
                    className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
                    style={{ color: 'var(--text-secondary)' }}
                    onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--text-primary)' }}
                    onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-secondary)' }}
                    onClick={() => { onManageWorktree(); onClose() }}
                >
                    <span>Manage Worktree...</span>
                </div>
                {worktreePath && (
                    <>
                        <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />
                        <div
                            className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
                            style={{ color: 'var(--red)' }}
                            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--red)' }}
                            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--red)' }}
                            onClick={handleDetach}
                        >
                            <span>Detach Worktree</span>
                        </div>
                    </>
                )}
            </div>
        </div>,
        document.body
    )
}
