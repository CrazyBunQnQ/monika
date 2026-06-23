import { useState, useEffect, useRef, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { IDockviewPanelProps } from 'dockview'
import { App as MonikaApp } from '../../../bindings/monika'
import type { ChangeStat, CommitInfo } from '../../../bindings/monika'
import { useStore } from '../../store'
import { GitBranch, GitCommitHorizontal, Copy, Clipboard, Tag, GitPullRequestArrow, RotateCcw, UndoDot, Pencil, Eye, Circle, CircleCheck, Upload } from 'lucide-react'
import ConfirmModal from '../Chat/ConfirmModal'
import Modal, { ModalHeader, ModalBody, ModalFooter, ModalButton } from '../ui/Modal'

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
    const canCommit = staged.length > 0 && commitMsg.trim() !== ''

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
                                <span
                                    onClick={(e) => { e.stopPropagation(); toggleUnstaged(stat.path) }}
                                    style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', flexShrink: 0 }}
                                >
                                    {selected
                                        ? <CircleCheck size={14} style={{ color: 'var(--accent)' }} strokeWidth={1.5} />
                                        : <Circle size={14} style={{ color: 'var(--text-dim)' }} strokeWidth={1.5} />}
                                </span>
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
                                <span
                                    onClick={(e) => { e.stopPropagation(); toggleStaged(stat.path) }}
                                    style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', flexShrink: 0 }}
                                >
                                    {selected
                                        ? <CircleCheck size={14} style={{ color: 'var(--accent)' }} strokeWidth={1.5} />
                                        : <Circle size={14} style={{ color: 'var(--text-dim)' }} strokeWidth={1.5} />}
                                </span>
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
                <div
                    className="rounded-md border transition-colors"
                    style={{ background: 'var(--bg-card)', borderColor: 'var(--border)' }}
                >
                    <textarea
                        className="w-full text-[12px] outline-none border-0 resize-none bg-transparent"
                        style={{
                            color: 'var(--text-primary)',
                            fontFamily: 'var(--font-sans)',
                            padding: '8px 10px',
                            height: '48px',
                        }}
                        placeholder="Commit message..."
                        value={commitMsg}
                        onChange={(e) => setCommitMsg(e.target.value)}
                        onKeyDown={handleKeyDown}
                        disabled={committing}
                    />
                    <div className="flex items-center gap-2 px-[10px] pb-[8px]">
                        {staged.length > 0 && (
                            <span className="text-[11px] text-[var(--text-dim)] select-none">
                                {staged.length} staged
                            </span>
                        )}
                        <div className="flex-1" />
                        <button
                            className="text-[11px] px-2 py-0.5 rounded cursor-pointer flex items-center gap-1 transition-colors"
                            style={{
                                background: 'var(--bg-elevated)',
                                border: '1px solid var(--border)',
                                color: canCommit ? 'var(--text-secondary)' : 'var(--text-dim)',
                                fontFamily: 'inherit',
                                opacity: canCommit ? 1 : 0.5,
                                cursor: canCommit ? 'pointer' : 'not-allowed',
                            }}
                            onMouseEnter={(e) => { if (canCommit) e.currentTarget.style.borderColor = 'var(--border-strong)' }}
                            onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'var(--border)' }}
                            onClick={() => handleCommit(false)}
                            disabled={committing || !canCommit}
                        >
                            <GitCommitHorizontal size={12} strokeWidth={1.5} />
                            {committing ? 'Committing...' : 'Commit'}
                        </button>
                        <button
                            className="text-[11px] px-2 py-0.5 rounded cursor-pointer flex items-center gap-1 transition-colors"
                            style={{
                                background: canCommit ? 'var(--accent)' : 'var(--bg-elevated)',
                                border: '1px solid var(--border)',
                                color: canCommit ? '#fff' : 'var(--text-dim)',
                                fontFamily: 'inherit',
                                opacity: canCommit ? 1 : 0.5,
                                cursor: canCommit ? 'pointer' : 'not-allowed',
                            }}
                            onMouseEnter={(e) => { if (canCommit) e.currentTarget.style.background = 'var(--accent-hover)' }}
                            onMouseLeave={(e) => { e.currentTarget.style.background = canCommit ? 'var(--accent)' : 'var(--bg-elevated)' }}
                            onClick={() => handleCommit(true)}
                            disabled={committing || !canCommit}
                        >
                            <Upload size={12} strokeWidth={1.5} />
                            {committing ? 'Pushing...' : 'Commit & Push'}
                        </button>
                    </div>
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
    const [confirmModal, setConfirmModal] = useState<{
        title: string; message: string; confirmLabel: string; variant: 'danger' | 'primary';
        onConfirm: () => Promise<void>;
    } | null>(null)
    const [inputModal, setInputModal] = useState<{
        title: string; label: string; defaultValue: string; confirmLabel: string;
        onConfirm: (value: string) => Promise<void>;
    } | null>(null)

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
        setInputModal({
            title: 'Create Tag',
            label: 'Tag name',
            defaultValue: '',
            confirmLabel: 'Create',
            onConfirm: async (name) => {
                await MonikaApp.CreateTag(effectivePath, c.hash, name)
                loadCommitHistory()
                setInputModal(null)
            },
        })
    }

    const handleCreateBranch = async (c: CommitInfo) => {
        setContextMenu(null)
        setInputModal({
            title: 'Create Branch at Commit',
            label: 'Branch name',
            defaultValue: '',
            confirmLabel: 'Create',
            onConfirm: async (name) => {
                await MonikaApp.CreateBranchAt(effectivePath, c.hash, name)
                loadCommitHistory()
                setInputModal(null)
            },
        })
    }

    const handleRevertCommit = (c: CommitInfo) => {
        setContextMenu(null)
        setConfirmModal({
            title: `Revert ${c.hash.slice(0, 7)}?`,
            message: `This will create a new commit that reverses "${c.message}".`,
            confirmLabel: 'Revert',
            variant: 'danger',
            onConfirm: async () => {
                await MonikaApp.RevertCommit(effectivePath, c.hash)
                loadCommitHistory()
                setConfirmModal(null)
            },
        })
    }

    const handleCherryPick = (c: CommitInfo) => {
        setContextMenu(null)
        setConfirmModal({
            title: 'Cherry-pick Commit?',
            message: `Apply "${c.hash.slice(0, 7)}: ${c.message}" to the current branch.`,
            confirmLabel: 'Cherry-pick',
            variant: 'primary',
            onConfirm: async () => {
                await MonikaApp.CherryPickCommit(effectivePath, c.hash)
                loadCommitHistory()
                setConfirmModal(null)
            },
        })
    }

    const handleReset = (c: CommitInfo, mode: 'soft' | 'mixed' | 'hard') => {
        setContextMenu(null)
        const descriptions: Record<string, string> = {
            soft: 'HEAD only — staged changes are kept.',
            mixed: 'HEAD + index — unstaged changes are kept (default).',
            hard: 'ALL changes will be permanently discarded.',
        }
        setConfirmModal({
            title: `Reset to ${c.hash.slice(0, 7)} (${mode})?`,
            message: descriptions[mode] + (mode === 'hard' ? ' This cannot be undone.' : ''),
            confirmLabel: `Reset ${mode}`,
            variant: mode === 'hard' ? 'danger' : 'primary',
            onConfirm: async () => {
                await MonikaApp.ResetToCommit(effectivePath, c.hash, mode)
                loadCommitHistory()
                setConfirmModal(null)
            },
        })
    }

    const handleAmendMessage = async (c: CommitInfo) => {
        setContextMenu(null)
        setInputModal({
            title: 'Amend Commit Message',
            label: 'New message',
            defaultValue: c.message,
            confirmLabel: 'Amend',
            onConfirm: async (newMsg) => {
                await MonikaApp.AmendMessage(effectivePath, newMsg)
                loadCommitHistory()
                setInputModal(null)
            },
        })
    }

    const renderContextMenu = () => {
        if (!contextMenu) return null
        const c = contextMenu.commit
        const menuItems: { label: string; icon: React.ReactNode; action: () => void; separator?: boolean; danger?: boolean }[] = [
            { label: 'View Details', icon: <Eye size={14} />, action: () => { setContextMenu(null); handleClick(c) } },
            { label: 'Copy Hash', icon: <Copy size={14} />, action: () => handleCopyHash(c), separator: true },
            { label: 'Copy Message', icon: <Clipboard size={14} />, action: () => handleCopyMessage(c) },
            { label: 'Checkout Commit...', icon: <GitCommitHorizontal size={14} />, action: () => handleCheckoutCommit(c), separator: true },
            { label: 'Create Tag...', icon: <Tag size={14} />, action: () => handleCreateTag(c) },
            { label: 'Create Branch at Commit...', icon: <GitBranch size={14} />, action: () => handleCreateBranch(c) },
            { label: 'Revert Commit', icon: <RotateCcw size={14} />, action: () => handleRevertCommit(c), separator: true },
            { label: 'Cherry-pick Commit', icon: <GitPullRequestArrow size={14} />, action: () => handleCherryPick(c) },
            { label: 'Reset to Commit (Soft)', icon: <UndoDot size={14} />, action: () => handleReset(c, 'soft'), separator: true },
            { label: 'Reset to Commit (Mixed)', icon: <UndoDot size={14} />, action: () => handleReset(c, 'mixed') },
            { label: 'Reset to Commit (Hard)', icon: <UndoDot size={14} />, action: () => handleReset(c, 'hard'), danger: true },
        ]
        if (c.hash === commitHistory.commits[0]?.hash) {
            menuItems.push({ label: 'Amend Message...', icon: <Pencil size={14} />, action: () => handleAmendMessage(c), separator: true })
        }
        return createPortal(
            <div
                ref={menuRef}
                className="fixed"
                style={{
                    left: contextMenu.x, top: contextMenu.y, zIndex: 2000,
                    background: 'var(--bg-elevated)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-md)', padding: '4px 0', minWidth: '220px',
                    boxShadow: '0 8px 24px rgba(0,0,0,0.4)', fontSize: '12px', fontFamily: 'var(--font-sans)',
                }}
                onClick={(e) => e.stopPropagation()}
            >
                {menuItems.map((item, i) => (
                    <div key={i}>
                        {item.separator && <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />}
                        <div
                            className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
                            style={{ color: item.danger ? 'var(--red)' : 'var(--text-secondary)' }}
                            onMouseEnter={(e) => { const t = e.currentTarget; t.style.background = 'var(--bg-hover)'; t.style.color = item.danger ? 'var(--red)' : 'var(--text-primary)' }}
                            onMouseLeave={(e) => { const t = e.currentTarget; t.style.background = 'transparent'; t.style.color = item.danger ? 'var(--red)' : 'var(--text-secondary)' }}
                            onClick={() => { setContextMenu(null); item.action() }}
                        >
                            <span className="flex-shrink-0 flex items-center" style={{ opacity: 0.7, width: 14 }}>{item.icon}</span>
                            <span>{item.label}</span>
                        </div>
                    </div>
                ))}
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
            {confirmModal && (
                <ConfirmModal
                    title={confirmModal.title}
                    message={confirmModal.message}
                    confirmLabel={confirmModal.confirmLabel}
                    variant={confirmModal.variant}
                    onConfirm={confirmModal.onConfirm}
                    onCancel={() => setConfirmModal(null)}
                />
            )}
            {inputModal && <InputModal config={inputModal} onCancel={() => setInputModal(null)} />}
        </>
    )
}

