import { tags, Tag } from '@lezer/highlight'
import { Decoration, DecorationSet, EditorView, ViewPlugin, ViewUpdate } from '@codemirror/view'
import { Language, Parser } from 'web-tree-sitter'

// ── Tag map: tree-sitter node type patterns → CodeMirror tags ────────────

// ── Tag map: tree-sitter node type patterns → CodeMirror tags ────────────

type TagEntry = [string, Tag]

const tagMap: TagEntry[] = [
    // Comments
    ['comment', tags.comment],
    ['*comment', tags.comment],
    ['*doc_comment*', tags.docComment],

    // Strings (exact matches BEFORE wildcards, so string literals beat *_literal)
    ['string', tags.string],
    ['interpreted_string_literal', tags.string],
    ['raw_string_literal', tags.string],
    ['string_literal', tags.string],
    ['interpreted_string_literal_content', tags.string],
    ['string_*', tags.string],
    ['*_string', tags.string],
    ['string_fragment', tags.string],
    ['escape_sequence', tags.escape],
    ['interpolation', tags.meta],

    // Numbers
    ['number', tags.number],
    ['int_literal', tags.number],
    ['float_literal', tags.number],
    ['*_number', tags.number],
    ['boolean', tags.bool],
    ['keyword', tags.keyword],
    ['keyword_*', tags.keyword],
    ['*keyword', tags.keyword],   // suffix: xxx_keyword
    // Individual keyword nodes (most tree-sitter grammars use keyword text as node type)
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
    // Declarations
    ['let', tags.definitionKeyword],
    ['var', tags.definitionKeyword],
    ['const', tags.definitionKeyword],
    ['func', tags.definitionKeyword],
    ['fn', tags.definitionKeyword],
    ['def', tags.definitionKeyword],
    ['function', tags.definitionKeyword],
    ['class', tags.definitionKeyword],
    ['struct', tags.definitionKeyword],
    ['enum', tags.definitionKeyword],
    ['trait', tags.definitionKeyword],
    ['impl', tags.definitionKeyword],
    ['interface', tags.definitionKeyword],
    ['type', tags.definitionKeyword],
    ['package', tags.moduleKeyword],
    ['import', tags.moduleKeyword],
    ['from', tags.moduleKeyword],
    ['export', tags.definitionKeyword],
    ['pub', tags.definitionKeyword],
    ['module', tags.moduleKeyword],
    // Modifiers
    ['async', tags.keyword],
    ['await', tags.keyword],
    ['pub', tags.keyword],
    ['mut', tags.keyword],
    ['ref', tags.keyword],
    ['static', tags.keyword],
    ['abstract', tags.keyword],
    ['virtual', tags.keyword],
    ['override', tags.keyword],
    ['defer', tags.keyword],
    ['go', tags.keyword],
    ['select', tags.keyword],
    ['range', tags.keyword],
    ['yield', tags.keyword],
    ['using', tags.keyword],
    ['as', tags.keyword],
    ['in', tags.keyword],
    ['of', tags.keyword],
    ['is', tags.keyword],

    // Literal keywords (nil, None, true, false, etc.)
    ['nil', tags.literal],
    ['None', tags.literal],
    ['true', tags.bool],
    ['false', tags.bool],
    ['null', tags.literal],
    ['undefined', tags.literal],
    ['self', tags.self],
    ['this', tags.self],
    ['super', tags.self],
    // Type identifiers & type-like nodes
    ['type_identifier', tags.typeName],
    ['qualified_type', tags.typeName],
    ['slice_type', tags.typeName],
    ['pointer_type', tags.typeName],
    ['array_type', tags.typeName],
    ['map_type', tags.typeName],
    ['channel_type', tags.typeName],
    ['generic_type', tags.typeName],
    ['type_arguments', tags.typeName],
    ['type_parameter', tags.typeName],

    // Identifiers & names (generic 'identifier' handled context-aware in resolveTag)
    ['field_identifier', tags.propertyName],
    ['property_identifier', tags.propertyName],
    ['package_identifier', tags.namespace],
    ['variable_name', tags.variableName],
    ['field_declaration', tags.propertyName],
    ['attribute', tags.attributeName],
    ['attribute_value', tags.attributeValue],
    ['doctype', tags.processingInstruction],
    ['entity', tags.character],

    // CSS
    ['property_name', tags.propertyName],
    ['class_selector', tags.className],
    ['tag_selector', tags.tagName],
    ['at_rule', tags.keyword],
    ['color_value', tags.color],
    ['unit', tags.unit],

    // Markdown
    ['heading', tags.heading],
    ['emphasis', tags.emphasis],
    ['strong', tags.strong],
    ['strikethrough', tags.strikethrough],
    ['link', tags.link],
    ['inline_code', tags.monospace],
    ['fenced_code', tags.monospace],
    ['quote', tags.quote],
    ['list_marker', tags.list],
    ['thematic_break', tags.contentSeparator],

    // Operators
    ['operator', tags.operator],
    ['*operator*', tags.operator],

    // Punctuation
    ['punctuation', tags.punctuation],
    ['delimiter', tags.punctuation],
    ['comma', tags.separator],
    ['semicolon', tags.separator],
    ['bracket', tags.bracket],

    // Regex
    ['regex', tags.regexp],
    ['regex_*', tags.regexp],
    ['*regex*', tags.regexp],

    // Meta / preprocessor
    ['shebang', tags.meta],
    ['preproc_*', tags.meta],
    ['include', tags.meta],

    // Shell
    ['command_name', tags.variableName],
    ['command_substitution', tags.meta],
    ['redirect', tags.operator],
    ['environment_variable', tags.variableName],
    ['variable', tags.variableName],

    // YAML/TOML
    ['pair', tags.propertyName],
    ['pair_key', tags.propertyName],
    ['bare_key', tags.propertyName],
    ['quoted_key', tags.propertyName],
]

