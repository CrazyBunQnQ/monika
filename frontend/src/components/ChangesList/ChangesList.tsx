import { useState, useEffect, useRef, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { IDockviewPanelProps } from 'dockview'
import { App as MonikaApp } from '../../../bindings/monika'
import type { ChangeStat, CommitInfo } from '../../../bindings/monika'
import { useStore } from '../../store'
import { IconFile } from '../Icons'
import { GitBranch } from 'lucide-react'

function useEffectiveChangesPath() {
    const projectPath = useStore((s) => s.projectPath)
    const activeSessionId = useStore((s) => s.activeSessionId)
    const sessionWorktrees = useStore((s) => s.sessionWorktrees)
    return useMemo(() => {
        const wt = activeSessionId ? sessionWorktrees[activeSessionId] : undefined
        return wt || projectPath
    }, [activeSessionId, sessionWorktrees, projectPath])
}

function ChangesList(_props: IDockviewPanelProps) {
    const [activeTab, setActiveTab] = useState<'changes' | 'history'>('changes')
    const effectivePath = useEffectiveChangesPath()
    const projectPath = useStore((s) => s.projectPath)
    const isWorktree = effectivePath !== projectPath

    // Extract branch name from worktree path
    const branchDisplay = useMemo(() => {
        if (!isWorktree) return null
        const parts = effectivePath.replace(/\\/g, '/').split('/')
        return parts[parts.length - 1]
    }, [effectivePath, isWorktree])

    return (
        <div
            className="flex flex-col h-full"
            style={{ background: 'var(--bg-sidebar)' }}
        >
            {/* Tab bar */}
            <div
                className="flex items-center gap-1 px-2 border-b border-[var(--border)] shrink-0 select-none"
                style={{ background: 'var(--bg-sidebar)', height: '30px' }}
            >
                <TabButton
                    label="CHANGES"
                    active={activeTab === 'changes'}
                    onClick={() => setActiveTab('changes')}
                />
                <TabButton
                    label="HISTORY"
                    active={activeTab === 'history'}
                    onClick={() => setActiveTab('history')}
                />
            </div>

            {/* Worktree path bar */}
            <div
                className="flex items-center gap-1.5 text-[12px] select-none shrink-0 border-b border-[var(--border)]"
                style={{ fontFamily: 'var(--font-sans)', padding: '4px 10px', background: 'var(--bg-sidebar)' }}
            >
                <GitBranch size={12} style={{ flexShrink: 0, color: isWorktree ? 'var(--accent)' : 'var(--text-dim)' }} />
                <span
                    className="truncate"
                    style={{ color: isWorktree ? 'var(--text-primary)' : 'var(--text-dim)', fontSize: '11px' }}
                    title={isWorktree ? effectivePath : projectPath}
                >
                    {isWorktree ? branchDisplay : 'Main'}
                </span>
            </div>

            {/* Tab content */}
            {/* Tab content — always mount both so background tab stays alive */}
            <div style={{ display: activeTab === 'changes' ? 'flex' : 'none', flex: 1, overflow: 'hidden' }}>
                <ChangesTab effectivePath={effectivePath} />
            </div>
            <div style={{ display: activeTab === 'history' ? 'flex' : 'none', flex: 1, overflow: 'hidden' }}>
                <HistoryTab active={activeTab === 'history'} effectivePath={effectivePath} />
            </div>
        </div>
    )
}

function TabButton({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
    return (
        <button
            className="text-[11px] px-2 py-1 cursor-pointer transition-colors rounded"
            style={{
                fontFamily: 'var(--font-sans)',
                color: active ? 'var(--text-primary)' : 'var(--text-dim)',
                background: active ? 'var(--bg-active)' : 'transparent',
                border: 'none',
                borderBottom: active ? '2px solid var(--accent)' : '2px solid transparent',
                fontWeight: 500,
            }}
            onMouseEnter={(e) => { if (!active) { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--text-secondary)' } }}
            onMouseLeave={(e) => { if (!active) { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-dim)' } }}
            onClick={onClick}
        >
            {label}
        </button>
    )
}

function ChangesTab({ effectivePath }: { effectivePath: string }) {
    const changes = useStore((s) => s.changeStats)
    const setPreviewDiff = useStore((s) => s.setPreviewDiff)
    const setPreviewFile = useStore((s) => s.setPreviewFile)
    const setRevealFilePath = useStore((s) => s.setRevealFilePath)
    const selectedPath = useStore((s) => s.preview.mode === 'diff' ? s.preview.filePath : null)
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; path: string } | null>(null)
    const menuRef = useRef<HTMLDivElement>(null)
    const contextMenuJustOpened = useRef(false)

    useEffect(() => {
        if (!contextMenu) return
        contextMenuJustOpened.current = true
        const onClick = () => {
            if (contextMenuJustOpened.current) {
                contextMenuJustOpened.current = false
                return
            }
            setContextMenu(null)
        }
        window.addEventListener('click', onClick)
        return () => window.removeEventListener('click', onClick)
    }, [contextMenu])

    const handleClick = async (stat: ChangeStat) => {
        try {
            const result = await MonikaApp.GetFileDiff(effectivePath, stat.path)
            const fileName = stat.path.split('/').pop() || stat.path
            if (result && result.lines) {
                setPreviewDiff(stat.path, fileName, result.lines)
            }
        } catch {
            // ignore
        }
    }

    const handleViewSource = async (path: string) => {
        try {
            const fileName = path.split('/').pop() || path
            const result = await MonikaApp.ReadFile(effectivePath, path)
            setPreviewFile(path, fileName, result?.content || '')
            setRevealFilePath(path)
        } catch {
            // ignore
        }
    }

    const handleContextMenu = (e: React.MouseEvent, path: string) => {
        e.preventDefault()
        e.stopPropagation()
        setContextMenu({ x: e.clientX, y: e.clientY, path })
    }

    const renderContextMenu = () => {
        if (!contextMenu) return null
        return createPortal(
            <div
                ref={menuRef}
                className="fixed"
                style={{
                    left: contextMenu.x,
                    top: contextMenu.y,
                    zIndex: 2000,
                    background: 'var(--bg-elevated)',
                    border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-md)',
                    padding: '4px 0',
                    minWidth: '200px',
                    boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
                    fontSize: '12px',
                    fontFamily: 'var(--font-sans)',
                }}
                onClick={(e) => e.stopPropagation()}
            >
                <div
                    className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
                    style={{ color: 'var(--text-secondary)' }}
                    onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--text-primary)' }}
                    onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-secondary)' }}
                    onClick={() => { setContextMenu(null); handleViewSource(contextMenu.path) }}
                >
                    <span className="flex-shrink-0 flex items-center" style={{ opacity: 0.7, width: 14 }}><IconFile size={14} /></span>
                    <span>View Source File</span>
                </div>
            </div>,
            document.body
        )
    }

    const basenameFn = (p: string) => p.split('/').pop() || p

    return (
        <>
            <div className="flex-1 overflow-y-auto" style={{ padding: '0 8px' }}>
                {changes.loading && changes.stats.length === 0 ? (
                    <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">Loading...</div>
                ) : changes.error && changes.stats.length === 0 ? (
                    <div className="py-4 text-[12px] text-[var(--red)] px-1">{changes.error}</div>
                ) : changes.stats.length === 0 ? (
                    <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">No changes</div>
                ) : (
                    changes.stats.map((stat) => {
                        const active = selectedPath === stat.path
                        return (
                            <div
                                key={stat.path}
                                className="flex items-center gap-1 cursor-pointer text-[13px] leading-[26px] rounded-md transition-colors duration-100 mx-1 px-[6px]"
                                style={{
                                    color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
                                    background: active ? 'var(--bg-active)' : 'transparent',
                                }}
                                onMouseEnter={(e) => {
                                    if (!active) e.currentTarget.style.background = 'rgba(255,255,255,0.04)'
                                }}
                                onMouseLeave={(e) => {
                                    e.currentTarget.style.background = active ? 'var(--bg-active)' : 'transparent'
                                }}
                                onClick={() => handleClick(stat)}
                                onContextMenu={(e) => handleContextMenu(e, stat.path)}
                                title={stat.path}
                            >
                                <span className="truncate flex-1">{basenameFn(stat.path)}</span>
                                {stat.added > 0 && (
                                    <span className="text-[11px] flex-shrink-0" style={{ color: 'var(--green)' }}>
                                        +{stat.added}
                                    </span>
                                )}
                                {stat.deleted > 0 && (
                                    <span className="text-[11px] flex-shrink-0 ml-0.5" style={{ color: 'var(--red)' }}>
                                        -{stat.deleted}
                                    </span>
                                )}
                            </div>
                        )
                    })
                )}
            </div>
            {renderContextMenu()}
        </>
    )
}

function HistoryTab({ active, effectivePath }: { active: boolean; effectivePath: string }) {
    const commitHistory = useStore((s) => s.commitHistory)
    const loadCommitHistory = useStore((s) => s.loadCommitHistory)

    // Load immediately on activation
    useEffect(() => {
        if (!active) return
        loadCommitHistory(effectivePath)
    }, [active, effectivePath]) // eslint-disable-line react-hooks/exhaustive-deps

    if (commitHistory.loading && commitHistory.commits.length === 0) {
        return (
            <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}>
                <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">Loading...</div>
            </div>
        )
    }

    if (commitHistory.error && commitHistory.commits.length === 0) {
        return (
            <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}>
                <div className="py-4 text-[12px] text-[var(--red)] px-1">{commitHistory.error}</div>
            </div>
        )
    }

    if (commitHistory.commits.length === 0) {
        return (
            <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}>
                <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">No commits</div>
            </div>
        )
    }

    return (
        <div className="flex-1 overflow-y-auto" style={{ padding: '0 8px' }}>
            {commitHistory.commits.map((commit, idx) => (
                <CommitRow key={commit.hash + '-' + idx} commit={commit} />
            ))}
        </div>
    )
}

