import { tags, Tag } from '@lezer/highlight'
import { getLanguageForFile, treeSitterInitPromise, getLanguage } from './treeSitter'
import { Parser } from 'web-tree-sitter'

// ── Tag → hex color (Catppuccin Mocha palette) ────────────────────────────
// Pre-computed at module load so compound tags (e.g. function(variableName))
// are available for direct lookup.

const TAG_TO_HEX_NAMES: Record<string, string> = {
    comment: '#6c7086',
    docComment: '#6c7086',
    string: '#a6e3a1',
    escape: '#a6e3a1',
    meta: '#cba6f7',
    number: '#fab387',
    literal: '#fab387',
    bool: '#fab387',
    color: '#fab387',
    unit: '#fab387',
    keyword: '#cba6f7',
    moduleKeyword: '#cba6f7',
    controlKeyword: '#cba6f7',
    definitionKeyword: '#cba6f7',
    self: '#f38ba8',
    typeName: '#f9e2af',
    className: '#f9e2af',
    namespace: '#f9e2af',
    variableName: '#cdd6f4',
    propertyName: '#cdd6f4',
    // Compound / function-modifier variants
    function: '#89b4fa',
    standard: '#89b4fa',
    attributeName: '#f5c2e7',
    labelName: '#89b4fa',
}

const tagHexMap = new Map<Tag, string>()

function buildTagHexMap() {
    if (tagHexMap.size > 0) return // already built

    // Register simple tags by their string name
    for (const [name, value] of Object.entries(tags)) {
        if (typeof value === 'function') continue // modifiers, not tags
        const hex = TAG_TO_HEX_NAMES[name]
        if (hex) tagHexMap.set(value as Tag, hex)
    }

    // Register compound tags needed by resolveTag
    const fnMod = tags.function as (t: Tag) => Tag
    const fnVar = fnMod(tags.variableName as Tag)
    const fnProp = fnMod(tags.propertyName as Tag)
    if (TAG_TO_HEX_NAMES['function']) {
        tagHexMap.set(fnVar, TAG_TO_HEX_NAMES['function'])
        tagHexMap.set(fnProp, TAG_TO_HEX_NAMES['function'])
    }
}

buildTagHexMap()

function tagToHex(tag: Tag | null): string | null {
    if (!tag) return null
    // Check the tag itself and walk its ancestor chain via tag.set
    for (const t of tag.set) {
        const hex = tagHexMap.get(t)
        if (hex) return hex
    }
    return null
}

// ── Node type → tag (simplified from treeSitterHighlight.resolveTag) ─────

type TagEntry = [string, Tag]

const patternCache = new Map<string, { re: RegExp; isExact: boolean }>()

