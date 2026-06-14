import type { editor } from 'monaco-editor'
import { getLineColors } from './treeSitterColors'

const DEFAULT_COLOR = '#cdd6f4'
const COMMENT_COLOR = '#6c7086'

const colorToClass = new Map<string, string>()
let styleEl: HTMLStyleElement | null = null

function ensureStyleEl(): HTMLStyleElement {
    if (styleEl) return styleEl
    styleEl = document.createElement('style')
    styleEl.id = 'monika-monaco-ts-deco'
    document.head.appendChild(styleEl)
    return styleEl
}

function classForColor(hex: string): string {
    let cls = colorToClass.get(hex)
    if (cls) return cls
    cls = 'tsd-' + hex.replace('#', '')
    const italic = hex === COMMENT_COLOR ? ';font-style:italic' : ''
    ensureStyleEl().textContent += `.${cls}{color:${hex}${italic}!important}`
    colorToClass.set(hex, cls)
    return cls
}

export async function applyTreeSitterDecorations(
    editorInstance: editor.IStandaloneCodeEditor,
    content: string,
    filePath: string,
    oldIds: string[],
): Promise<string[]> {
    const lineColors = await getLineColors(content, filePath)

    const decorations: editor.IModelDeltaDecoration[] = []
    for (const { line, spans } of lineColors) {
        for (const s of spans) {
            if (s.color === DEFAULT_COLOR) continue
            decorations.push({
                range: {
                    startLineNumber: line + 1,
                    startColumn: s.from + 1,
                    endLineNumber: line + 1,
                    endColumn: s.to + 1,
                },
                options: { inlineClassName: classForColor(s.color) },
            })
        }
    }

    return editorInstance.deltaDecorations(oldIds, decorations)
}
