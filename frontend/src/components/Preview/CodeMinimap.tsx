import React, { useEffect, useRef, useCallback } from 'react'
import { EditorView } from '@codemirror/view'
import { LspDiagnostic } from '../../lib/lspService'

interface CodeMinimapProps {
    content: string
    totalLines: number
    editorView: EditorView | null
    width?: number
    diagnostics?: LspDiagnostic[]
}

// VS Code Dark+ constants
const BG_COLOR = '#1e1e1e'

// VS Code slider colors (from minimapColors.ts)
const SLIDER_BG = 'rgba(121,121,121,0.2)'
const SLIDER_HOVER_BG = 'rgba(100,100,100,0.35)'
const SLIDER_ACTIVE_BG = 'rgba(100,100,100,0.5)'

// VS Code minimap marker colors (from minimapColors.ts)
const ERROR_COLOR = '#ff1212'   // `new RGBA(255, 18, 18, 0.7)`
const WARN_COLOR = '#e5c07b'

// VS Code minimap char rendering constants (from minimapCharSheet.ts)
const CHAR_WIDTH = 1    // BASE_CHAR_WIDTH
const CHAR_HEIGHT = 2   // BASE_CHAR_HEIGHT
const CODE_COLOR = '#5c6370'

export function CodeMinimap({ content, totalLines, editorView, width = 60, diagnostics = [] }: CodeMinimapProps) {
    const containerRef = useRef<HTMLDivElement>(null)
    const canvasRef = useRef<HTMLCanvasElement>(null)
    const sliderRef = useRef<HTMLDivElement>(null)
    const containerHeightRef = useRef(0)
    const rafId = useRef(0)
    const paintRef = useRef<(() => void) | null>(null)

    // Container height tracking via ResizeObserver
    useEffect(() => {
        const el = containerRef.current
        if (!el) return
        const ro = new ResizeObserver((entries) => {
            for (const entry of entries) {
                containerHeightRef.current = entry.contentRect.height
            }
            paintRef.current?.()
        })
        ro.observe(el)
        return () => ro.disconnect()
    }, [])

    // VS Code _renderLine: character-by-character rendering
    // Spaces/tabs advance x but render nothing; visible chars get a 1x2 pixel block
    const paint = useCallback(() => {
        const canvas = canvasRef.current
        const containerHeight = containerHeightRef.current
        if (!canvas || !containerHeight) return
        const ctx = canvas.getContext('2d')
        if (!ctx) return

        const dpr = window.devicePixelRatio || 1
        canvas.width = width * dpr
        canvas.height = containerHeight * dpr
        ctx.scale(dpr, dpr)

        // Background
        ctx.fillStyle = BG_COLOR
        ctx.fillRect(0, 0, width, containerHeight)

        if (totalLines === 0) return

        const lineH = Math.max(CHAR_HEIGHT + 1, containerHeight / totalLines)
        const maxDx = width - CHAR_WIDTH

        // Build per-line diagnostic severity map
        const lineDiag = new Map<number, number>()
        for (const d of diagnostics) {
            const severity = d.severity || 99
            for (let l = d.startLine; l <= Math.min(d.endLine, totalLines - 1); l++) {
                const existing = lineDiag.get(l)
                if (existing === undefined || severity < existing) {
                    lineDiag.set(l, severity)
                }
            }
        }

        const lines = content.split('\n')

        for (let i = 0; i < totalLines; i++) {
            const y = i * lineH
            const lineText = i < lines.length ? lines[i] : ''
            const severity = lineDiag.get(i)
            const isError = severity !== undefined && severity <= 2

            // Set color for this line's code pixels
            ctx.fillStyle = isError
                ? (severity === 1 ? ERROR_COLOR : WARN_COLOR)
                : CODE_COLOR
            ctx.globalAlpha = isError ? 0.6 : 0.5

            // Per-character rendering (VS Code _renderLine)
            let dx = 0
            for (let ci = 0; ci < lineText.length && dx <= maxDx; ci++) {
                const ch = lineText.charCodeAt(ci)
                if (ch === 0x09) {
                    dx += 4 * CHAR_WIDTH // tab → advance 4 chars
                } else if (ch === 0x20) {
                    dx += CHAR_WIDTH // space → advance, no render
                } else {
                    ctx.fillRect(dx, y, CHAR_WIDTH, CHAR_HEIGHT)
                    dx += CHAR_WIDTH
                }
            }

            ctx.globalAlpha = 1

            // Diagnostic right-edge marker
            if (isError) {
                ctx.fillStyle = severity === 1 ? ERROR_COLOR : WARN_COLOR
                ctx.globalAlpha = 0.7
                ctx.fillRect(width - 3, y, 3, Math.ceil(lineH))
                ctx.globalAlpha = 1
            }
        }
    }, [totalLines, width, diagnostics, content])
    paintRef.current = paint

    // Repaint when deps change
    useEffect(() => { paint() }, [paint])

    // Update slider position from editor scroll (VS Code ratio)
    const updateSlider = useCallback(() => {
        if (!editorView || !sliderRef.current) return
        const scroller = editorView.scrollDOM
        const { scrollTop, scrollHeight, clientHeight: viewportHeight } = scroller
        const containerHeight = containerHeightRef.current
        if (!containerHeight || viewportHeight === 0) return

        const sliderHeightPx = Math.max(1, (viewportHeight / scrollHeight) * containerHeight)
        const maxSliderTop = Math.max(0, containerHeight - sliderHeightPx)
        const scrollRange = Math.max(1, scrollHeight - viewportHeight)
        const sliderTop = (scrollTop / scrollRange) * maxSliderTop

        sliderRef.current.style.height = `${sliderHeightPx}px`
        sliderRef.current.style.top = `${sliderTop}px`
    }, [editorView])

    // Scroll sync from editor
    useEffect(() => {
        if (!editorView) return
        const scroller = editorView.scrollDOM
        const onScroll = () => {
            if (!rafId.current) {
                rafId.current = requestAnimationFrame(() => {
                    rafId.current = 0
                    updateSlider()
                })
            }
        }
        scroller.addEventListener('scroll', onScroll, { passive: true })
        updateSlider()
        return () => {
            scroller.removeEventListener('scroll', onScroll)
            if (rafId.current) cancelAnimationFrame(rafId.current)
        }
    }, [editorView, updateSlider])

    // Click/drag on minimap → scroll editor
    function scrollToY(clientY: number) {
        const ch = containerHeightRef.current
        if (!ch || !editorView) return
        const rect = containerRef.current?.getBoundingClientRect()
        if (!rect) return
        const y = clientY - rect.top
        const ratio = y / ch
        const scroller = editorView.scrollDOM
        scroller.scrollTop = ratio * (scroller.scrollHeight - scroller.clientHeight)
    }

    const handleMouseDown = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
        if ((e.target as HTMLElement).closest('[data-minimap-slider]')) return
        if (!containerHeightRef.current || !editorView) return
        scrollToY(e.clientY)
        const onMove = (ev: MouseEvent) => scrollToY(ev.clientY)
        const onUp = () => {
            window.removeEventListener('mousemove', onMove)
            window.removeEventListener('mouseup', onUp)
        }
        window.addEventListener('mousemove', onMove)
        window.addEventListener('mouseup', onUp)
    }, [editorView])

    // Slider drag
    const onSliderPointerDown = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
        e.preventDefault()
        e.stopPropagation()
        const slider = sliderRef.current
        if (!slider || !editorView) return
        slider.style.background = SLIDER_ACTIVE_BG

        const startY = e.clientY
        const startSliderTop = slider.offsetTop
        const scroller = editorView.scrollDOM
        const ch = containerHeightRef.current
        if (!ch) return

        const onMove = (ev: PointerEvent) => {
            const dy = ev.clientY - startY
            const newTop = Math.max(0, Math.min(ch - slider.offsetHeight, startSliderTop + dy))
            slider.style.top = `${newTop}px`
            const ratio = newTop / Math.max(1, ch - slider.offsetHeight)
            scroller.scrollTop = ratio * (scroller.scrollHeight - scroller.clientHeight)
        }
        const onUp = () => {
            slider.style.background = SLIDER_BG
            window.removeEventListener('pointermove', onMove)
            window.removeEventListener('pointerup', onUp)
        }
        window.addEventListener('pointermove', onMove)
        window.addEventListener('pointerup', onUp)
    }, [editorView])

    return (
        <div
            ref={containerRef}
            onMouseDown={handleMouseDown}
            style={{
                position: 'absolute',
                top: 0,
                right: 0,
                height: '100%',
                width,
                zIndex: 10,
                cursor: 'pointer',
            }}
        >
            <canvas
                ref={canvasRef}
                style={{ width: '100%', height: '100%', display: 'block' }}
            />
            <div
                ref={sliderRef}
                data-minimap-slider
                onPointerDown={onSliderPointerDown}
                onMouseEnter={() => {
                    if (sliderRef.current) sliderRef.current.style.background = SLIDER_HOVER_BG
                }}
                onMouseLeave={() => {
                    if (sliderRef.current) sliderRef.current.style.background = SLIDER_BG
                }}
                style={{
                    position: 'absolute',
                    left: 0,
                    right: 0,
                    height: 0,
                    top: 0,
                    background: SLIDER_BG,
                    pointerEvents: 'auto',
                    cursor: 'pointer',
                    transition: 'background 0.1s ease',
                }}
            >
                <div style={{ position: 'absolute', left: 0, width: '100%', height: '100%', top: 0 }} />
            </div>
        </div>
    )
}
