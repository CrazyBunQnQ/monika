import { useState, useEffect, useCallback } from 'react'
import { App } from '../../../bindings/monika'

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
    const [selectedFile, setSelectedFile] = useState<KBFileInfo | null>(null)
    const [fileContent, setFileContent] = useState('')
    const [searchQuery, setSearchQuery] = useState('')
    const [searchResults, setSearchResults] = useState<KBFileInfo[]>([])
    const [loading, setLoading] = useState(false)
    const [editing, setEditing] = useState(false)
    const [editContent, setEditContent] = useState('')

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

    const filesToShow = searchResults.length > 0 ? searchResults : files

    return (
        <div className="flex h-full">
            <div className="w-64 border-r border-[var(--border)] flex flex-col">
                <div className="p-2 border-b border-[var(--border)]">
                    <div className="flex gap-1 mb-2">
                        <button
                            onClick={() => setScope('project')}
                            className={`px-2 py-1 text-xs rounded ${scope === 'project' ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-secondary)]'}`}
                        >
                            Project
                        </button>
                        <button
                            onClick={() => setScope('global')}
                            className={`px-2 py-1 text-xs rounded ${scope === 'global' ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-secondary)]'}`}
                        >
                            Global
                        </button>
                    </div>
                    <div className="flex gap-1">
                        <input
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            placeholder="Search..."
                            className="flex-1 px-2 py-1 text-xs bg-[var(--bg-input)] border border-[var(--border)] rounded"
                        />
                        <button onClick={handleSearch} className="px-2 py-1 text-xs bg-[var(--accent)] text-white rounded">
                            Search
                        </button>
                    </div>
                </div>

                {stats && (
                    <div className="px-2 py-1 text-[10px] text-[var(--text-dim)] border-b border-[var(--border)]">
                        Total: {stats.total} | Active: {stats.active} | Archived: {stats.archived}
                    </div>
                )}

                <div className="flex-1 overflow-y-auto">
                    {loading ? (
                        <div className="p-2 text-xs text-[var(--text-dim)]">Loading...</div>
                    ) : (
                        filesToShow.map((f) => (
                            <div
                                key={f.path}
                                onClick={() => handleSelectFile(f)}
                                className={`px-2 py-1.5 cursor-pointer border-b border-[var(--border)] ${selectedFile?.path === f.path ? 'bg-[var(--bg-active)]' : ''
                                    }`}
                            >
                                <div className="text-xs font-medium truncate">{f.title}</div>
                                <div className="text-[10px] text-[var(--text-dim)]">
                                    {f.category} · {f.confidence} · {f.updated_at}
                                </div>
                            </div>
                        ))
                    )}
                </div>
            </div>

            <div className="flex-1 flex flex-col">
                {selectedFile ? (
                    <>
                        <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--border)]">
                            <div>
                                <span className="text-sm font-semibold">{selectedFile.title}</span>
                                <span className="ml-2 text-[10px] text-[var(--text-dim)]">{selectedFile.path}</span>
                            </div>
                            <div className="flex gap-1">
                                {editing ? (
                                    <>
                                        <button onClick={handleSave} className="px-2 py-1 text-xs bg-[var(--green)] text-white rounded">
                                            Save
                                        </button>
                                        <button onClick={() => { setEditing(false); setEditContent(fileContent) }} className="px-2 py-1 text-xs bg-[var(--bg-hover)] rounded">
                                            Cancel
                                        </button>
                                    </>
                                ) : (
                                    <>
                                        <button
                                            onClick={() => { setEditing(true); setEditContent(fileContent) }}
                                            className="px-2 py-1 text-xs bg-[var(--bg-hover)] text-[var(--text-secondary)] rounded"
                                        >
                                            Edit
                                        </button>
                                        <button
                                            onClick={() => handleDelete(selectedFile)}
                                            className="px-2 py-1 text-xs bg-[var(--red-muted)] text-[var(--red)] rounded"
                                        >
                                            Delete
                                        </button>
                                    </>
                                )}
                            </div>
                        </div>
                        <div className="flex-1 overflow-y-auto p-3">
                            {editing ? (
                                <textarea
                                    value={editContent}
                                    onChange={(e) => setEditContent(e.target.value)}
                                    className="w-full h-full min-h-[300px] bg-[var(--bg-input)] border border-[var(--border)] rounded p-2 text-xs font-mono resize-none"
                                />
                            ) : (
                                <pre className="text-xs whitespace-pre-wrap font-sans">{fileContent}</pre>
                            )}
                        </div>
                    </>
                ) : (
                    <div className="flex-1 flex items-center justify-center text-sm text-[var(--text-dim)]">
                        Select a file to preview
                    </div>
                )}
            </div>
        </div>
    )
}

export default KnowledgeBaseTab
