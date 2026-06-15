import * as monaco from 'monaco-editor'

const GLYPH_MARGIN_CLASS = 'debug-breakpoint-glyph'
const GLYPH_VERIFIED_CLASS = 'debug-breakpoint-verified'
const GLYPH_UNVERIFIED_CLASS = 'debug-breakpoint-unverified'
const CURRENT_LINE_CLASS = 'debug-current-line'

let breakpointDecorations: string[] = []
let currentLineDecorations: string[] = []

export function setBreakpointGlyphs(
    editor: monaco.editor.IStandaloneCodeEditor,
    breakpoints: { file: string; line: number; verified: boolean }[],
    currentFile: string
): void {
    const model = editor.getModel()
    if (!model) return

    const fileBreakpoints = breakpoints.filter(
        (bp) => bp.file === currentFile || model.uri.path === bp.file
    )

    const decorations = fileBreakpoints.map((bp) => {
        const verified = bp.verified
        const glyphClass = verified ? `${GLYPH_MARGIN_CLASS} ${GLYPH_VERIFIED_CLASS}` : `${GLYPH_MARGIN_CLASS} ${GLYPH_UNVERIFIED_CLASS}`

        return {
            range: new monaco.Range(bp.line, 1, bp.line, 1),
            options: {
                isWholeLine: true,
                glyphMarginClassName: glyphClass,
                glyphMarginHoverMessage: { value: verified ? 'Breakpoint (verified)' : 'Breakpoint (unverified)' },
                marginClassName: `debug-breakpoint-margin-${verified ? 'verified' : 'unverified'}`,
                inlineClassName: `debug-breakpoint-inline-${verified ? 'verified' : 'unverified'}`,
                stickiness: monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
                linesDecorationsClassName: `debug-breakpoint-decoration-${verified ? 'verified' : 'unverified'}`,
            },
        }
    })

    breakpointDecorations = editor.deltaDecorations(breakpointDecorations, decorations)
}

export function setCurrentLineHighlight(
    editor: monaco.editor.IStandaloneCodeEditor,
    sourcePath: string | undefined,
    line: number | undefined,
    currentFile: string
): void {
    const model = editor.getModel()
    if (!model) return

    if (!sourcePath || !line) {
        currentLineDecorations = editor.deltaDecorations(currentLineDecorations, [])
        return
    }

    const matches = model.uri.path === sourcePath || sourcePath.endsWith(currentFile)
    if (!matches) {
        currentLineDecorations = editor.deltaDecorations(currentLineDecorations, [])
        return
    }

    const decoration = {
        range: new monaco.Range(line, 1, line, 1),
        options: {
            isWholeLine: true,
            className: CURRENT_LINE_CLASS,
            glyphMarginClassName: 'debug-current-line-arrow',
            stickiness: monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
        },
    }

    currentLineDecorations = editor.deltaDecorations(currentLineDecorations, [decoration])
}

export function clearAllDecorations(editor: monaco.editor.IStandaloneCodeEditor): void {
    breakpointDecorations = editor.deltaDecorations(breakpointDecorations, [])
    currentLineDecorations = editor.deltaDecorations(currentLineDecorations, [])
}