function escapeRegex(s: string): string {
    return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function compilePattern(p: string): { re: RegExp; isExact: boolean } {
    const cached = patternCache.get(p)
    if (cached) return cached
    let isExact = false
    let re: RegExp
    if (p.startsWith('*') && p.endsWith('*')) {
        re = new RegExp(escapeRegex(p.slice(1, -1)))
    } else if (p.endsWith('*')) {
        re = new RegExp('^' + escapeRegex(p.slice(0, -1)))
    } else if (p.startsWith('*')) {
        re = new RegExp(escapeRegex(p.slice(1)) + '$')
    } else {
        re = new RegExp('^' + escapeRegex(p) + '$')
        isExact = true
    }
    const entry = { re, isExact }
    patternCache.set(p, entry)
    return entry
}

const tagMap: TagEntry[] = [
    ['comment', tags.comment],
    ['*comment', tags.comment],
    ['*doc_comment*', tags.docComment],
    ['string', tags.string],
    ['interpreted_string_literal', tags.string],
    ['string_literal', tags.string],
    ['string_fragment', tags.string],
    ['string_*', tags.string],
    ['*_string', tags.string],
    ['escape_sequence', tags.escape],
    ['interpolation', tags.meta],
    ['number', tags.number],
    ['int_literal', tags.number],
    ['float_literal', tags.number],
    ['*_number', tags.number],
    ['boolean', tags.bool],
    ['keyword', tags.keyword],
    ['keyword_*', tags.keyword],
    ['*keyword', tags.keyword],
    ['if', tags.controlKeyword],
    ['else', tags.controlKeyword],
    ['for', tags.controlKeyword],
    ['while', tags.controlKeyword],
    ['do', tags.controlKeyword],
    ['switch', tags.controlKeyword],
    ['case', tags.controlKeyword],
    ['default', tags.keyword],
    ['break', tags.controlKeyword],
    ['continue', tags.controlKeyword],
    ['return', tags.controlKeyword],
    ['match', tags.controlKeyword],
    ['let', tags.definitionKeyword],
    ['var', tags.definitionKeyword],
    ['const', tags.definitionKeyword],
    ['fn', tags.definitionKeyword],
    ['function_definition', tags.definitionKeyword],
    ['func', tags.definitionKeyword],
    ['def', tags.definitionKeyword],
    ['type_definition', tags.definitionKeyword],
    ['type', tags.definitionKeyword],
    ['struct', tags.definitionKeyword],
    ['enum', tags.definitionKeyword],
    ['trait', tags.definitionKeyword],
    ['interface', tags.definitionKeyword],
    ['impl', tags.definitionKeyword],
    ['self', tags.self],
    ['type_identifier', tags.typeName],
    ['type_*', tags.typeName],
    ['class_*', tags.className],
    ['namespace_*', tags.namespace],
    ['identifier', tags.variableName],
    ['field_identifier', tags.propertyName],
    ['property_identifier', tags.propertyName],
    ['(', tags.paren],
    [')', tags.paren],
    ['{', tags.brace],
    ['}', tags.brace],
    ['[', tags.bracket],
    [']', tags.bracket],
    [';', tags.separator],
    [',', tags.separator],
    ['.', tags.punctuation],
    ['operator', tags.operator],
    ['*_operator', tags.operator],
    ['attribute', tags.attributeName],
    ['*_attribute', tags.attributeName],
]

function matchTag(nt: string): Tag | null {
    const exact: TagEntry[] = []
    const wild: TagEntry[] = []
    for (const [p, tag] of tagMap) {
        const { isExact } = compilePattern(p)
            ; (isExact ? exact : wild).push([p, tag])
    }
    for (const [p, tag] of exact) {
        const { re } = patternCache.get(p)!
        if (re.test(nt)) return tag
    }
    for (const [p, tag] of wild) {
        const { re } = patternCache.get(p)!
        if (re.test(nt)) return tag
    }
    return null
}

function resolveTag(nt: string, parentStack: string[]): Tag | null {
    const pLen = parentStack.length
    const parent = pLen > 0 ? parentStack[pLen - 1] : ''
    const grandparent = pLen > 1 ? parentStack[pLen - 2] : ''

    if (nt === 'identifier') {
        if (parent === 'call_expression') return tags.function(tags.variableName)
        return null
    }

    if (nt === 'field_identifier' || nt === 'property_identifier') {
        if (parent === 'selector_expression' && grandparent === 'call_expression') {
            return tags.function(tags.variableName)
        }
    }

    return matchTag(nt)
}

// ── Public API ───────────────────────────────────────────────────────────

export interface ColorSpan {
    from: number // character offset relative to line start
    to: number
    color: string // hex color
}

export interface LineColors {
    line: number // 0-based line number
    spans: ColorSpan[]
}

export async function getLineColors(
    content: string,
    filePath: string
): Promise<LineColors[]> {
    const result: LineColors[] = []
    const langName = getLanguageForFile(filePath)
    const lines = content.split('\n')
    if (!langName) {
        // No language — all base color
        // No language — all base color
        const lines = content.split('\n')
        for (let i = 0; i < lines.length; i++) {
            result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#cdd6f4' }] })
        }
        return result
    }

    let parser: Parser
    try {
        await treeSitterInitPromise
        parser = new Parser()
    } catch {
        // Fallback to monochrome
        const lines = content.split('\n')
        for (let i = 0; i < lines.length; i++) {
            result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#cdd6f4' }] })
        }
        return result
    }

    const lang = await getLanguage(langName)
    if (!lang) {
        parser.delete()
        const lines = content.split('\n')
        for (let i = 0; i < lines.length; i++) {
            result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#cdd6f4' }] })
        }
        return result
    }

    parser.setLanguage(lang)
    const tree = parser.parse(content)
    if (!tree) {
        parser.delete()
        const lines = content.split('\n')
        for (let i = 0; i < lines.length; i++) {
            result.push({ line: i, spans: [{ from: 0, to: lines[i].length, color: '#cdd6f4' }] })
        }
        return result
    }

    const root = tree.rootNode

    // Precompute line start byte offsets for normalizing spans to line-relative coords
    const lineStartOffsets: number[] = []
    let offset = 0
    for (const line of lines) {
        lineStartOffsets.push(offset)
        offset += line.length + 1 // +1 for '\n'
    }

    // Walk the syntax tree and collect colored spans per line
    const lineColors = new Map<number, { from: number; to: number; color: string }[]>()
    for (let i = 0; i < lines.length; i++) {
        lineColors.set(i, [])
    }

    function walk(node: any, parents: string[]) {
        const nt = node.type || ''
        const tag = resolveTag(nt, parents)
        const hex = tagToHex(tag)
        if (hex) {
            const startLine = content.slice(0, node.startIndex).split('\n').length - 1
            const endLine = content.slice(0, node.endIndex).split('\n').length - 1
            if (startLine === endLine) {
                const lineStart = lineStartOffsets[startLine]
                const col = lineColors.get(startLine)
                if (col) {
                    col.push({ from: node.startIndex - lineStart, to: node.endIndex - lineStart, color: hex })
                }
            }
        }
        parents.push(nt)
        if (node.children) {
            for (const child of node.children) {
                walk(child, parents)
            }
        }
        parents.pop()
    }

    if (root.children) {
        for (const child of root.children) {
            walk(child, [])
        }
    }

    tree.delete()
    parser.delete()

    // Build result: for each line, merge overlapping spans and fill gaps with base color
    for (let i = 0; i < lines.length; i++) {
        const spans = lineColors.get(i) || []
        // Sort by from
        spans.sort((a, b) => a.from - b.from)
        // Merge overlapping spans
        const merged: { from: number; to: number; color: string }[] = []
        let prev = spans[0]
        if (prev) {
            merged.push(prev)
            for (let j = 1; j < spans.length; j++) {
                const cur = spans[j]
                if (cur.from <= prev.to && cur.from >= prev.from) {
                    // Same color, extend
                    if (cur.to > prev.to && cur.color === prev.color) {
                        prev.to = cur.to
                    }
                    // Different color on same range — keep first
                } else {
                    merged.push(cur)
                    prev = cur
                }
            }
        }
        result.push({ line: i, spans: merged.length > 0 ? merged : [{ from: 0, to: lines[i].length, color: '#cdd6f4' }] })
    }

    return result
}
