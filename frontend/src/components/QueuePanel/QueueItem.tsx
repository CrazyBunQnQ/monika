import { useState } from 'react'
import { useStore } from '../../store'
import { App } from '../../../bindings/monika'

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
        item.status === 'executing' ? '🔄' :
        item.status === 'error' ? '❌' :
        '⏳'

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

    return (
        <div
            className="flex items-start gap-2 rounded border p-1.5 text-[12px]"
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
                            className="w-full rounded p-1 text-[12px] border"
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
                        <div className="flex gap-2 mt-0.5">
                            {manualMode && item.status === 'queued' && (
                                <button className="text-[10px] hover:underline" style={{ color: 'var(--accent)' }} onClick={handleExecute}>▶ Run</button>
                            )}
                            {canEdit && (
                                <button className="text-[10px] hover:underline" style={{ color: 'var(--accent)' }} onClick={() => setEditing(true)}>Edit</button>
                            )}
                            {item.status === 'error' && (
                                <>
                                    <button className="text-[10px] hover:underline" style={{ color: 'var(--green)' }} onClick={handleRetry}>Retry</button>
                                    <button className="text-[10px] hover:underline" style={{ color: 'var(--yellow)' }} onClick={handleSkip}>Skip</button>
                                </>
                            )}
                            <button className="text-[10px] hover:underline" style={{ color: 'var(--red)' }} onClick={handleCancel}>Cancel</button>
                        </div>
                    </>
                )}
            </div>
        </div>
    )
}
