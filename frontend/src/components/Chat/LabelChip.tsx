// Inline SVG icons for contentEditable chip rendering
const ICON_SVG: Record<string, string> = {
    command: `<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:middle;display:inline-block"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`,
    file: `<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:middle;display:inline-block"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/></svg>`,
    paste: `<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:middle;display:inline-block"><path d="M15 2H9a1 1 0 0 0-1 1v2a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1Z"/><path d="M8 4H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2h-2"/></svg>`,
    skill: `<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:middle;display:inline-block"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`,
}

export function renderChipHTML(type: string, text: string): string {
    const svg = ICON_SVG[type] || ''
    return `<span contenteditable="false" data-type="${type}" style="display:inline;background:var(--bg-elevated);box-shadow:0 0 0 1px var(--border-strong);color:var(--text-primary);border-radius:4px;padding:0 4px;white-space:nowrap;font-size:inherit;line-height:inherit;font-family:inherit;user-select:none">${svg}${escapeHTML(text)}</span>`
}

function escapeHTML(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

export interface LabelRegion {
    type: 'command' | 'file' | 'paste' | 'skill'
    label: string
    start: number
    end: number
}

/** Parse raw text and find label-able regions */
export function findLabels(text: string, skillNames?: Set<string>): LabelRegion[] {
    const regions: LabelRegion[] = []

    // 1. Commands at line start: /command-name
    const cmdRe = /^\/([\w][\w-]*)(?:\s|$)/gm
    let m: RegExpExecArray | null
    while ((m = cmdRe.exec(text)) !== null) {
        const name = m[1]
        const isSkill = skillNames?.has(name)
        regions.push({ type: isSkill ? 'skill' : 'command', label: name, start: m.index, end: m.index + m[0].length })
    }

    // 2. @ file paths (contains / or \ or has extension)
    const atRe = /@(\S+?)(?:\s|$)/g
    while ((m = atRe.exec(text)) !== null) {
        const raw = m[1]
        const name = raw.replace(/^.*[/\\]/, '')
        if (raw.includes('/') || raw.includes('\\') || /\.[a-zA-Z0-9]{1,8}$/.test(raw)) {
            regions.push({ type: 'file', label: name, start: m.index, end: m.index + m[0].length })
        }
    }

    // 3. Paste markers: [Paste N]
    const pasteRe = /\[Paste (\d+)\]/g
    while ((m = pasteRe.exec(text)) !== null) {
        regions.push({ type: 'paste', label: `Paste ${m[1]}`, start: m.index, end: m.index + m[0].length })
    }

    // 4. File markers: [filename.ext] or [filename.ext START~END] or [path/to/file.ext START~END]
    const fileRe = /\[([^\]]+?\.\w{1,8}(?:\s+\d+~\d+)?)\]/g
    while ((m = fileRe.exec(text)) !== null) {
        regions.push({ type: 'file', label: m[1], start: m.index, end: m.index + m[0].length })
    }

    // 5. General bracketed names (e.g. pasted directory paths without extensions)
    const bracketRe = /\[([\w][\w.-]*)\]/g
    while ((m = bracketRe.exec(text)) !== null) {
        regions.push({ type: 'file', label: m[1], start: m.index, end: m.index + m[0].length })
    }

    // Sort by start position and remove overlaps
    regions.sort((a, b) => a.start - b.start)
    const filtered: LabelRegion[] = []
    for (const r of regions) {
        if (filtered.length === 0 || r.start >= filtered[filtered.length - 1].end) {
            filtered.push(r)
        }
    }

    return filtered
}