function CommitRow({ commit, onClick, onContextMenu }: { commit: CommitInfo; onClick: () => void; onContextMenu: (e: React.MouseEvent) => void }) {
    const [hover, setHover] = useState(false)
    const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 })
    const rowRef = useRef<HTMLDivElement>(null)

    const handleMouseEnter = () => {
        if (rowRef.current) {
            const rect = rowRef.current.getBoundingClientRect()
            setTooltipPos({ x: rect.left + 8, y: rect.bottom + 4 })
        }
        setHover(true)
    }

    return (
        <>
            <div
                ref={rowRef}
                className="flex items-center gap-1 text-[12px] leading-[22px] rounded-md transition-colors duration-150 px-1 cursor-pointer hover:bg-[var(--bg-hover)]"
                style={{ fontFamily: 'var(--font-sans)', color: 'var(--text-secondary)' }}
                onClick={onClick}
                onContextMenu={onContextMenu}
                onMouseEnter={handleMouseEnter}
                onMouseLeave={() => setHover(false)}
            >
                <GraphLine line={commit.graph_line} />
                <span className="flex-shrink-0" style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)', fontSize: '11px', width: '7ch' }}>
                    {commit.hash.slice(0, 7)}
                </span>
                {commit.refs && <RefTags refs={commit.refs} />}
                <span className="truncate flex-1 min-w-0" style={{ color: 'var(--text-primary)' }}>{commit.message}</span>
                <span className="flex-shrink-0 flex items-center gap-1.5" style={{ color: 'var(--text-dim)', fontSize: '11px', width: '22ch' }}>
                    <span className="truncate" style={{ minWidth: 0 }}>{commit.author}</span>
                    <span className="flex-shrink-0">{commit.date}</span>
                </span>
            </div>
            {hover && createPortal(
                <div
                    className="fixed"
                    style={{
                        left: tooltipPos.x,
                        top: tooltipPos.y,
                        zIndex: 1500,
                        background: 'var(--bg-elevated)',
                        border: '1px solid var(--border)',
                        borderRadius: 'var(--radius-md)',
                        padding: '8px 12px',
                        maxWidth: '480px',
                        boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
                        fontSize: '12px',
                        fontFamily: 'var(--font-sans)',
                        pointerEvents: 'none',
                    }}
                >
                    <div className="flex items-center gap-2 mb-1">
                        <span style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)', fontSize: '11px' }}>{commit.hash.slice(0, 7)}</span>
                        {commit.refs && <span style={{ color: 'var(--text-dim)', fontSize: '11px' }}>{commit.refs}</span>}
                    </div>
                    <div style={{ color: 'var(--text-primary)', whiteSpace: 'pre-wrap', wordBreak: 'break-word', lineHeight: '1.5' }}>
                        {commit.message}
                    </div>
                    <div className="flex items-center gap-2 mt-1" style={{ color: 'var(--text-dim)', fontSize: '11px' }}>
                        <span>{commit.author}</span>
                        <span>·</span>
                        <span>{commit.date}</span>
                    </div>
                </div>,
                document.body
            )}
        </>
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

