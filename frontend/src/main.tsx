import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'
import { TrayPopup } from './components/TrayPopup/TrayPopup'
import { setupWailsEvents, initProject } from './store'
import { initTreeSitter } from './lib/treeSitter'

function isResizeObserverError(msg: string | undefined): boolean {
    if (!msg) return false
    return msg.includes('ResizeObserver loop completed') ||
        msg.includes('ResizeObserver loop limit exceeded')
}

// Override window.onerror — fires BEFORE the 'error' event dispatch
const _origOnError = window.onerror
window.onerror = function(message, _source, _lineno, _colno, error) {
    const msg = typeof message === 'string' ? message : (error?.message || '')
    if (isResizeObserverError(msg)) return true
    return _origOnError ? _origOnError.call(this, message, _source, _lineno, _colno, error) : false
}

window.addEventListener(
    'error',
    (e) => {
        if (isResizeObserverError(e.message)) {
            e.stopImmediatePropagation()
            e.preventDefault()
        }
    },
    true,
)

window.addEventListener(
    'unhandledrejection',
    (e) => {
        const msg = e.reason?.message || e.reason?.toString?.()
        if (isResizeObserverError(msg)) {
            e.stopImmediatePropagation()
            e.preventDefault()
        }
    },
    true,
)

class ErrorBoundary extends React.Component<
    { children: React.ReactNode },
    { hasError: boolean; error: Error | null }
> {
    constructor(props: { children: React.ReactNode }) {
        super(props)
        this.state = { hasError: false, error: null }
    }
    static getDerivedStateFromError(error: Error) {
        return { hasError: true, error }
    }
    render() {
        if (this.state.hasError) {
            return (
                <div style={{ padding: 40, color: '#d4d4dc', background: '#08090d', height: '100%', overflow: 'auto' }}>
                    <h2 style={{ color: '#cd5454' }}>Render Error</h2>
                    <pre style={{ fontSize: 13, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                        {this.state.error?.stack || this.state.error?.message || 'Unknown error'}
                    </pre>
                </div>
            )
        }
        return this.props.children
    }
}

try {
    setupWailsEvents()
    initTreeSitter().catch(e => console.error('[tree-sitter] init failed:', e))
} catch (e) {
    document.getElementById('root')!.innerHTML = `<pre style="color:#d4d4dc;background:#08090d;padding:40px;height:100%">setupWailsEvents error: ${String(e)}\n${(e as Error).stack || ''}</pre>`
    throw e
}

const isTrayPopup = window.location.hash === '#/tray-popup'

if (isTrayPopup) {
    ReactDOM.createRoot(document.getElementById('root')!).render(
        <ErrorBoundary>
            <React.StrictMode>
                <TrayPopup />
            </React.StrictMode>
        </ErrorBoundary>,
    )
} else {
    ReactDOM.createRoot(document.getElementById('root')!).render(
        <ErrorBoundary>
            <React.StrictMode>
                <App />
            </React.StrictMode>
        </ErrorBoundary>,
    )
    initProject()
}