const exactTags = new Map<string, Tag>()
const wildcardTags: [string, string, Tag][] = [] // [prefix*, *suffix, *contains*]

function initTagLookup() {
    if (exactTags.size > 0) return
    for (const [pattern, tag] of tagMap) {
        const hasLhs = pattern.startsWith('*')
        const hasRhs = pattern.endsWith('*')
        if (hasLhs && hasRhs) {
            wildcardTags.push(['*', pattern.slice(1, -1), tag])
        } else if (hasLhs) {
            wildcardTags.push(['suffix', pattern.slice(1), tag])
        } else if (hasRhs) {
            wildcardTags.push(['prefix', pattern.slice(0, -1), tag])
        } else {
            exactTags.set(pattern, tag)
        }
    }
}

function matchTag(nodeType: string): Tag | null {
    initTagLookup()
    const exact = exactTags.get(nodeType)
    if (exact) return exact
    for (const [kind, pat, tag] of wildcardTags) {
        if (kind === '*' && nodeType.includes(pat)) return tag
        if (kind === 'prefix' && nodeType.startsWith(pat)) return tag
        if (kind === 'suffix' && nodeType.endsWith(pat)) return tag
    }
    return null
}

// ── Context-aware tag resolution (uses parent node types) ──────────

const fnDefParents = new Set([
    'function_declaration', 'method_declaration',
    'function_item', 'function_definition', 'method_definition',
    'generator_function_declaration', 'generator_function',
])


function resolveTag(nt: string, parentStack: string[]): Tag | null {
    const pLen = parentStack.length
    const parent = pLen > 0 ? parentStack[pLen - 1] : ''
    const grandparent = pLen > 1 ? parentStack[pLen - 2] : ''

    if (nt === 'identifier') {
        if (fnDefParents.has(parent)) {
            return tags.function(tags.variableName)
        }
        if (parent === 'call_expression') {
            return tags.function(tags.variableName)
        }
        return null
    }

    if (nt === 'field_identifier' || nt === 'property_identifier') {
        if (parent === 'selector_expression' && grandparent === 'call_expression') {
            return tags.function(tags.variableName)
        }
        // fall through to matchTag → tags.propertyName
    }

    return matchTag(nt)
}

// ── Tag → CSS style mapping (oneDark palette) ─────────────────────────

// ── Tag → CSS class name ──────────────────────────────────────────────
// Each unique style string gets a short CSS class; a <style> element is
// injected once at module load so CodeMirror class-based decorations work.

const tagStyleEntries: [Tag, string][] = [
    // Comments
    [tags.comment, 'color:#5c6370;font-style:italic'],
    [tags.docComment, 'color:#5c6370;font-style:italic'],

    // Strings
    [tags.string, 'color:#98c379'],
    [tags.escape, 'color:#98c379'],
    [tags.meta, 'color:#c678dd'],

    // Numbers & literals
    [tags.number, 'color:#d19a66'],
    [tags.literal, 'color:#d19a66'],
    [tags.bool, 'color:#d19a66'],
    [tags.color, 'color:#d19a66'],
    [tags.unit, 'color:#d19a66'],

    // Keywords
    [tags.keyword, 'color:#c678dd'],
    [tags.moduleKeyword, 'color:#c678dd'],
    [tags.controlKeyword, 'color:#c678dd'],
    [tags.definitionKeyword, 'color:#c678dd'],
    [tags.self, 'color:#c678dd'],

    // Types
    [tags.typeName, 'color:#56b6c2'],
    [tags.className, 'color:#56b6c2'],
    [tags.namespace, 'color:#56b6c2'],

    // Names
    [tags.variableName, 'color:#e06c75'],
    [tags.propertyName, 'color:#e06c75'],
    [tags.function(tags.variableName), 'color:#61afef'],
    [tags.function(tags.propertyName), 'color:#61afef'],
    [tags.labelName, 'color:#61afef'],

    // HTML/XML
    [tags.tagName, 'color:#e06c75'],
    [tags.attributeName, 'color:#98c379'],
    [tags.attributeValue, 'color:#98c379'],
    [tags.processingInstruction, 'color:#5c6370'],
    [tags.character, 'color:#98c379'],

    // Markdown
    [tags.heading, 'color:#e06c75;font-weight:700'],
    [tags.emphasis, 'font-style:italic'],
    [tags.strong, 'font-weight:700'],
    [tags.strikethrough, 'text-decoration:line-through'],
    [tags.link, 'color:#61afef;text-decoration:underline'],
    [tags.monospace, 'color:#98c379;font-family:monospace'],
    [tags.quote, 'color:#5c6370;font-style:italic'],
    [tags.list, 'color:#5c6370'],
    [tags.contentSeparator, 'color:#5c6370'],

    // Operators & punctuation
    [tags.operator, 'color:#abb2bf'],
    [tags.punctuation, 'color:#abb2bf'],
    [tags.separator, 'color:#abb2bf'],
    [tags.bracket, 'color:#abb2bf'],

    // Regex
    [tags.regexp, 'color:#98c379'],
    // Special
    [tags.special(tags.string), 'color:#98c379'],
    [tags.special(tags.variableName), 'color:#c678dd'],
]

