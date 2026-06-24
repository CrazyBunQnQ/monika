import { useState } from 'react'
import { useStore } from '../../store'
import { App } from '../../../bindings/monika'
import { QueueItem } from './QueueItem'

const MAX_VISIBLE = 5

export function QueuePanel() {
    const projectPath = useStore((s) => s.projectPath)
    const activeSessionId = useStore((s) => s.activeSessionId)
    const sessionQueues = useStore((s) => s.sessionQueues)
    const queuePaused = useStore((s) => s.queuePaused)
    const reorderQueue = useStore((s) => s.reorderQueue)
    const toggleQueuePause = useStore((s) => s.toggleQueuePause)

    const [expanded, setExpanded] = useState(true)
    const [showAll, setShowAll] = useState(false)
    const [dragIndex, setDragIndex] = useState<number | null>(null)

    const queue = sessionQueues[activeSessionId] || []
    const manualMode = queuePaused[activeSessionId] || false

    const hasError = queue.some((q) => q.status === 'error')
    const visibleItems = showAll ? queue : queue.slice(0, MAX_VISIBLE)

    const handleModeToggle = async () => {
        try {
            if (manualMode) {
                await App.ResumeQueue(projectPath, activeSessionId)
                toggleQueuePause(activeSessionId, false)
            } else {
                await App.PauseQueue(projectPath, activeSessionId)
                toggleQueuePause(activeSessionId, true)
            }
        } catch (err) {
            console.error('Failed to toggle mode:', err)
        }
    }

    const handleDragStart = (index: number) => () => {
        setDragIndex(index)
    }

    const handleDragOver = (e: React.DragEvent) => {
        e.preventDefault()
    }

    const handleDrop = (index: number) => () => {
        if (dragIndex === null || dragIndex === index) return
        const execIndex = queue.findIndex((q) => q.status === 'executing')
        if (execIndex >= 0 && index <= execIndex) return
        const newOrder = [...queue]
        const [moved] = newOrder.splice(dragIndex, 1)
        newOrder.splice(index, 0, moved)
        const itemIds = newOrder.map((item) => item.id)

        reorderQueue(activeSessionId, itemIds)
        setDragIndex(null)

        App.ReorderQueue(projectPath, activeSessionId, itemIds).catch(console.error)
    }

    return (
        <div
            className="rounded-md border mb-2 overflow-hidden"
            style={{
                borderColor: hasError ? 'var(--red)' : 'var(--border)',
                background: 'var(--bg-card)',
            }}
        >
            <div
                className="flex items-center gap-2 px-3 py-1.5 select-none"
                style={{ background: 'var(--bg-sidebar)' }}
            >
                <span className="text-[11px]" style={{ color: hasError ? 'var(--red)' : 'var(--text-dim)' }}>
                    {hasError ? '⚠' : '⏳'}
                </span>
                <span className="text-[12px]" style={{ color: 'var(--text-primary)' }}>
                    Queue ({queue.length})
                </span>
                {hasError && (
                    <span className="text-[10px] px-1.5 py-0.5 rounded" style={{ background: 'var(--red)', color: '#fff' }}>
                        Error
                    </span>
                )}
                <div className="flex-1" />
                <button
                    className="text-[10px] px-2 py-0.5 rounded transition-opacity hover:opacity-80"
                    style={{
                        background: manualMode ? 'var(--yellow)' : 'var(--green)',
                        color: manualMode ? '#000' : '#fff',
                        border: '1px solid var(--border)',
                    }}
                    onClick={(e) => {
                        e.stopPropagation()
                        handleModeToggle()
                    }}
                    title={manualMode ? 'Click to switch to auto mode' : 'Click to switch to manual mode'}
                >
                    {manualMode ? 'Manual Mode' : 'Auto Mode'}
                </button>
                {queue.length > 0 && (
                    <span
                        className="text-[10px] cursor-pointer"
                        style={{ color: 'var(--text-dim)' }}
                        onClick={() => setExpanded(!expanded)}
                    >
                        {expanded ? '▴' : '▾'}
                    </span>
                )}
            </div>
            {expanded && queue.length > 0 && (
                <div className="p-2 space-y-1.5 max-h-[200px] overflow-y-auto">
                    {visibleItems.map((item, index) => (
                        <QueueItem
                            key={item.id}
                            item={item}
                            sessionId={activeSessionId}
                            projectPath={projectPath}
                            manualMode={manualMode}
                            onDragStart={handleDragStart(index)}
                            onDragOver={handleDragOver}
                            onDrop={handleDrop(index)}
                        />
                    ))}
                    {queue.length > MAX_VISIBLE && (
                        <button
                            className="text-[11px] hover:underline w-full text-center"
                            style={{ color: 'var(--accent)' }}
                            onClick={(e) => {
                                e.stopPropagation()
                                setShowAll(!showAll)
                            }}
                        >
                            {showAll ? 'Collapse' : `Show All (${queue.length})`}
                        </button>
                    )}
                </div>
            )}
        </div>
    )
}
