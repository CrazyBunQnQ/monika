import { useState, useEffect, useRef, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { IDockviewPanelProps } from 'dockview'
import { App as MonikaApp } from '../../../bindings/monika'
import type { ChangeStat, CommitInfo } from '../../../bindings/monika'
import { useStore } from '../../store'
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
    const stageFiles = useStore((s) => s.stageFiles)
    const unstageFiles = useStore((s) => s.unstageFiles)
    const commitChanges = useStore((s) => s.commitChanges)
    const feedback = useStore((s) => s.feedback)
    const clearFeedback = useStore((s) => s.clearFeedback)
    const [selectedUnstaged, setSelectedUnstaged] = useState<Set<string>>(new Set())
    const [selectedStaged, setSelectedStaged] = useState<Set<string>>(new Set())
    const [commitMsg, setCommitMsg] = useState('')
    const [committing, setCommitting] = useState(false)

    useEffect(() => {
        if (!feedback.message) return
        const t = setTimeout(() => clearFeedback(), 4000)
        return () => clearTimeout(t)
    }, [feedback.message])

    const unstaged = changes.stats.filter(s => !s.staged)
    const staged = changes.stats.filter(s => s.staged)

    const toggleUnstaged = (path: string) => {
        setSelectedUnstaged(prev => {
            const next = new Set(prev)
            if (next.has(path)) next.delete(path); else next.add(path)
            return next
        })
    }

    const toggleStaged = (path: string) => {
        setSelectedStaged(prev => {
            const next = new Set(prev)
            if (next.has(path)) next.delete(path); else next.add(path)
            return next
        })
    }

    const handleStage = async () => {
        if (selectedUnstaged.size === 0) return
        const paths = Array.from(selectedUnstaged)
        await stageFiles(paths)
        setSelectedUnstaged(new Set())
    }

    const handleUnstage = async () => {
        if (selectedStaged.size === 0) return
        await unstageFiles(Array.from(selectedStaged))
        setSelectedStaged(new Set())
    }

    const handleCommit = async (push: boolean) => {
        if (!commitMsg.trim() || staged.length === 0) return
        setCommitting(true)
        await commitChanges(commitMsg, push)
        setCommitting(false)
        setCommitMsg('')
    }

    const handleFileClick = async (stat: ChangeStat) => {
        try {
            const fileName = stat.path.split('/').pop() || stat.path
            if (stat.staged) {
                const result = await MonikaApp.GetStagedFileDiff(effectivePath, stat.path)
                if (result && result.lines) {
                    setPreviewDiff(stat.path, fileName, result.lines)
                }
            } else {
                const result = await MonikaApp.GetFileDiff(effectivePath, stat.path)
                if (result && result.lines) {
                    setPreviewDiff(stat.path, fileName, result.lines)
                }
            }
        } catch {
            // ignore
        }
    }

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && e.ctrlKey && e.shiftKey) {
            e.preventDefault()
            handleCommit(true)
        } else if (e.key === 'Enter' && e.ctrlKey) {
            e.preventDefault()
            handleCommit(false)
        }
    }

    const basenameFn = (p: string) => p.split('/').pop() || p

    return (
        <div className="flex flex-col flex-1 overflow-hidden">
            {feedback.message && (
                <div
                    className="shrink-0 px-2 py-1 text-[11px]"
                    style={{
                        color: feedback.type === 'error' ? 'var(--red)' :
                            feedback.type === 'success' ? 'var(--green)' : 'var(--text-dim)',
                        background: feedback.type === 'error' ? 'rgba(255,50,50,0.1)' :
                            feedback.type === 'success' ? 'rgba(0,200,100,0.1)' : 'transparent',
                    }}
                >
                    {feedback.message}
                </div>
            )}

            <div className="flex-1 overflow-y-auto" style={{ minHeight: 0 }}>
                <div className="flex items-center justify-between px-2 py-1 text-[11px] text-[var(--text-dim)] sticky top-0" style={{ background: 'var(--bg-sidebar)' }}>
                    <span>Unstaged ({unstaged.length})</span>
                    {selectedUnstaged.size > 0 && (
                        <button
                            className="px-2 py-0.5 rounded text-[11px] cursor-pointer"
                            style={{ background: 'var(--bg-active)', color: 'var(--accent)', border: 'none' }}
                            onClick={handleStage}
                        >Stage ▶</button>
                    )}
                </div>
                {unstaged.length === 0 ? (
                    <div className="py-2 text-[11px] text-[var(--text-dim)] px-2">
                        {changes.loading ? 'Loading...' : 'No unstaged changes'}
                    </div>
                ) : (
                    unstaged.map((stat) => {
                        const selected = selectedUnstaged.has(stat.path)
                        return (
                            <div
                                key={'un-' + stat.path}
                                className="flex items-center gap-1 cursor-pointer text-[12px] leading-[24px] rounded-md transition-colors mx-1 px-[6px]"
                                style={{ color: 'var(--text-secondary)', background: selected ? 'var(--bg-active)' : 'transparent' }}
                                onDoubleClick={() => { stageFiles([stat.path]); }}
                            >
                                <input
                                    type="checkbox"
                                    checked={selected}
                                    onChange={() => toggleUnstaged(stat.path)}
                                    onClick={(e) => e.stopPropagation()}
                                    style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
                                />
                                <span className="truncate flex-1" onClick={() => handleFileClick(stat)} title={stat.path}>
                                    {basenameFn(stat.path)}
                                </span>
                                {stat.added > 0 && <span className="text-[11px] flex-shrink-0" style={{ color: 'var(--green)' }}>+{stat.added}</span>}
                                {stat.deleted > 0 && <span className="text-[11px] flex-shrink-0 ml-0.5" style={{ color: 'var(--red)' }}>-{stat.deleted}</span>}
                            </div>
                        )
                    })
                )}

                <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />

                <div className="flex items-center justify-between px-2 py-1 text-[11px] text-[var(--text-dim)] sticky top-0" style={{ background: 'var(--bg-sidebar)', zIndex: 1 }}>
                    <span>Staged ({staged.length})</span>
                    {selectedStaged.size > 0 && (
                        <button
                            className="px-2 py-0.5 rounded text-[11px] cursor-pointer"
                            style={{ background: 'var(--bg-active)', color: 'var(--accent)', border: 'none' }}
                            onClick={handleUnstage}
                        >◀ Unstage</button>
                    )}
                </div>
                {staged.length === 0 ? (
                    <div className="py-2 text-[11px] text-[var(--text-dim)] px-2">No staged changes</div>
                ) : (
                    staged.map((stat) => {
                        const selected = selectedStaged.has(stat.path)
                        return (
                            <div
                                key={'st-' + stat.path}
                                className="flex items-center gap-1 cursor-pointer text-[12px] leading-[24px] rounded-md transition-colors mx-1 px-[6px]"
                                style={{ color: 'var(--text-secondary)', background: selected ? 'var(--bg-active)' : 'transparent' }}
                                onDoubleClick={() => { unstageFiles([stat.path]); }}
                            >
                                <input
                                    type="checkbox"
                                    checked={selected}
                                    onChange={() => toggleStaged(stat.path)}
                                    onClick={(e) => e.stopPropagation()}
                                    style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
                                />
                                <span className="truncate flex-1" onClick={() => handleFileClick(stat)} title={stat.path}>
                                    {basenameFn(stat.path)}
                                </span>
                                {stat.added > 0 && <span className="text-[11px] flex-shrink-0" style={{ color: 'var(--green)' }}>+{stat.added}</span>}
                                {stat.deleted > 0 && <span className="text-[11px] flex-shrink-0 ml-0.5" style={{ color: 'var(--red)' }}>-{stat.deleted}</span>}
                            </div>
                        )
                    })
                )}
            </div>

            <div className="shrink-0 border-t border-[var(--border)]" style={{ padding: '8px' }}>
                <textarea
                    className="w-full rounded-md text-[12px]"
                    style={{
                        background: 'var(--bg-input, var(--bg-sidebar))',
                        border: '1px solid var(--border)',
                        color: 'var(--text-primary)',
                        fontFamily: 'var(--font-sans)',
                        padding: '6px 8px',
                        resize: 'none',
                        height: '50px',
                        outline: 'none',
                    }}
                    placeholder="Commit message..."
                    value={commitMsg}
                    onChange={(e) => setCommitMsg(e.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={committing}
                />
                <div className="flex gap-2 mt-2">
                    <button
                        className="flex-1 px-3 py-1.5 rounded-md text-[12px] font-medium cursor-pointer transition-colors"
                        style={{
                            background: staged.length > 0 && commitMsg.trim() ? 'var(--accent)' : 'var(--bg-hover)',
                            color: staged.length > 0 && commitMsg.trim() ? '#fff' : 'var(--text-dim)',
                            border: 'none',
                            opacity: staged.length > 0 && commitMsg.trim() ? 1 : 0.5,
                        }}
                        onClick={() => handleCommit(false)}
                        disabled={committing || staged.length === 0 || !commitMsg.trim()}
                    >{committing ? 'Committing...' : 'Commit'}</button>
                    <button
                        className="flex-1 px-3 py-1.5 rounded-md text-[12px] font-medium cursor-pointer transition-colors"
                        style={{
                            background: staged.length > 0 && commitMsg.trim() ? 'var(--accent)' : 'var(--bg-hover)',
                            color: staged.length > 0 && commitMsg.trim() ? '#fff' : 'var(--text-dim)',
                            border: 'none',
                            opacity: staged.length > 0 && commitMsg.trim() ? 1 : 0.5,
                        }}
                        onClick={() => handleCommit(true)}
                        disabled={committing || staged.length === 0 || !commitMsg.trim()}
                    >{committing ? 'Pushing...' : 'Commit & Push'}</button>
                </div>
            </div>
        </div>
    )
}

function HistoryTab({ active, effectivePath }: { active: boolean; effectivePath: string }) {
    const commitHistory = useStore((s) => s.commitHistory)
    const loadCommitHistory = useStore((s) => s.loadCommitHistory)
    const setPreviewCommit = useStore((s) => s.setPreviewCommit)
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; commit: CommitInfo } | null>(null)
    const menuRef = useRef<HTMLDivElement>(null)
    const menuJustOpened = useRef(false)

    useEffect(() => {
        if (!active) return
        loadCommitHistory(effectivePath)
    }, [active, effectivePath]) // eslint-disable-line react-hooks/exhaustive-deps

    useEffect(() => {
        if (!contextMenu) return
        menuJustOpened.current = true
        const onClick = () => {
            if (menuJustOpened.current) { menuJustOpened.current = false; return }
            setContextMenu(null)
        }
        window.addEventListener('click', onClick)
        return () => window.removeEventListener('click', onClick)
    }, [contextMenu])

    const handleClick = (commit: CommitInfo) => {
        setPreviewCommit(commit.hash)
    }

    const handleContextMenu = (e: React.MouseEvent, commit: CommitInfo) => {
        e.preventDefault()
        e.stopPropagation()
        setContextMenu({ x: e.clientX, y: e.clientY, commit })
    }

    const handleCopyHash = (commit: CommitInfo) => {
        navigator.clipboard.writeText(commit.hash)
        setContextMenu(null)
    }

    const handleCopyMessage = (commit: CommitInfo) => {
        navigator.clipboard.writeText(commit.message)
        setContextMenu(null)
    }

    const handleCheckoutCommit = async (c: CommitInfo) => {
        setContextMenu(null)
        try {
            await MonikaApp.CheckoutCommit(effectivePath, c.hash)
            loadCommitHistory()
        } catch { /* feedback handled by store */ }
    }

    const handleCreateTag = async (c: CommitInfo) => {
        setContextMenu(null)
        const name = prompt('Tag name:')
        if (!name) return
        try {
            await MonikaApp.CreateTag(effectivePath, c.hash, name)
        } catch { /* ignore */ }
    }

    const handleCreateBranch = async (c: CommitInfo) => {
        setContextMenu(null)
        const name = prompt('Branch name:')
        if (!name) return
        try {
            await MonikaApp.CreateBranchAt(effectivePath, c.hash, name)
        } catch { /* ignore */ }
    }

    const renderContextMenu = () => {
        if (!contextMenu) return null
        const c = contextMenu.commit
        return createPortal(
            <div
                ref={menuRef}
                className="fixed"
                style={{
                    left: contextMenu.x, top: contextMenu.y, zIndex: 2000,
                    background: 'var(--bg-elevated)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-md)', padding: '4px 0', minWidth: '200px',
                    boxShadow: '0 8px 24px rgba(0,0,0,0.4)', fontSize: '12px', fontFamily: 'var(--font-sans)',
                }}
                onClick={(e) => e.stopPropagation()}
            >
                <ContextMenuItem icon="◆" label="View Details" onClick={() => { setContextMenu(null); handleClick(c) }} />
                <ContextMenuDivider />
                <ContextMenuItem label="Copy Hash" onClick={() => handleCopyHash(c)} />
                <ContextMenuItem label="Copy Message" onClick={() => handleCopyMessage(c)} />
                <ContextMenuDivider />
                <ContextMenuItem label="Checkout Commit..." onClick={() => handleCheckoutCommit(c)} />
                <ContextMenuItem label="Create Tag..." onClick={() => handleCreateTag(c)} />
                <ContextMenuItem label="Create Branch at Commit..." onClick={() => handleCreateBranch(c)} />
            </div>,
            document.body
        )
    }

    if (commitHistory.loading && commitHistory.commits.length === 0) {
        return <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}><div className="py-4 text-[12px] text-[var(--text-dim)] px-1">Loading...</div></div>
    }
    if (commitHistory.error && commitHistory.commits.length === 0) {
        return <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}><div className="py-4 text-[12px] text-[var(--red)] px-1">{commitHistory.error}</div></div>
    }
    if (commitHistory.commits.length === 0) {
        return <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}><div className="py-4 text-[12px] text-[var(--text-dim)] px-1">No commits</div></div>
    }
    return (
        <>
            <div className="flex-1 overflow-y-auto" style={{ padding: '0 8px' }}>
                {commitHistory.commits.map((commit, idx) => (
                    <CommitRow
                        key={commit.hash + '-' + idx}
                        commit={commit}
                        onClick={() => handleClick(commit)}
                        onContextMenu={(e) => handleContextMenu(e, commit)}
                    />
                ))}
            </div>
            {renderContextMenu()}
        </>
    )
}