function InputModal({ config, onCancel }: { config: { title: string; label: string; defaultValue: string; confirmLabel: string; onConfirm: (value: string) => Promise<void> }; onCancel: () => void }) {
    const [value, setValue] = useState(config.defaultValue)
    const [isLoading, setIsLoading] = useState(false)
    const [error, setError] = useState('')
    const inputRef = useRef<HTMLInputElement>(null)

    useEffect(() => {
        inputRef.current?.focus()
        inputRef.current?.select()
    }, [])

    const handleSubmit = async () => {
        if (!value.trim()) return
        setIsLoading(true)
        setError('')
        try {
            await config.onConfirm(value.trim())
        } catch (err: any) {
            setError(err?.message || 'Operation failed')
            setIsLoading(false)
        }
    }

    return (
        <Modal onClose={onCancel} loading={isLoading} width={400}>
            <ModalHeader>
                <h2 className="text-[14px] font-semibold text-[var(--text-primary)] m-0">{config.title}</h2>
            </ModalHeader>
            <ModalBody>
                <label className="text-[12px] text-[var(--text-dim)] block mb-2">{config.label}</label>
                <input
                    ref={inputRef}
                    type="text"
                    className="w-full rounded-md text-[13px]"
                    style={{
                        background: 'var(--bg-sidebar)',
                        border: '1px solid var(--border)',
                        color: 'var(--text-primary)',
                        fontFamily: 'var(--font-sans)',
                        padding: '8px 10px',
                        outline: 'none',
                    }}
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    onKeyDown={(e) => {
                        if (e.key === 'Enter') { e.preventDefault(); handleSubmit() }
                        if (e.key === 'Escape') { e.preventDefault(); onCancel() }
                    }}
                    disabled={isLoading}
                />
                {error && <p className="text-[12px] text-[var(--red)] mt-3 mb-0">{error}</p>}
            </ModalBody>
            <ModalFooter>
                <ModalButton onClick={onCancel} disabled={isLoading}>Cancel</ModalButton>
                <ModalButton variant="primary" onClick={handleSubmit} disabled={isLoading || !value.trim()}>
                    {isLoading ? `${config.confirmLabel}ing...` : config.confirmLabel}
                </ModalButton>
            </ModalFooter>
        </Modal>
    )
}

const GRAPH_COLORS = [
    '#e06c75', '#61afef', '#98c379', '#e5c07b',
    '#c678dd', '#56b6c2', '#d19a66', '#be5046',
]

function GraphLine({ line }: { line: string }) {
    const chars = line.split('')
    return (
        <span className="flex-shrink-0 select-none" style={{ fontFamily: 'var(--font-mono)', lineHeight: '22px', fontSize: '11px', whiteSpace: 'pre' }}>
            {chars.map((ch, i) => {
                const lane = Math.floor(i / 2)
                const color = GRAPH_COLORS[lane % GRAPH_COLORS.length]
                let display = ch
                switch (ch) {
                    case '*': display = '●'; break
                    case '|': display = '│'; break
                    case '/': display = '╱'; break
                    case '\\': display = '╲'; break
                    default: display = ch
                }
                const isGraph = ch === '*' || ch === '|' || ch === '/' || ch === '\\'
                return (
                    <span key={i} style={isGraph ? { color } : undefined}>{display}</span>
                )
            })}
        </span>
    )
}
