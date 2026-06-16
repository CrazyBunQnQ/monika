import { useState, useEffect, useCallback } from 'react'
import { App } from '../../../bindings/monika'
import Modal, { ModalHeader, ModalBody, ModalFooter, ModalButton } from '../ui/Modal'
import ConfirmModal from '../Chat/ConfirmModal'
import { IconDatabase, IconFile, IconSearch, IconFilePlus, IconExternalLink, IconTrash, IconEdit } from '../Icons'

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

const inputCls = 'w-full px-3 py-2 text-[12px] rounded-md border border-[var(--border)] bg-[var(--bg-card)] text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--border-strong)] form-input-glow transition-colors duration-150'
const labelCls = 'block text-[11px] font-medium text-[var(--text-secondary)] mb-1.5'

function KnowledgeBaseTab() {
    const [scope, setScope] = useState<'global' | 'project'>('project')
    const [files, setFiles] = useState<KBFileInfo[]>([])
    const [stats, setStats] = useState<KBStats | null>(null)
    const [selectedFile, setSelectedFile] = useState<KBFileInfo | null>(null)
    const [fileContent, setFileContent] = useState('')
    const [searchQuery, setSearchQuery] = useState('')
    const [searchResults, setSearchResults] = useState<KBFileInfo[]>([])
    const [loading, setLoading] = useState(false)
    const [editing, setEditing] = useState(false)
    const [editContent, setEditContent] = useState('')

    const [showDocModal, setShowDocModal] = useState(false)
    const [showRepoModal, setShowRepoModal] = useState(false)
    const [confirmDelete, setConfirmDelete] = useState<KBFileInfo | null>(null)
    const [saving, setSaving] = useState(false)

    // Upload document form state
    const [docTitle, setDocTitle] = useState('')
    const [docContent, setDocContent] = useState('')
    const [docScope, setDocScope] = useState<'project' | 'global'>('project')

    // Add repo form state
    const [repoURL, setRepoURL] = useState('')
    const [repoDesc, setRepoDesc] = useState('')
    const [repoScope, setRepoScope] = useState<'project' | 'global'>('project')

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

    const handleSelectFile = async (f: KBFileInfo) => {
        setSelectedFile(f)
        setEditing(false)
        try {
            const content = await App.KBReadFile(f.scope, f.path)
            setFileContent(content)
        } catch (e) {
            setFileContent('(error loading)')
        }
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
            loadFiles()
            loadStats()
            if (selectedFile?.path === f.path) {
                setSelectedFile(null)
                setFileContent('')
            }
        } catch (e) {
            console.error('KBDeleteFile:', e)
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

    const handleSave = async () => {
        if (!selectedFile) return
        try {
            await App.KBWriteFile({
                scope: selectedFile.scope,
                category: selectedFile.category,
                title: selectedFile.title,
                content: editContent,
                tags: selectedFile.tags,
                confidence: selectedFile.confidence,
            })
            setEditing(false)
            setFileContent(editContent)
            loadFiles()
        } catch (e) {
            console.error('KBWriteFile:', e)
        }
    }

    const handleUploadDoc = async () => {
        if (!docTitle.trim() || !docContent.trim()) return
        setSaving(true)
        try {
            await App.KBCreateMemory({
                scope: docScope,
                category: 'raw/doc',
                title: docTitle.trim(),
                content: docContent.trim(),
                tags: [],
                confidence: 'medium',
            })
            setShowDocModal(false)
            setDocTitle('')
            setDocContent('')
            loadFiles()
            loadStats()
        } catch (e) {
            console.error('KBCreateMemory (doc):', e)
        } finally {
            setSaving(false)
        }
    }

    const handleAddRepo = async () => {
        if (!repoURL.trim()) return
        setSaving(true)
        try {
            await App.KBCreateMemory({
                scope: repoScope,
                category: 'raw/code',
                title: repoURL.trim(),
                content: repoDesc.trim() || repoURL.trim(),
                tags: ['repo'],
                confidence: 'medium',
            })
            setShowRepoModal(false)
            setRepoURL('')
            setRepoDesc('')
            loadFiles()
            loadStats()
        } catch (e) {
            console.error('KBCreateMemory (repo):', e)
        } finally {
            setSaving(false)
        }
    }

    const filesToShow = searchResults.length > 0 ? searchResults : files

    const categoryBadge = (cat: string) => {
        const styles: Record<string, string> = {
            'raw/doc': 'text-[var(--accent)] bg-[var(--accent-muted)]',
            'raw/code': 'text-[var(--green)] bg-[var(--green)]/10',
        }
        const labels: Record<string, string> = {
            'raw/doc': 'doc',
            'raw/code': 'repo',
        }
        const cls = styles[cat] || 'text-[var(--text-dim)] bg-[var(--bg-sidebar)]'
        return <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium ${cls}`}>{labels[cat] || cat}</span>
    }

    return (
        <div>
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h3 className="text-[15px] font-semibold m-0 mb-1">Knowledge Base</h3>
                    <p className="text-[11px] text-[var(--text-dim)] m-0">Manage files, documents and repository links</p>
                </div>
                <div className="flex items-center gap-2">
                    {/* Scope toggle */}
                    <div className="inline-flex rounded-md border border-[var(--border)] overflow-hidden">
                        {(['project', 'global'] as const).map((s) => (
                            <button
                                key={s}
                                onClick={() => setScope(s)}
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
                    <button
                        onClick={() => { setShowDocModal(true); setDocScope(scope) }}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors"
                    >
                        <IconFilePlus size={12} /> Upload
                    </button>
                    <button
                        onClick={() => { setShowRepoModal(true); setRepoScope(scope) }}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors"
                    >
                        <IconExternalLink size={12} /> Add Repo
                    </button>
                </div>
            </div>

            {/* Search bar */}
            <div className="mb-4">
                <div className="flex gap-2">
                    <div className="relative flex-1">
                        <div className="absolute left-2.5 top-1/2 -translate-y-1/2" style={{ color: 'var(--text-dim)' }}>
                            <IconSearch size={12} />
                        </div>
                        <input
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            placeholder="Search knowledge base..."
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
                <div className="mb-4 px-3 py-2 text-[10px] text-[var(--text-dim)] rounded-lg border border-[var(--border)]" style={{ background: 'var(--bg-card)' }}>
                    Total: {stats.total} &middot; Active: {stats.active} &middot; Archived: {stats.archived}
                    {stats.last_update && <> &middot; Last update: {stats.last_update}</>}
                </div>
            )}

            {/* Main area: two columns */}
            <div className="flex gap-4">
                {/* File list */}
                <div className="w-72 flex flex-col space-y-1.5">
                    {loading ? (
                        <div className="px-3 py-4 text-[11px] text-[var(--text-dim)]">Loading...</div>
                    ) : filesToShow.length === 0 ? (
                        <div className="flex flex-col items-center justify-center py-16 text-[var(--text-dim)]">
                            <IconDatabase size={32} />
                            <span className="text-[13px] mt-3">No files found.</span>
                            <span className="text-[11px] mt-1">Upload a document or add a repo.</span>
                        </div>
                    ) : (
                        filesToShow.map((f) => (
                            <div
                                key={f.path}
                                onClick={() => handleSelectFile(f)}
                                className="rounded-lg border border-[var(--border)] px-3 py-2.5 cursor-pointer transition-colors"
                                style={{
                                    background: selectedFile?.path === f.path ? 'var(--bg-active, var(--accent-muted))' : 'var(--bg-card)',
                                    borderColor: selectedFile?.path === f.path ? 'var(--accent)' : 'var(--border)',
                                    opacity: f.status === 'archived' ? 0.5 : 1,
                                }}
                            >
                                <div className="flex items-center gap-2 mb-1">
                                    <span style={{ color: 'var(--text-dim)' }}>
                                        <IconFile size={13} />
                                    </span>
                                    <span className="text-[12px] font-medium truncate flex-1">{f.title}</span>
                                    {categoryBadge(f.category)}
                                    <button
                                        onClick={(e) => { e.stopPropagation(); handleToggleStatus(f) }}
                                        className="relative w-8 h-[18px] rounded-full border-none cursor-pointer transition-colors shrink-0"
                                        style={{ background: f.status === 'active' ? 'var(--accent)' : 'var(--border)' }}
                                        title={f.status === 'active' ? 'Disable' : 'Enable'}
                                    >
                                        <span
                                            className="absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-all"
                                            style={{ left: f.status === 'active' ? '14px' : '2px' }}
                                        />
                                    </button>
                                </div>
                                <div className="flex items-center gap-2 text-[10px] text-[var(--text-dim)]">
                                    <span>{f.confidence}</span>
                                    <span>&middot;</span>
                                    <span>{f.updated_at}</span>
                                </div>
                            </div>
                        ))
                    )}
                </div>

                {/* Detail / preview panel */}
                <div className="flex-1 min-h-[300px]">
                    {selectedFile ? (
                        <div className="rounded-lg border border-[var(--border)] overflow-hidden" style={{ background: 'var(--bg-card)' }}>
                            <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border)]">
                                <div className="min-w-0 flex-1">
                                    <div className="flex items-center gap-2">
                                        <span className="text-[13px] font-semibold truncate">{selectedFile.title}</span>
                                        {categoryBadge(selectedFile.category)}
                                    </div>
                                    <div className="text-[10px] text-[var(--text-dim)] mt-0.5 truncate font-mono">{selectedFile.path}</div>
                                </div>
                                <div className="flex items-center gap-1.5 shrink-0 ml-3">
                                    {editing ? (
                                        <>
                                            <button onClick={handleSave} className="inline-flex items-center px-3 py-1.5 text-[11px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors">
                                                Save
                                            </button>
                                            <button onClick={() => { setEditing(false); setEditContent(fileContent) }} className="inline-flex items-center px-3 py-1.5 text-[11px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors">
                                                Cancel
                                            </button>
                                        </>
                                    ) : (
                                        <>
                                            <button
                                                onClick={() => { setEditing(true); setEditContent(fileContent) }}
                                                className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--text-primary)] px-1.5 py-1 cursor-pointer bg-transparent border-none rounded transition-colors"
                                                aria-label="Edit"
                                            >
                                                <IconEdit size={13} />
                                            </button>
                                            <button
                                                onClick={() => setConfirmDelete(selectedFile)}
                                                className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--red)] px-1.5 py-1 cursor-pointer bg-transparent border-none rounded transition-colors"
                                                aria-label="Delete"
                                            >
                                                <IconTrash size={13} />
                                            </button>
                                        </>
                                    )}
                                </div>
                            </div>
                            <div className="p-4">
                                {editing ? (
                                    <textarea
                                        value={editContent}
                                        onChange={(e) => setEditContent(e.target.value)}
                                        className="w-full min-h-[300px] bg-[var(--bg-input)] border border-[var(--border)] rounded p-3 text-[12px] font-mono resize-none focus:outline-none focus:border-[var(--border-strong)] transition-colors"
                                        style={{ color: 'var(--text-primary)' }}
                                    />
                                ) : (
                                    <pre className="text-[12px] text-[var(--text-secondary)] whitespace-pre-wrap font-sans m-0 leading-relaxed">{fileContent}</pre>
                                )}
                            </div>
                        </div>
                    ) : (
                        <div className="flex flex-col items-center justify-center h-full min-h-[300px] text-[var(--text-dim)] rounded-lg border border-dashed border-[var(--border)]">
                            <IconDatabase size={32} />
                            <span className="text-[13px] mt-3">Select a file to preview</span>
                        </div>
                    )}
                </div>
            </div>

            {/* Upload Document Modal */}
            {showDocModal && (
                <Modal onClose={() => setShowDocModal(false)} loading={saving} width={480}>
                    <ModalHeader icon={<IconFilePlus size={15} />}>
                        <h4 className="text-[14px] font-semibold m-0">Upload Document</h4>
                    </ModalHeader>
                    <ModalBody>
                        <div className="space-y-4">
                            <div>
                                <label className={labelCls}>Title</label>
                                <input
                                    type="text"
                                    className={inputCls}
                                    placeholder="Document title"
                                    value={docTitle}
                                    onChange={(e) => setDocTitle(e.target.value)}
                                    autoFocus
                                />
                            </div>
                            <div>
                                <label className={labelCls}>Content</label>
                                <textarea
                                    className={inputCls + ' min-h-[150px] resize-y font-mono'}
                                    placeholder="Paste document content here..."
                                    value={docContent}
                                    onChange={(e) => setDocContent(e.target.value)}
                                />
                            </div>
                            <div>
                                <label className={labelCls}>Scope</label>
                                <div className="inline-flex rounded-md border border-[var(--border)] overflow-hidden">
                                    {(['project', 'global'] as const).map((s) => (
                                        <button
                                            key={s}
                                            onClick={() => setDocScope(s)}
                                            className="px-3 py-1.5 text-[11px] font-medium cursor-pointer transition-colors border-none"
                                            style={{
                                                background: docScope === s ? 'var(--accent-muted)' : 'transparent',
                                                color: docScope === s ? 'var(--accent)' : 'var(--text-secondary)',
                                            }}
                                        >
                                            {s.charAt(0).toUpperCase() + s.slice(1)}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </ModalBody>
                    <ModalFooter>
                        <ModalButton onClick={() => setShowDocModal(false)} disabled={saving}>Cancel</ModalButton>
                        <ModalButton variant="primary" onClick={handleUploadDoc} disabled={saving || !docTitle.trim() || !docContent.trim()}>
                            {saving ? 'Uploading...' : 'Upload'}
                        </ModalButton>
                    </ModalFooter>
                </Modal>
            )}

            {/* Add Repo Modal */}
            {showRepoModal && (
                <Modal onClose={() => setShowRepoModal(false)} loading={saving} width={480}>
                    <ModalHeader icon={<IconExternalLink size={15} />}>
                        <h4 className="text-[14px] font-semibold m-0">Add Repository</h4>
                    </ModalHeader>
                    <ModalBody>
                        <div className="space-y-4">
                            <div>
                                <label className={labelCls}>Repository URL</label>
                                <input
                                    type="text"
                                    className={inputCls}
                                    placeholder="https://github.com/owner/repo"
                                    value={repoURL}
                                    onChange={(e) => setRepoURL(e.target.value)}
                                    autoFocus
                                />
                            </div>
                            <div>
                                <label className={labelCls}>Description</label>
                                <textarea
                                    className={inputCls + ' min-h-[100px] resize-y'}
                                    placeholder="Optional description..."
                                    value={repoDesc}
                                    onChange={(e) => setRepoDesc(e.target.value)}
                                />
                            </div>
                            <div>
                                <label className={labelCls}>Scope</label>
                                <div className="inline-flex rounded-md border border-[var(--border)] overflow-hidden">
                                    {(['project', 'global'] as const).map((s) => (
                                        <button
                                            key={s}
                                            onClick={() => setRepoScope(s)}
                                            className="px-3 py-1.5 text-[11px] font-medium cursor-pointer transition-colors border-none"
                                            style={{
                                                background: repoScope === s ? 'var(--accent-muted)' : 'transparent',
                                                color: repoScope === s ? 'var(--accent)' : 'var(--text-secondary)',
                                            }}
                                        >
                                            {s.charAt(0).toUpperCase() + s.slice(1)}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </ModalBody>
                    <ModalFooter>
                        <ModalButton onClick={() => setShowRepoModal(false)} disabled={saving}>Cancel</ModalButton>
                        <ModalButton variant="primary" onClick={handleAddRepo} disabled={saving || !repoURL.trim()}>
                            {saving ? 'Saving...' : 'Add Repository'}
                        </ModalButton>
                    </ModalFooter>
                </Modal>
            )}

            {/* Delete confirmation */}
            {confirmDelete && (
                <ConfirmModal
                    title="Delete File"
                    message={`Are you sure you want to delete "${confirmDelete.title}"?`}
                    confirmLabel="Delete"
                    icon={<IconTrash size={15} />}
                    onConfirm={async () => {
                        await handleDelete(confirmDelete)
                        setConfirmDelete(null)
                    }}
                    onCancel={() => setConfirmDelete(null)}
                />
            )}
        </div>
    )
}

export default KnowledgeBaseTab