const tagClassMap = new Map<Tag, string>()
const styleToClass = new Map<string, string>()

// Inject CSS rules once at module load
{
    const el = document.createElement('style')
    el.id = 'monika-ts-highlight'
    let css = ''
    let nextId = 0
    for (const [t, s] of tagStyleEntries) {
        let cls = styleToClass.get(s)
        if (!cls) {
            cls = `ts-${nextId++}`
            styleToClass.set(s, cls)
            css += `.${cls} { ${s} }\n`
        }
        tagClassMap.set(t, cls)
    }
    el.textContent = css
    document.head.appendChild(el)
}

function tagToClass(tag: Tag): string | null {
    return tagClassMap.get(tag) ?? null
}

// ── Build decorations from tree-sitter parse ──────────────────────────

export function buildDecorations(lang: Language, source: string): DecorationSet {
    const parser = new Parser()
    parser.setLanguage(lang)
    const tree = parser.parse(source)
    if (!tree) {
        console.warn('[ts-hl] parse returned null')
        parser.delete()
        return Decoration.set([])
    }

    const cursor = tree.walk()
    const ranges: { from: number; to: number; cls: string }[] = []
    const parentStack: string[] = []

    // Walk the tree (first child enters the root node's body)
    let visitChildren = cursor.gotoFirstChild()
    while (true) {
        if (visitChildren) {
            const nt = cursor.currentNode.type
            const start = cursor.startIndex
            const end = cursor.endIndex
            if (start < end && nt !== 'ERROR' && nt !== 'MISSING') {
                const tag = resolveTag(nt, parentStack)
                if (tag) {
                    const cls = tagToClass(tag)
                    if (cls) {
                        ranges.push({ from: start, to: end, cls })
                    }
                }
            }
        }

        const currentType = cursor.currentNode.type
        if (cursor.gotoFirstChild()) {
            parentStack.push(currentType)
            visitChildren = true
            continue
        }

        while (!cursor.gotoNextSibling()) {
            if (!cursor.gotoParent()) {
                // Reached the top — done
                cursor.delete()
                tree.delete()
                parser.delete()

                if (ranges.length > 0) {
                    console.log(
                        `[ts-hl] ${ranges.length} ranges, first 3:`,
                        ranges.slice(0, 3).map(r => `${source.slice(r.from, r.to).slice(0, 20)}[.${r.cls}]`)
                    )
                }

                if (ranges.length === 0) return Decoration.set([])

                // Sort and merge: prefer child (smaller) ranges over parent (larger) ranges
                ranges.sort((a, b) => a.from - b.from || a.to - b.to)
                const merged: typeof ranges = [ranges[0]]
                for (let i = 1; i < ranges.length; i++) {
                    const last = merged[merged.length - 1]
                    const cur = ranges[i]
                    if (cur.from === last.to && cur.cls === last.cls) {
                        last.to = cur.to
                    } else if (cur.from >= last.from && cur.to <= last.to) {
                        if (cur.cls !== last.cls) {
                            const before = last.from < cur.from
                                ? { from: last.from, to: cur.from, cls: last.cls }
                                : null
                            const after = cur.to < last.to
                                ? { from: cur.to, to: last.to, cls: last.cls }
                                : null
                            merged.pop()
                            if (before) merged.push(before)
                            merged.push(cur)
                            if (after) merged.push(after)
                        }
                    } else if (cur.from < last.to) {
                        last.to = cur.from
                        merged.push(cur)
                    } else {
                        merged.push(cur)
                    }
                }

                return Decoration.set(
                    merged
                        .filter(({ from, to }) => from < to)
                        .map(({ from, to, cls }) =>
                            Decoration.mark({ class: cls }).range(from, to)
                        )
                )
            }
            parentStack.pop()
        }
        visitChildren = true
    }
}


// ── CodeMirror ViewPlugin ──────────────────────────────────────────────

export function treeSitterHighlightExtension(lang: Language) {
    return ViewPlugin.fromClass(class {
        decorations: DecorationSet
        constructor(view: EditorView) {
            this.decorations = buildDecorations(lang, view.state.doc.toString())
        }
        update(update: ViewUpdate) {
            if (update.docChanged) {
                this.decorations = buildDecorations(lang, update.state.doc.toString())
            }
        }
    }, {
        decorations: v => v.decorations,
    })
}