function CommitRow({ commit }: { commit: CommitInfo }) {
    return (
        <div
            className="flex items-center gap-1 text-[12px] leading-[22px] rounded-md transition-colors duration-150 px-1 cursor-pointer hover:bg-[var(--bg-hover)]"
            style={{
                fontFamily: 'var(--font-sans)',
                color: 'var(--text-secondary)',
            }}
        >
            {/* Graph column */}
            <span
                className="flex-shrink-0 select-none"
                style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-dim)',
                    whiteSpace: 'pre',
                    lineHeight: '22px',
                    fontSize: '11px',
                }}
            >
                {commit.graph_line}
            </span>

            {/* Hash */}
            <span
                className="flex-shrink-0"
                style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)', fontSize: '11px', width: '8ch' }}
            >
                {commit.hash.slice(0, 7)}
            </span>

            {/* Refs */}
            {commit.refs && <RefTags refs={commit.refs} />}

            {/* Message */}
            <span className="truncate flex-1 min-w-0" style={{ color: 'var(--text-primary)' }}>
                {commit.message}
            </span>

            {/* Author + date */}
            <span className="flex-shrink-0 flex items-center" style={{ color: 'var(--text-dim)', fontSize: '11px', width: '25ch' }}>
                <span>{commit.author}</span>
                <span className="ml-auto">{commit.date}</span>
            </span>
        </div>
    )
}

function RefTags({ refs }: { refs: string }) {
    const parts = refs.split(',').map((s) => s.trim()).filter(Boolean)
    return (
        <span className="flex items-center gap-1 flex-shrink-0">
            {parts.map((ref) => {
                let color = 'var(--text-dim)'
                let bg = 'var(--bg-sidebar)'
                if (ref.startsWith('tag:')) {
                    color = '#f0c040'
                    bg = 'rgba(240,192,64,0.1)'
                } else if (ref.startsWith('HEAD')) {
                    color = 'var(--accent)'
                    bg = 'rgba(100,150,255,0.1)'
                } else if (ref.startsWith('origin/')) {
                    color = 'var(--green)'
                    bg = 'rgba(80,200,120,0.1)'
                }
                return (
                    <span
                        key={ref}
                        style={{
                            color,
                            background: bg,
                            borderRadius: '3px',
                            padding: '0 4px',
                            fontSize: '10px',
                            lineHeight: '18px',
                            whiteSpace: 'nowrap',
                        }}
                    >
                        {ref}
                    </span>
                )
            })}
        </span>
    )
}

export default ChangesList
