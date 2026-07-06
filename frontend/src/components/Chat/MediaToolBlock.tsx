import React, { useState } from 'react'
import MarkdownBlock from './MarkdownBlock'
import { IconVideo, IconImage, IconChevronDown, IconChevronRight, IconClock, IconPlay } from '../Icons'

interface ToolCall {
    id?: string
    name: string
    input: string
    output?: string
    status: 'running' | 'done' | 'error'
}

interface VideoResult {
    filePath?: string
    fileName?: string
    duration_seconds?: number
    frame_count?: number
    frame_interval_seconds?: number
    question?: string
    summary?: string
    timeline?: { t: number; what: string }[]
    key_moments?: { t: number; title: string; description: string }[]
    thumbnails?: { t: number; url: string }[]
    // image_understand uses these instead of thumbnails[]/timeline
    mimeType?: string
    size?: number
    thumbnail?: string
    error?: string
}

// Header style mirrors the existing ToolBlock badge conventions so media tools
// feel native alongside the other tool outputs in the chat stream.
const MEDIA_STYLE: Record<string, { color: string; label: string }> = {
    video_understand: { color: '#c084fc', label: 'video' },   // purple
    image_understand: { color: '#22d3ee', label: 'image' },   // cyan
}

const HEADER_BG = 'var(--bg-sidebar)'
const BORDER = '1px solid var(--border)'

interface MediaToolBlockProps {
    tool: ToolCall
    onOpenMedia?: (filePath: string, fileName: string, mime: string) => void
}

function formatDuration(seconds?: number): string {
    if (!seconds || !isFinite(seconds)) return '—'
    if (seconds >= 60) {
        const m = Math.floor(seconds / 60)
        const s = Math.round(seconds % 60)
        return `${m}m ${s}s`
    }
    return `${seconds.toFixed(0)}s`
}

function formatTimestamp(t: number): string {
    const mm = Math.floor(t / 60)
    const ss = Math.floor(t % 60)
    return `${mm}:${ss.toString().padStart(2, '0')}`
}

