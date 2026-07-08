import { useState, useEffect, useCallback } from 'react'
import { useStore } from '../../store'
import type { ProxyConfig } from '../../store'

export default function NetworkTab() {
    const loadProxyConfig = useStore(s => s.loadProxyConfig)
    const saveProxyConfig = useStore(s => s.saveProxyConfig)

    const [enabled, setEnabled] = useState(false)
    const [url, setUrl] = useState('')
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [saved, setSaved] = useState(false)
    const [error, setError] = useState('')

    useEffect(() => {
        loadProxyConfig().then((cfg: ProxyConfig) => {
            setEnabled(cfg.enabled || false)
            setUrl(cfg.url || '')
            setLoading(false)
        }).catch(() => setLoading(false))
    }, [loadProxyConfig])

    const handleSave = useCallback(async () => {
        setSaving(true)
        setError('')
        setSaved(false)
        try {
            await saveProxyConfig({ enabled, url: url.trim() })
            setSaved(true)
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to save')
        } finally {
            setSaving(false)
        }
    }, [enabled, url, saveProxyConfig])

    if (loading) {
        return <div className="text-[12px] text-[var(--text-dim)]">Loading...</div>
    }

    const inputCls = 'w-full px-3 py-2 text-[12px] rounded-md border border-[var(--border)] bg-[var(--bg-card)] text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--border-strong)] form-input-glow transition-colors duration-150'
    const labelCls = 'block text-[11px] font-medium text-[var(--text-secondary)] mb-1.5'

    return (
        <div>
            <div className="mb-4">
                <h3 className="text-[15px] font-semibold m-0 mb-1">Network</h3>
                <p className="text-[11px] text-[var(--text-dim)] m-0">Configure proxy settings for outbound HTTP requests.</p>
            </div>

            <div className="rounded-lg p-4 space-y-4" style={{ background: 'var(--bg-card)' }}>
                <div className="flex items-center justify-between">
                    <div>
                        <span className="text-[13px] font-medium text-[var(--text-primary)]">Enable Proxy</span>
                        <p className="text-[11px] text-[var(--text-dim)] m-0 mt-0.5">Route all HTTP requests through a proxy server.</p>
                    </div>
                    <button
                        onClick={() => setEnabled(!enabled)}
                        className="relative w-9 h-5 rounded-full transition-colors cursor-pointer border-none"
                        style={{
                            background: enabled ? 'var(--accent)' : 'var(--bg-sidebar)',
                        }}
                        aria-label="Toggle proxy"
                    >
                        <span
                            className="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform"
                            style={{ left: enabled ? '18px' : '2px' }}
                        />
                    </button>
                </div>

                {enabled && (
                    <div>
                        <label className={labelCls}>Proxy URL</label>
                        <input
                            className={inputCls}
                            value={url}
                            onChange={e => setUrl(e.target.value)}
                            placeholder="http://127.0.0.1:10808"
                        />
                        <p className="text-[10px] text-[var(--text-dim)] m-0 mt-1">
                            Supports HTTP and SOCKS5 proxies. Example: http://127.0.0.1:10808
                        </p>
                    </div>
                )}

                {error && <p className="text-[11px] text-[var(--red)] m-0">{error}</p>}
                {saved && <p className="text-[11px] m-0" style={{ color: 'var(--green)' }}>Saved. Restart may be required for all changes to take effect.</p>}

                <div className="flex justify-end">
                    <button
                        onClick={handleSave}
                        disabled={saving}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors disabled:opacity-50"
                        style={{ color: 'var(--text-primary)' }}
                    >
                        {saving ? 'Saving...' : 'Save'}
                    </button>
                </div>
            </div>
        </div>
    )
}