function CommitRow({ commit, onClick, onContextMenu }: { commit: CommitInfo; onClick: () => void; onContextMenu: (e: React.MouseEvent) => void }) {
    return (
        <div
            className="flex items-center gap-1 text-[12px] leading-[22px] rounded-md transition-colors duration-150 px-1 cursor-pointer hover:bg-[var(--bg-hover)]"
            style={{ fontFamily: 'var(--font-sans)', color: 'var(--text-secondary)' }}
            onClick={onClick}
            onContextMenu={onContextMenu}
        >
            <span className="flex-shrink-0 select-none" style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-dim)', whiteSpace: 'pre', lineHeight: '22px', fontSize: '11px' }}>
                {commit.graph_line}
            </span>
            <span className="flex-shrink-0" style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)', fontSize: '11px', width: '8ch' }}>
                {commit.hash.slice(0, 7)}
            </span>
            {commit.refs && <RefTags refs={commit.refs} />}
            <span className="truncate flex-1 min-w-0" style={{ color: 'var(--text-primary)' }}>{commit.message}</span>
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

function ContextMenuItem({ icon, label, onClick, variant }: { icon?: string; label: string; onClick: () => void; variant?: 'danger' }) {
    const color = variant === 'danger' ? 'var(--red)' : 'var(--text-secondary)'
    const hoverColor = variant === 'danger' ? 'var(--red)' : 'var(--text-primary)'
    return (
        <div
            className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
            style={{ color }}
            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = hoverColor }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = color }}
            onClick={onClick}
        >
            {icon && <span className="flex-shrink-0 flex items-center" style={{ opacity: 0.7, width: 14 }}>{icon}</span>}
            <span>{label}</span>
        </div>
    )
}

function ContextMenuDivider() {
    return <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />
}

export default ChangesList