export function MediaToolBlock({ tool, onOpenMedia }: MediaToolBlockProps) {
    const isVideo = tool.name === 'video_understand'
    const isImage = tool.name === 'image_understand'
    if (!isVideo && !isImage) return null

    const isRunning = tool.status === 'running'
    const isError = tool.status === 'error'
    const style = MEDIA_STYLE[tool.name] || MEDIA_STYLE.video_understand

    let parsed: VideoResult | null = null
    if (tool.output && !isRunning) {
        try {
            parsed = JSON.parse(tool.output) as VideoResult
        } catch {
            parsed = null
        }
    }

    const handleOpenMedia = () => {
        if (!parsed?.filePath || !onOpenMedia) return
        const mime = isVideo ? 'video/mp4' : 'image/png'
        onOpenMedia(parsed.filePath, parsed.fileName || 'media', mime)
    }

    return (
        <div
            className="rounded-md overflow-hidden text-[13px]"
            style={{ background: HEADER_BG, border: BORDER }}
        >
            <div className="flex items-center gap-2 px-3 py-2" style={{ borderBottom: isRunning || (parsed && !isError) ? BORDER : 'none' }}>
                {isVideo ? <IconVideo size={14} style={{ color: style.color }} /> : <IconImage size={14} style={{ color: style.color }} />}
                <span
                    className="text-[10px] font-mono px-1.5 py-0.5 rounded uppercase tracking-wide"
                    style={{ background: 'rgba(255,255,255,0.04)', color: style.color, border: `1px solid ${style.color}40` }}
                >
                    {style.label}
                </span>
                <span className="font-medium truncate flex-1" style={{ color: 'var(--text)' }}>
                    {parsed?.fileName || (isVideo ? 'Video' : 'Image')}
                </span>
                {parsed?.duration_seconds != null && isVideo && (
                    <span className="text-[10px] flex items-center gap-1" style={{ color: 'var(--text-dim)' }}>
                        <IconClock size={11} />
                        {formatDuration(parsed.duration_seconds)}
                    </span>
                )}
                {parsed?.filePath && onOpenMedia && !isRunning && (
                    <button
                        type="button"
                        onClick={handleOpenMedia}
                        className="text-[10px] px-1.5 py-0.5 rounded hover:bg-white/5"
                        style={{ color: 'var(--text-dim)', border: BORDER }}
                        title="Open in preview panel"
                    >
                        Open
                    </button>
                )}
                <span
                    className="text-[10px] font-semibold uppercase"
                    style={{ color: isError ? 'var(--red)' : isRunning ? 'var(--yellow)' : 'var(--text-dim)' }}
                >
                    {isError ? 'error' : isRunning ? 'analyzing…' : 'done'}
                </span>
            </div>

            {(isRunning || isError || parsed || tool.output) && (
                <div className="px-3 py-2.5">
                    {isRunning && <RunningState isVideo={isVideo} parsed={parsed} />}
                    {isError && <ErrorState output={tool.output} />}
                    {!isRunning && !isError && parsed && (
                        <ResultState toolName={tool.name} parsed={parsed} onOpenMedia={onOpenMedia} />
                    )}
                    {!isRunning && !isError && !parsed && tool.output && (
                        <div className="text-[13px]" style={{ color: 'var(--text)' }}>
                            <MarkdownBlock content={tool.output} streaming={false} />
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}

function RunningState({ isVideo, parsed }: { isVideo: boolean; parsed: VideoResult | null }) {
    const phase = isVideo ? 'Sampling key frames and analyzing with vision model…' : 'Analyzing image…'
    return (
        <div className="flex items-center gap-2" style={{ color: 'var(--text-dim)' }}>
            <SpinnerDot />
            <span>{phase}</span>
            {parsed?.frame_count != null && (
                <span className="text-[10px] ml-1" style={{ color: 'var(--text-faint, var(--text-dim))' }}>
                    ({parsed.frame_count} frames)
                </span>
            )}
        </div>
    )
}

function SpinnerDot() {
    return (
        <span
            className="inline-block w-2 h-2 rounded-full"
            style={{ background: 'var(--accent)', animation: 'label-blink 1.4s ease-in-out infinite' }}
        />
    )
}

function ErrorState({ output }: { output?: string }) {
    let msg = output || 'Tool failed'
    try {
        const j = JSON.parse(output || '{}')
        if (j && typeof j.error === 'string') msg = j.error
    } catch { /* keep raw */ }
    return (
        <div className="text-[12px]" style={{ color: 'var(--red)' }}>
            {msg}
        </div>
    )
}

interface ResultStateProps {
    toolName: string
    parsed: VideoResult
    onOpenMedia?: (filePath: string, fileName: string, mime: string) => void
}

function ResultState({ toolName, parsed, onOpenMedia }: ResultStateProps) {
    if (toolName === 'image_understand') {
        return <ImageResult parsed={parsed} onOpenMedia={onOpenMedia} />
    }
    return <VideoResultView parsed={parsed} onOpenMedia={onOpenMedia} />
}

function ImageResult({ parsed, onOpenMedia }: { parsed: VideoResult; onOpenMedia?: (filePath: string, fileName: string, mime: string) => void }) {
    const thumb = parsed.thumbnail || parsed.thumbnails?.[0]?.url
    const handleClick = () => {
        if (parsed.filePath && onOpenMedia) {
            onOpenMedia(parsed.filePath, parsed.fileName || 'image', parsed.mimeType || 'image/png')
        }
    }
    return (
        <div className="space-y-2">
            {thumb && (
                <button
                    type="button"
                    onClick={handleClick}
                    className="block w-full rounded overflow-hidden text-left"
                    style={{ background: 'rgba(255,255,255,0.02)', maxHeight: 240, cursor: onOpenMedia ? 'zoom-in' : 'default', border: 0, padding: 0 }}
                    disabled={!onOpenMedia}
                >
                    <img src={thumb} alt={parsed.fileName || 'image'} style={{ width: '100%', maxHeight: 240, objectFit: 'contain', display: 'block' }} />
                </button>
            )}
            {parsed.summary && (
                <MarkdownBlock content={parsed.summary} streaming={false} />
            )}
        </div>
    )
}

function VideoResultView({ parsed, onOpenMedia }: { parsed: VideoResult; onOpenMedia?: (filePath: string, fileName: string, mime: string) => void }) {
    const thumbs = parsed.thumbnails || []
    return (
        <div className="space-y-3">
            {thumbs.length > 0 && (
                <ThumbStrip
                    thumbs={thumbs}
                    onOpen={() => {
                        if (parsed.filePath && onOpenMedia) {
                            onOpenMedia(parsed.filePath, parsed.fileName || 'video', 'video/mp4')
                        }
                    }}
                />
            )}
            {parsed.summary && (
                <div>
                    <MarkdownBlock content={parsed.summary} streaming={false} />
                </div>
            )}
            {parsed.key_moments && parsed.key_moments.length > 0 && (
                <CollapsibleSection title={`Key moments (${parsed.key_moments.length})`} defaultOpen>
                    <ul className="space-y-2">
                        {parsed.key_moments.map((km, i) => (
                            <li key={i} className="flex gap-2 items-start text-[12px]">
                                <span
                                    className="font-mono text-[10px] px-1 py-0.5 rounded shrink-0 mt-0.5"
                                    style={{ background: 'rgba(192,132,252,0.08)', color: '#c084fc', border: '1px solid rgba(192,132,252,0.2)' }}
                                >
                                    {formatTimestamp(km.t)}
                                </span>
                                <div className="flex-1 min-w-0">
                                    <div className="font-medium" style={{ color: 'var(--text)' }}>{km.title}</div>
                                    {km.description && (
                                        <div style={{ color: 'var(--text-dim)' }}>{km.description}</div>
                                    )}
                                </div>
                            </li>
                        ))}
                    </ul>
                </CollapsibleSection>
            )}
            {parsed.timeline && parsed.timeline.length > 0 && (
                <CollapsibleSection title={`Timeline (${parsed.timeline.length})`}>
                    <ul className="space-y-1">
                        {parsed.timeline.map((tl, i) => (
                            <li key={i} className="flex gap-2 items-baseline text-[12px]">
                                <span
                                    className="font-mono text-[10px] shrink-0"
                                    style={{ color: 'var(--text-faint, var(--text-dim))', minWidth: 38 }}
                                >
                                    {formatTimestamp(tl.t)}
                                </span>
                                <span style={{ color: 'var(--text-dim)' }}>{tl.what}</span>
                            </li>
                        ))}
                    </ul>
                </CollapsibleSection>
            )}
        </div>
    )
}

function ThumbStrip({ thumbs, onOpen }: { thumbs: { t: number; url: string }[]; onOpen: () => void }) {
    return (
        <div
            className="flex gap-1.5 overflow-x-auto py-1"
            style={{ scrollbarWidth: 'thin' }}
        >
            {thumbs.map((th, i) => (
                <ThumbCell key={i} t={th.t} url={th.url} onOpen={onOpen} />
            ))}
        </div>
    )
}

function ThumbCell({ t, url, onOpen }: { t: number; url: string; onOpen: () => void }) {
    const [hover, setHover] = useState(false)
    return (
        <button
            type="button"
            onMouseEnter={() => setHover(true)}
            onMouseLeave={() => setHover(false)}
            onClick={onOpen}
            className="relative shrink-0 rounded overflow-hidden group"
            style={{
                width: 110, height: 64,
                background: 'rgba(255,255,255,0.04)',
                border: BORDER,
                cursor: 'pointer',
                padding: 0,
            }}
            title={`Open video (frame at ${formatTimestamp(t)})`}
        >
            <img src={url} alt={`Frame at ${formatTimestamp(t)}`} style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }} />
            <span
                className="absolute bottom-0 left-0 right-0 text-[9px] font-mono text-center py-0.5"
                style={{ background: 'rgba(0,0,0,0.65)', color: '#fff' }}
            >
                {formatTimestamp(t)}
            </span>
            {hover && (
                <span
                    className="absolute inset-0 flex items-center justify-center pointer-events-none"
                    style={{ background: 'rgba(0,0,0,0.4)' }}
                >
                    <IconPlay size={18} />
                </span>
            )}
        </button>
    )
}

function CollapsibleSection({ title, defaultOpen = false, children }: { title: string; defaultOpen?: boolean; children: React.ReactNode }) {
    const [open, setOpen] = useState(defaultOpen)
    return (
        <div className="rounded" style={{ border: BORDER }}>
            <button
                type="button"
                onClick={() => setOpen(o => !o)}
                className="w-full flex items-center gap-1.5 px-2 py-1.5 text-left text-[11px] font-semibold uppercase tracking-wide"
                style={{ color: 'var(--text-dim)', background: 'rgba(255,255,255,0.02)', border: 0 }}
            >
                {open ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
                {title}
            </button>
            {open && <div className="px-2 py-2">{children}</div>}
        </div>
    )
}

export default MediaToolBlock
