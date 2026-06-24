import { useState } from 'react'
import { useStore } from '../../store'
import { App } from '../../../bindings/monika'
import { IconPencilLine, IconPlay, IconRefresh, IconSkipForward, IconClose, IconClock, IconXCircle } from '../Icons'

interface QueueItemProps {
    item: {
        id: string
        text: string
        provider_id: string
        model: string
        status: string
        error?: string
        created_at: number
    }
    sessionId: string
    projectPath: string
    manualMode: boolean
    onDragStart: () => void
    onDragOver: (e: React.DragEvent) => void
    onDrop: () => void
}

export function QueueItem({ item, sessionId, projectPath, manualMode, onDragStart, onDragOver, onDrop }: QueueItemProps) {
    const [editing, setEditing] = useState(false)
    const [editText, setEditText] = useState(item.text)
    const removeQueueItem = useStore((s) => s.removeQueueItem)

    const statusColor =
        item.status === 'executing' ? 'var(--accent)' :
            item.status === 'error' ? 'var(--red)' :
                'var(--yellow)'

    const statusIcon =
        item.status === 'executing'
            ? <span className="inline-block w-3 h-3 border-2 border-[var(--accent)] border-t-transparent rounded-full animate-spin" />
            : item.status === 'error'
                ? <IconXCircle size={13} />
                : <IconClock size={13} />
    const handleSaveEdit = async () => {
        try {
            await App.EditQueueItem(projectPath, sessionId, item.id, editText)
            setEditing(false)
        } catch (err) {
            console.error('Failed to edit queue item:', err)
        }
    }

    const handleCancel = async () => {
        try {
            await App.CancelQueueItem(projectPath, sessionId, item.id)
            if (item.status !== 'executing') {
                removeQueueItem(sessionId, item.id)
            }
        } catch (err) {
            console.error('Failed to cancel queue item:', err)
        }
    }

    const handleExecute = async () => {
        try {
            await App.ExecuteQueueItem(projectPath, sessionId, item.id)
        } catch (err) {
            console.error('Failed to execute queue item:', err)
        }
    }

    const handleRetry = async () => {
        try {
            await App.RetryQueueItem(projectPath, sessionId, item.id)
        } catch (err) {
            console.error('Failed to retry queue item:', err)
        }
    }

    const handleSkip = async () => {
        try {
            await App.SkipQueueItem(projectPath, sessionId, item.id)
            removeQueueItem(sessionId, item.id)
        } catch (err) {
            console.error('Failed to skip queue item:', err)
        }
    }

    const canEdit = item.status === 'queued' || item.status === 'error'
    const canDrag = item.status !== 'executing'

    const iconBtnClass = 'flex items-center justify-center w-5 h-5 rounded hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)] transition-colors'

    return (
        <div
            className="group flex items-start gap-2 rounded border p-1.5 text-[12px]"
            style={{
                borderColor: 'var(--border)',
                background: 'var(--bg-elevated)',
            }}
            draggable={canDrag}
            onDragStart={onDragStart}
            onDragOver={onDragOver}
            onDrop={onDrop}
        >
            {canDrag && <span className="cursor-grab select-none" style={{ color: 'var(--text-dim)' }}>⠿</span>}
            <span style={{ color: statusColor }}>{statusIcon}</span>
            <div className="flex-1 min-w-0">
                {editing ? (
                    <div className="flex flex-col gap-1">
                        <textarea
                            className="w-full rounded p-1 text-[12px] border outline-none"
                            style={{
                                background: 'var(--bg-sidebar)',
                                color: 'var(--text-primary)',
                                borderColor: 'var(--border)',
                            }}
                            value={editText}
                            onChange={(e) => setEditText(e.target.value)}
                            rows={2}
                        />
                        <div className="flex gap-2">
                            <button className="text-[11px] hover:underline" style={{ color: 'var(--green)' }} onClick={handleSaveEdit}>Save</button>
                            <button className="text-[11px] hover:underline" style={{ color: 'var(--text-dim)' }} onClick={() => { setEditText(item.text); setEditing(false) }}>Cancel</button>
                        </div>
                    </div>
                ) : (
                    <>
                        <p className="truncate" style={{ color: 'var(--text-primary)' }}>{item.text}</p>
                        {item.status === 'error' && item.error && (
                            <p className="text-[10px] mt-0.5" style={{ color: 'var(--red)' }}>{item.error}</p>
                        )}
                    </>
                )}
            </div>
            {!editing && (
                <div className="flex items-center gap-0.5 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                    {manualMode && item.status === 'queued' && (
                        <button
                            className={iconBtnClass}
                            style={{ color: 'var(--text-dim)' }}
                            onClick={(e) => { e.stopPropagation(); handleExecute() }}
                            title="Run"
                        >
                            <IconPlay size={13} />
                        </button>
                    )}
                    {canEdit && (
                        <button
                            className={iconBtnClass}
                            style={{ color: 'var(--text-dim)' }}
                            onClick={(e) => { e.stopPropagation(); setEditing(true) }}
                            title="Edit"
                        >
                            <IconPencilLine size={13} />
                        </button>
                    )}
                    {item.status === 'error' && (
                        <>
                            <button
                                className={iconBtnClass}
                                style={{ color: 'var(--text-dim)' }}
                                onClick={(e) => { e.stopPropagation(); handleRetry() }}
                                title="Retry"
                            >
                                <IconRefresh size={13} />
                            </button>
                            <button
                                className={iconBtnClass}
                                style={{ color: 'var(--text-dim)' }}
                                onClick={(e) => { e.stopPropagation(); handleSkip() }}
                                title="Skip"
                            >
                                <IconSkipForward size={13} />
                            </button>
                        </>
                    )}
                    <button
                        className={iconBtnClass}
                        style={{ color: 'var(--text-dim)' }}
                        onClick={(e) => { e.stopPropagation(); handleCancel() }}
                        title="Cancel"
                    >
                        <IconClose size={13} />
                    </button>
                </div>
            )}
        </div>
    )
}
