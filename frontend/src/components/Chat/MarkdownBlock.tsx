import { useState, useEffect, useTransition, ReactNode, createElement } from 'react'
import type { ElementType } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { CodeBlock, ExternalLink } from '../MarkdownShared'

/* ---- inline color swatch: detect hex colors in LLM text ---- */

const HEX_COLOR_RE = /#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\b/g

function ColorChip({ color }: { color: string }) {
    return (
        <span className="md-color-chip" data-color={color}>
            <span className="md-color-swatch" style={{ background: color }} aria-hidden />
            <span className="md-color-code">{color}</span>
        </span>
    )
}

function renderTextWithColors(text: string): ReactNode {
    HEX_COLOR_RE.lastIndex = 0
    const matches = [...text.matchAll(HEX_COLOR_RE)]
    if (!matches.length) return text

    const parts: ReactNode[] = []
    let lastIdx = 0
    let i = 0
    for (const m of matches) {
        const idx = m.index!
        if (idx > lastIdx) parts.push(text.slice(lastIdx, idx))
        parts.push(<ColorChip key={`c${i++}-${idx}`} color={m[0]} />)
        lastIdx = idx + m[0].length
    }
    if (lastIdx < text.length) parts.push(text.slice(lastIdx))
    return parts
}

function processChildren(node: ReactNode): ReactNode {
    if (typeof node === 'string') return renderTextWithColors(node)
    if (Array.isArray(node)) return node.map(processChildren)
    return node
}

function wrapWithColors(Tag: ElementType) {
    return function ColorTag({ children, ...props }: any) {
        return createElement(Tag, props, processChildren(children))
    }
}

const TEXT_TAGS: ElementType[] = ['p', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'td', 'th', 'strong', 'em', 'blockquote']

const colorComponents: Record<string, ReturnType<typeof wrapWithColors>> = {}
for (const tag of TEXT_TAGS) {
    colorComponents[tag as unknown as string] = wrapWithColors(tag)
}

const mdComponents: Components = {
    pre: CodeBlock,
    a: ({ children, ...props }: any) => (
        <ExternalLink {...props}>{processChildren(children)}</ExternalLink>
    ),
    // inline code: process colors only when not inside a fenced code block
    code: ({ className, children, ...props }: any) => {
        if (className && /language-|hljs/.test(className)) {
            return <code className={className} {...props}>{children}</code>
        }
        return <code {...props}>{processChildren(children)}</code>
    },
    ...colorComponents,
}

export default function MarkdownBlock({ content, muted, streaming }: { content: string; muted?: boolean; streaming?: boolean }) {
    const [pending, startTransition] = useTransition()
    const [parsed, setParsed] = useState('')

    useEffect(() => {
        if (!streaming && content) {
            startTransition(() => setParsed(content))
        }
    }, [streaming, content, startTransition])

    if (streaming) {
        return (
            <div className={`markdown-body ${muted ? 'markdown-body--muted' : ''}`}>
                <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                    {content}
                </div>
            </div>
        )
    }

    return (
        <div className={`markdown-body ${muted ? 'markdown-body--muted' : ''}`}>
            {pending && (
                <div className="mb-2" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', opacity: 0.5, fontSize: '13px' }}>
                    {content}
                </div>
            )}
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight]}
                components={mdComponents}
            >
                {parsed || content}
            </ReactMarkdown>
        </div>
    )
}
