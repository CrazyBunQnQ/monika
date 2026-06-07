import React, { useCallback, useRef, useState } from 'react'
import { Browser } from '@wailsio/runtime'

export function CodeBlock({ children, ...rest }: React.ComponentPropsWithoutRef<'pre'>) {
    const ref = useRef<HTMLPreElement>(null)
    const [copied, setCopied] = useState(false)

    const handleCopy = useCallback(() => {
        const el = ref.current
        if (!el) return
        const text = el.textContent || ''
        navigator.clipboard.writeText(text).catch(() => { })
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
    }, [])

    return (
        <div className="relative group/codeblock">
            <pre ref={ref} {...rest}>{children}</pre>
            <button
                className="absolute top-[6px] right-[6px] opacity-0 group-hover/codeblock:opacity-100 transition-opacity
                   text-[10px] font-semibold uppercase tracking-[0.04em] rounded px-1.5 py-0.5
                   hover:bg-[var(--bg-hover)] cursor-pointer"
                style={{ color: 'var(--text-dim)' }}
                onClick={handleCopy}
                aria-label="Copy code"
            >
                {copied ? 'Copied' : 'Copy'}
            </button>
        </div>
    )
}

export function ExternalLink({ href, children, ...rest }: React.ComponentPropsWithoutRef<'a'>) {
    const handleClick = useCallback((e: React.MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault()
        if (href) Browser.OpenURL(href)
    }, [href])

    return <a href={href} onClick={handleClick} target="_blank" rel="noopener noreferrer" {...rest}>{children}</a>
}
