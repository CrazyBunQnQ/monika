import { useState, useEffect, useCallback } from 'react'
import { App } from '../../../bindings/monika'
import Modal, { ModalHeader, ModalBody, ModalFooter, ModalButton } from '../ui/Modal'
import ConfirmModal from '../Chat/ConfirmModal'
import { IconDatabase, IconSearch, IconTrash, IconEdit, IconEye } from '../Icons'

interface KBFileInfo {
    path: string
    scope: string
    category: string
    title: string
    tags: string[]
    confidence: string
    status: string
    char_count: number
    created_at: string
    updated_at: string
}

interface KBStats {
    total: number
    active: number
    archived: number
    last_update: string
}

function KnowledgeBaseTab() {
    const [scope, setScope] = useState<'global' | 'project'>('project')
    const [files, setFiles] = useState<KBFileInfo[]>([])
    const [stats, setStats] = useState<KBStats | null>(null)
    const [searchQuery, setSearchQuery] = useState('')
    const [searchResults, setSearchResults] = useState<KBFileInfo[]>([])
    const [loading, setLoading] = useState(false)

    // modal state
    const [viewFile, setViewFile] = useState<KBFileInfo | null>(null)
    const [fileContent, setFileContent] = useState('')
    const [contentLoading, setContentLoading] = useState(false)
    const [editing, setEditing] = useState(false)
    const [editContent, setEditContent] = useState('')
    const [confirmDelete, setConfirmDelete] = useState<KBFileInfo | null>(null)

    const loadFiles = useCallback(async () => {
        setLoading(true)
        try {
            const result = await App.KBListFiles(scope)
            setFiles(result || [])
        } catch (e) {
            console.error('KBListFiles:', e)
            setFiles([])
        }
        setLoading(false)
    }, [scope])

    const loadStats = useCallback(async () => {
        try {
            const result = await App.KBStatistics(scope)
            setStats(result)
        } catch (e) {
            console.error('KBStatistics:', e)
        }
    }, [scope])

    useEffect(() => {
        loadFiles()
        loadStats()
    }, [loadFiles, loadStats])

    const handleView = async (f: KBFileInfo) => {
        setViewFile(f)
        setEditing(false)
        setContentLoading(true)
        try {
            const content = await App.KBReadFile(f.scope, f.path)
            setFileContent(content)
        } catch (e) {
            setFileContent('(error loading)')
        }
        setContentLoading(false)
    }

    const closeModal = () => {
        setViewFile(null)
        setEditing(false)
        setFileContent('')
    }

    const handleSearch = async () => {
        if (!searchQuery.trim()) {
            setSearchResults([])
            return
        }
        try {
            const results = await App.KBSearch(searchQuery, scope)
            setSearchResults(results || [])
        } catch (e) {
            console.error('KBSearch:', e)
        }
    }

    const handleDelete = async (f: KBFileInfo) => {
        try {
            await App.KBDeleteFile(f.scope, f.path)
            setConfirmDelete(null)
            setFiles((prev) => prev.filter((x) => x.path !== f.path))
            loadStats()
            if (viewFile?.path === f.path) {
                closeModal()
            }
        } catch (e) {
            console.error('KBDeleteFile:', e)
        }
    }

    const handleSave = async () => {
        if (!viewFile) return
        try {
            await App.KBWriteFile({
                scope: viewFile.scope,
                category: viewFile.category,
                title: viewFile.title,
                content: editContent,
                tags: viewFile.tags,
                confidence: viewFile.confidence,
            })
            setEditing(false)
            setFileContent(editContent)
            loadFiles()
        } catch (e) {
            console.error('KBWriteFile:', e)
        }
    }

    const handleToggleStatus = async (f: KBFileInfo) => {
        const newStatus = f.status === 'active' ? 'archived' : 'active'
        try {
            await App.KBSetFileStatus(f.scope, f.path, newStatus)
            loadFiles()
            loadStats()
        } catch (e) {
            console.error('KBSetFileStatus:', e)
        }
    }

    const filesToShow = searchResults.length > 0 ? searchResults : files

    const categoryBadge = (cat: string) => {
        const label = cat.replace(/^(raw|wiki)\//, '')
        const styles: Record<string, string> = {
            doc: 'text-[var(--accent)] bg-[var(--accent-muted)]',
            code: 'text-[var(--green)] bg-[var(--green)]/10',
            lessons: 'text-[#f59e0b] bg-[rgba(245,158,11,0.1)]',
            topics: 'text-[#8b5cf6] bg-[rgba(139,92,246,0.1)]',
            knowledge: 'text-[var(--text-primary)] bg-[var(--bg-sidebar)]',
            profile: 'text-[var(--text-dim)] bg-[var(--bg-sidebar)]',
        }
        const cls = styles[label] || 'text-[var(--text-dim)] bg-[var(--bg-sidebar)]'
        return <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium ${cls}`}>{label}</span>
    }

    return (
        <div>
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h3 className="text-[15px] font-semibold m-0 mb-1">Memory</h3>
                    <p className="text-[11px] text-[var(--text-dim)] m-0">Browse, search and manage stored memories</p>
                </div>
                <div className="inline-flex rounded-md border border-[var(--border)] overflow-hidden">
                    {(['project', 'global'] as const).map((s) => (
                        <button
                            key={s}
                            onClick={() => { setScope(s); setSearchResults([]) }}
                            className="px-3 py-1.5 text-[11px] font-medium cursor-pointer transition-colors border-none"
                            style={{
                                background: scope === s ? 'var(--accent-muted)' : 'transparent',
                                color: scope === s ? 'var(--accent)' : 'var(--text-secondary)',
                            }}
                        >
                            {s.charAt(0).toUpperCase() + s.slice(1)}
                        </button>
                    ))}
                </div>
            </div>

            {/* Search bar */}
            <div className="mb-3">
                <div className="flex gap-2">
                    <div className="relative flex-1">
                        <div className="absolute left-2.5 top-1/2 -translate-y-1/2" style={{ color: 'var(--text-dim)' }}>
                            <IconSearch size={12} />
                        </div>
                        <input
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            placeholder="Search memory..."
                            className="w-full pl-8 pr-3 py-2 text-[12px] rounded-md border border-[var(--border)] bg-[var(--bg-card)] text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--border-strong)] form-input-glow transition-colors duration-150"
                        />
                    </div>
                    <button onClick={handleSearch} className="inline-flex items-center gap-1 px-3 py-1.5 text-[12px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors">
                        Search
                    </button>
                </div>
            </div>

            {/* Stats */}
            {stats && (
                <div className="mb-3 px-3 py-2 text-[10px] text-[var(--text-dim)] rounded-lg border border-[var(--border)]" style={{ background: 'var(--bg-card)' }}>
                    Total: {stats.total} &middot; Active: {stats.active} &middot; Archived: {stats.archived}
                    {stats.last_update && <> &middot; Last update: {stats.last_update}</>}
                </div>
            )}

            {/* Table */}
            {loading ? (
                <div className="py-8 text-center text-[12px] text-[var(--text-dim)]">Loading...</div>
            ) : filesToShow.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-[var(--text-dim)]">
                    <IconDatabase size={32} />
                    <span className="text-[13px] mt-3">No files found.</span>
                    <span className="text-[11px] mt-1">Use AI chat to add documents, repos, or memories.</span>
                </div>
            ) : (
                <div className="rounded-lg border border-[var(--border)] overflow-hidden" style={{ background: 'var(--bg-card)' }}>
                    <table className="w-full" style={{ borderCollapse: 'collapse' }}>
                        <thead>
                            <tr style={{ borderBottom: '1px solid var(--border)' }}>
                                <th className="text-left px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-[var(--text-dim)]">Title</th>
                                <th className="text-left px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-[var(--text-dim)] w-[90px]">Category</th>
                                <th className="text-left px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-[var(--text-dim)] w-[70px]">Confidence</th>
                                <th className="text-left px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-[var(--text-dim)] w-[120px]">Updated</th>
                                <th className="text-center px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-[var(--text-dim)] w-[50px]">Status</th>
                                <th className="text-right px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-[var(--text-dim)] w-[60px]">Action</th>
                            </tr>
                        </thead>
                        <tbody>
                            {filesToShow.map((f, i) => (
                                <tr
                                    key={f.path}
                                    style={{
                                        borderBottom: i < filesToShow.length - 1 ? '1px solid var(--border)' : 'none',
                                        opacity: f.status === 'archived' ? 0.5 : 1,
                                    }}
                                    className="hover:bg-[var(--bg-hover)] transition-colors"
                                >
                                    <td className="px-3 py-2.5">
                                        <div className="text-[12px] font-medium text-[var(--text-primary)] truncate max-w-[280px]">{f.title}</div>
                                        <div className="text-[10px] text-[var(--text-dim)] font-mono truncate max-w-[280px]">{f.path}</div>
                                    </td>
                                    <td className="px-3 py-2.5">{categoryBadge(f.category)}</td>
                                    <td className="px-3 py-2.5">
                                        <span className="text-[11px] text-[var(--text-dim)]">{f.confidence}</span>
                                    </td>
                                    <td className="px-3 py-2.5">
                                        <span className="text-[11px] text-[var(--text-dim)]">{f.updated_at}</span>
                                    </td>
                                    <td className="px-3 py-2.5 text-center">
                                        <button
                                            onClick={() => handleToggleStatus(f)}
                                            className="relative w-8 h-[18px] rounded-full border-none cursor-pointer transition-colors"
                                            style={{ background: f.status === 'active' ? 'var(--accent)' : 'var(--border)' }}
                                            title={f.status === 'active' ? 'Disable' : 'Enable'}
                                        >
                                            <span
                                                className="absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-all"
                                                style={{ left: f.status === 'active' ? '14px' : '2px' }}
                                            />
                                        </button>
                                    </td>
                                    <td className="px-3 py-2.5 text-right">
                                        <button
                                            onClick={() => handleView(f)}
                                            className="inline-flex items-center gap-1 px-2 py-1 text-[11px] font-medium rounded border border-[var(--border)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors"
                                        >
                                            <IconEye size={11} /> View
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {/* View / Edit Modal */}
            {viewFile && (
                <Modal onClose={closeModal} width={600}>
                    <ModalHeader icon={<IconDatabase size={15} />}>
                        <div className="min-w-0">
                            <div className="flex items-center gap-2">
                                <h4 className="text-[14px] font-semibold m-0 truncate">{viewFile.title}</h4>
                                {categoryBadge(viewFile.category)}
                            </div>
                            <div className="text-[10px] text-[var(--text-dim)] mt-0.5 font-mono truncate">{viewFile.path}</div>
                        </div>
                    </ModalHeader>

                    <ModalBody>
                        {contentLoading ? (
                            <div className="py-8 text-center text-[12px] text-[var(--text-dim)]">Loading...</div>
                        ) : editing ? (
                            <textarea
                                value={editContent}
                                onChange={(e) => setEditContent(e.target.value)}
                                className="w-full min-h-[300px] bg-[var(--bg-input)] border border-[var(--border)] rounded p-3 text-[12px] font-mono resize-none text-[var(--text-primary)] focus:outline-none focus:border-[var(--border-strong)]"
                            />
                        ) : (
                            <pre className="text-[12px] whitespace-pre-wrap font-sans text-[var(--text-primary)] leading-relaxed m-0">{fileContent}</pre>
                        )}
                        {!editing && viewFile.tags && viewFile.tags.length > 0 && (
                            <div className="mt-3 pt-3 border-t border-[var(--border)] flex flex-wrap gap-1">
                                {viewFile.tags.map((tag) => (
                                    <span key={tag} className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--bg-sidebar)] text-[var(--text-dim)] font-mono">
                                        {tag}
                                    </span>
                                ))}
                            </div>
                        )}
                    </ModalBody>

                    <ModalFooter>
                        {editing ? (
                            <>
                                <ModalButton onClick={handleSave} variant="primary">Save</ModalButton>
                                <ModalButton onClick={() => { setEditing(false); setEditContent(fileContent) }}>Cancel</ModalButton>
                            </>
                        ) : (
                            <>
                                <ModalButton onClick={() => setConfirmDelete(viewFile)} variant="danger">
                                    <span className="inline-flex items-center gap-1"><IconTrash size={11} /> Delete</span>
                                </ModalButton>
                                <ModalButton onClick={() => { setEditing(true); setEditContent(fileContent) }}>
                                    <span className="inline-flex items-center gap-1"><IconEdit size={11} /> Edit</span>
                                </ModalButton>
                                <ModalButton onClick={closeModal}>Close</ModalButton>
                            </>
                        )}
                    </ModalFooter>
                </Modal>
            )}

            {/* Delete confirmation */}
            {confirmDelete && (
                <ConfirmModal
                    title="Delete Memory"
                    message={`Are you sure you want to delete "${confirmDelete.title}"? This will soft-delete the file.`}
                    confirmLabel="Delete"
                    onConfirm={() => handleDelete(confirmDelete)}
                    onCancel={() => setConfirmDelete(null)}
                />
            )}
        </div>
    )
}

export default KnowledgeBaseTab
