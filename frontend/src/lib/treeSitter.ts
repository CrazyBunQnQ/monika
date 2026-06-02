import { Parser, type Language, type Query, type QueryMatch, type Tree } from 'web-tree-sitter'
import { Events, Call } from '@wailsio/runtime'

type GrammarLoader = () => Promise<Language>

interface GrammarEntry {
  name: string
  extensions: string[]
  load: GrammarLoader
}

const grammars: GrammarEntry[] = []
let initialized = false
const langCache = new Map<string, Language>()

function extToLang(ext: string): string | undefined {
  const e = ext.toLowerCase()
  for (const g of grammars) {
    if (g.extensions.includes(e)) return g.name
  }
  return undefined
}

function langFromPath(path: string): string | undefined {
  const dot = path.lastIndexOf('.')
  if (dot < 0) return undefined
  return extToLang(path.slice(dot))
}

async function getLanguage(langName: string): Promise<Language | null> {
  const cached = langCache.get(langName)
  if (cached) return cached

  const entry = grammars.find(g => g.name === langName)
  if (!entry) return null

  const lang = await entry.load()
  langCache.set(langName, lang)
  return lang
}

async function doQuery(langName: string, source: string, pattern: string): Promise<QueryMatch[]> {
  const lang = await getLanguage(langName)
  if (!lang) return []

  const parser = new Parser()
  parser.setLanguage(lang)
  const tree = parser.parse(source)
  if (!tree) return []

  try {
    const query = new Query(lang, pattern)
    try {
      const matches = query.matches(tree.rootNode)
      return matches.map(m => ({
        patternIndex: m.patternIndex,
        captures: m.captures.map(c => ({
          patternIndex: c.patternIndex,
          name: c.name,
          node: {
            type: c.node.type,
            text: c.node.text,
            startIndex: c.node.startIndex,
            endIndex: c.node.endIndex,
            startPosition: { row: c.node.startPosition.row, column: c.node.startPosition.column },
            endPosition: { row: c.node.endPosition.row, column: c.node.endPosition.column },
            childCount: c.node.childCount,
            isNamed: c.node.isNamed,
          },
        })),
      }))
    } finally {
      query.delete()
    }
  } finally {
    tree.delete()
    parser.delete()
  }
}

interface SummaryNode {
  type: string
  text: string
  startRow: number
  endRow: number
  children?: SummaryNode[]
  folded?: boolean
}

const foldableTypes = new Set([
  // Go
  'function_declaration', 'method_declaration', 'type_declaration', 'interface_type',
  'struct_type', 'func_literal', 'block',
  // JS/TS
  'function_declaration', 'function_expression', 'arrow_function', 'method_definition',
  'class_declaration', 'class_body', 'interface_declaration', 'type_declaration',
  'object', 'switch_statement', 'try_statement',
  // Python
  'function_definition', 'class_definition', 'decorated_definition',
  'if_statement', 'for_statement', 'while_statement', 'try_statement', 'with_statement',
  // Rust
  'function_item', 'impl_item', 'struct_item', 'enum_item', 'trait_item',
  'match_expression', 'closure_expression',
  // Java
  'class_declaration', 'method_declaration', 'interface_declaration',
  'constructor_declaration', 'enum_declaration',
  // C/C++
  'function_definition', 'class_specifier', 'struct_specifier', 'namespace_definition',
])

function buildSummary(node: any, source: string, totalLines: number, minBodyLines = 4): SummaryNode {
  const startRow = node.startPosition.row
  const endRow = node.endPosition.row
  const bodyLines = endRow - startRow + 1

  const result: SummaryNode = {
    type: node.type,
    text: node.text.split('\n')[0],
    startRow,
    endRow,
  }

  if (bodyLines > minBodyLines && foldableTypes.has(node.type)) {
    result.folded = true
  }

  if (!result.folded && node.childCount > 0) {
    const children: SummaryNode[] = []
    for (let i = 0; i < node.childCount; i++) {
      const child = node.child(i)
      if (child.isNamed) {
        children.push(buildSummary(child, source, totalLines, minBodyLines))
      }
    }
    if (children.length > 0) result.children = children
  }

  return result
}

