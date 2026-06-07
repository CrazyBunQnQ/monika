import { Range, StateEffect, StateField } from '@codemirror/state'
import { Decoration, DecorationSet, EditorView } from '@codemirror/view'
import { LspDiagnostic } from './lspService'

const setDiagnostics = StateEffect.define<{ diags: LspDiagnostic[]; doc: string }>()

function diagStyle(diag: LspDiagnostic): string {
    if (diag.severity === 1) return 'text-decoration: underline wavy #e06c75'
    if (diag.severity === 2) return 'text-decoration: underline wavy #e5c07b'
    if (diag.severity === 3) return 'text-decoration: underline dashed #61afef'
    return 'text-decoration: underline dotted #5c6370'
}

export const lspDiagnosticField = StateField.define<DecorationSet>({
    create() { return Decoration.none },
    update(decos, tr) {
        for (const e of tr.effects) {
            if (e.is(setDiagnostics)) {
                const ranges: Range<Decoration>[] = []
                const lines = e.value.doc.split('\n')
                for (const d of e.value.diags) {
                    const fromLine = Math.max(0, Math.min(d.startLine, lines.length - 1))
                    const toLine = Math.max(0, Math.min(d.endLine, lines.length - 1))
                    let from = 0
                    for (let i = 0; i < fromLine; i++) from += lines[i].length + 1
                    from += Math.min(d.startCol, lines[fromLine].length)
                    let to = 0
                    for (let i = 0; i < toLine; i++) to += lines[i].length + 1
                    to += Math.min(d.endCol, lines[toLine].length)
                    if (to <= from) to = from + 1
                    ranges.push(
                        Decoration.mark({ attributes: { style: diagStyle(d), title: d.message } }).range(from, to)
                    )
                }
                return Decoration.set(ranges, true)
            }
        }
        return decos.map(tr.changes)
    },
    provide: f => EditorView.decorations.from(f),
})

export function updateLspDiagnostics(view: EditorView, diags: LspDiagnostic[]) {
    const doc = view.state.doc.toString()
    view.dispatch({ effects: setDiagnostics.of({ diags, doc }) })
}