async function doSummarize(langName: string, source: string): Promise<SummaryNode | null> {
  const lang = await getLanguage(langName)
  if (!lang) return null

  const parser = new Parser()
  parser.setLanguage(lang)
  const tree = parser.parse(source)
  if (!tree) return null

  try {
    const totalLines = source.split('\n').length
    return buildSummary(tree.rootNode, source, totalLines)
  } finally {
    tree.delete()
    parser.delete()
  }
}

function formatSummary(node: SummaryNode, indent = 0): string {
  const prefix = '  '.repeat(indent)
  const lines: string[] = []

  if (node.folded) {
    lines.push(`${prefix}${node.type} [${node.startRow + 1}-${node.endRow + 1}] ${truncate(node.text, 60)}`)
    lines.push(`${prefix}  ..`)
  } else {
    lines.push(`${prefix}${node.type} [${node.startRow + 1}-${node.endRow + 1}] ${truncate(node.text, 60)}`)
    if (node.children) {
      for (const child of node.children) {
        lines.push(...formatSummary(child, indent + 1).split('\n'))
      }
    }
  }

  return lines.join('\n')
}

function truncate(s: string, max: number): string {
  const oneLine = s.replace(/\n/g, '\\n').trim()
  if (oneLine.length <= max) return oneLine
  return oneLine.slice(0, max - 3) + '...'
}

async function doSupportedLanguages(): Promise<string[]> {
  return grammars.map(g => g.name)
}

// IPC handler — listens for requests from Go backend
function setupIPCHandler() {
  Events.On('ts:request', async (ev: any) => {
    const { id, method, params } = ev.data as { id: string; method: string; params: any }
    try {
      let result: any
      switch (method) {
        case 'query':
          result = await doQuery(params.lang, params.source, params.pattern)
          break
        case 'summarize':
          result = await doSummarize(params.lang, params.source)
          break
        case 'supportedLanguages':
          result = await doSupportedLanguages()
          break
        default:
          throw new Error(`unknown method: ${method}`)
      }
      await Call.ByName('monika/internal/api.App.TSResponse', { id, ok: true, data: result })
    } catch (err: any) {
      await Call.ByName('monika/internal/api.App.TSResponse', { id, ok: false, error: String(err?.message || err) })
    }
  })
}

function wasmUrl(lang: string): string {
  return `/grammars/tree-sitter-${lang}.wasm`
}

function registerBuiltinGrammars() {
  const defs: [string, string[]][] = [
    ['go', ['.go']],
    ['javascript', ['.js', '.jsx', '.mjs', '.cjs']],
    ['typescript', ['.ts', '.tsx']],
    ['python', ['.py', '.pyw']],
    ['rust', ['.rs']],
    ['java', ['.java']],
    ['c', ['.c', '.h']],
    ['cpp', ['.cpp', '.cc', '.cxx', '.hpp', '.hh', '.hxx']],
    ['c_sharp', ['.cs']],
    ['ruby', ['.rb']],
    ['php', ['.php']],
    ['swift', ['.swift']],
    ['kotlin', ['.kt', '.kts']],
    ['scala', ['.scala']],
    ['json', ['.json']],
    ['yaml', ['.yaml', '.yml']],
    ['toml', ['.toml']],
    ['html', ['.html', '.htm']],
    ['css', ['.css']],
    ['bash', ['.sh', '.bash', '.zsh']],
  ]

  for (const [name, exts] of defs) {
    registerGrammar(name, exts, async () => {
      const lang = await Parser.Language.load(wasmUrl(name))
      return lang
    })
  }
}

// Grammar registry API — call from setup to add new languages
export function registerGrammar(name: string, extensions: string[], loader: GrammarLoader) {
  grammars.push({ name, extensions, loader })
}

// Get language name for a file path
export function getLanguageForFile(path: string): string | undefined {
  return langFromPath(path)
}

// Query API
export async function query(lang: string, source: string, pattern: string) {
  return doQuery(lang, source, pattern)
}

// Summarize API
export async function summarize(lang: string, source: string) {
  return doSummarize(lang, source)
}

// Initialize the tree-sitter service
export async function initTreeSitter() {
  if (initialized) return
  await Parser.init()
  registerBuiltinGrammars()
  setupIPCHandler()
  initialized = true
  console.log('[tree-sitter] initialized, grammars:', grammars.map(g => g.name).join(', '))
}
