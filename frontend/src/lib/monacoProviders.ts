import type { languages, editor } from 'monaco-editor'
import { Call } from '@wailsio/runtime'
import { lspService, LspDiagnostic } from './lspService'
let monacoRef: typeof import('monaco-editor') | null = null

export function initMonacoProviders(monaco: typeof import('monaco-editor')) {
    monacoRef = monaco
    registerCompletionProvider(monaco)
    registerHoverProvider(monaco)
    registerDefinitionProvider(monaco)
}

function getModelParams(model: editor.ITextModel): { pp: string; fp: string } | null {
    const uri = model.uri.toString()
    const m = uri.match(/^monika:\/\/(.+?)\/(.+)$/)
    if (!m) return null
    return { pp: decodeURIComponent(m[1]), fp: decodeURIComponent(m[2]) }
}

function toMonacoSeverity(severity: number): import('monaco-editor').MarkerSeverity {
    switch (severity) {
        case 1: return monacoRef!.MarkerSeverity.Error
        case 2: return monacoRef!.MarkerSeverity.Warning
        case 3: return monacoRef!.MarkerSeverity.Info
        case 4: return monacoRef!.MarkerSeverity.Hint
        default: return monacoRef!.MarkerSeverity.Info
    }
}

export function updateDiagnostics(model: editor.ITextModel, diags: LspDiagnostic[]) {
    const markers: editor.IMarkerData[] = diags.map(d => ({
        severity: toMonacoSeverity(d.severity),
        message: d.message,
        startLineNumber: d.startLine + 1,
        startColumn: d.startCol + 1,
        endLineNumber: d.endLine + 1,
        endColumn: d.endCol + 1,
    }))
    monacoRef?.editor.setModelMarkers(model, 'lsp', markers)
}

const EXT_LANG_MAP: Record<string, string> = {
    ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
    mjs: 'javascript', py: 'python', go: 'go', rs: 'rust',
    json: 'json', css: 'css', scss: 'scss', less: 'less',
    html: 'html', htm: 'html', xml: 'xml', svg: 'xml',
    md: 'markdown', mdx: 'markdown',
    yaml: 'yaml', yml: 'yaml', toml: 'plaintext',
    sh: 'shell', bash: 'shell', zsh: 'shell',
    sql: 'sql', graphql: 'graphql', gql: 'graphql',
    java: 'java', c: 'c', cpp: 'cpp', h: 'c', hpp: 'cpp',
    kt: 'kotlin', swift: 'swift', dart: 'dart',
}

function extToMonacoLang(filePath: string): string {
    const ext = filePath.split('.').pop()?.toLowerCase() || ''
    return EXT_LANG_MAP[ext] || 'plaintext'
}

async function ensureModelExists(pp: string, fp: string) {
    if (!monacoRef) return
    const uri = monacoRef.Uri.parse('monika://' + encodeURIComponent(pp) + '/' + encodeURIComponent(fp))
    if (monacoRef.editor.getModel(uri)) return
    try {
        const result: any = await Call.ByName('monika/internal/api.App.ReadFile', pp, fp)
        monacoRef.editor.createModel(result.content || '', extToMonacoLang(fp), uri)
    } catch { /* ignore */ }
}

function mapCompletionKind(monaco: typeof import('monaco-editor'), kind?: number): languages.CompletionItemKind {
    const map: Record<number, languages.CompletionItemKind> = {
        1: monaco.languages.CompletionItemKind.Text,
        2: monaco.languages.CompletionItemKind.Method,
        3: monaco.languages.CompletionItemKind.Function,
        4: monaco.languages.CompletionItemKind.Constructor,
        5: monaco.languages.CompletionItemKind.Field,
        6: monaco.languages.CompletionItemKind.Variable,
        7: monaco.languages.CompletionItemKind.Class,
        8: monaco.languages.CompletionItemKind.Interface,
        9: monaco.languages.CompletionItemKind.Module,
        10: monaco.languages.CompletionItemKind.Property,
        12: monaco.languages.CompletionItemKind.Value,
        13: monaco.languages.CompletionItemKind.Enum,
        14: monaco.languages.CompletionItemKind.Keyword,
        15: monaco.languages.CompletionItemKind.Snippet,
        17: monaco.languages.CompletionItemKind.File,
        21: monaco.languages.CompletionItemKind.Constant,
        22: monaco.languages.CompletionItemKind.Struct,
        24: monaco.languages.CompletionItemKind.Operator,
        25: monaco.languages.CompletionItemKind.TypeParameter,
    }
    return map[kind ?? 0] || monaco.languages.CompletionItemKind.Property
}

function registerCompletionProvider(monaco: typeof import('monaco-editor')) {
    monaco.languages.registerCompletionItemProvider('*', {
        triggerCharacters: ['.'],
        provideCompletionItems: async (model, position) => {
            try {
                const params = getModelParams(model)
                if (!params) return null

                const pp = params.pp
                const fp = params.fp
                const line = position.lineNumber - 1
                const col = position.column - 1

                const content = model.getValue()
                await lspService.didChange(pp, fp, content, Date.now())

                const result = await lspService.completion(pp, fp, line, col)
                if (!result || !result.items || result.items.length === 0) return null

                return {
                    suggestions: result.items.map(item => ({
                        label: item.label,
                        kind: mapCompletionKind(monaco, item.kind),
                        detail: item.detail,
                        documentation: item.documentation || undefined,
                        insertText: item.insertText || item.label,
                        range: {
                            startLineNumber: position.lineNumber,
                            startColumn: position.column,
                            endLineNumber: position.lineNumber,
                            endColumn: position.column,
                        },
                    })),
                } satisfies languages.CompletionList
            } catch (e) {
                console.warn('[lsp] completion provider error:', e)
                return null
            }
        },
    })
}

function registerHoverProvider(monaco: typeof import('monaco-editor')) {
    monaco.languages.registerHoverProvider('*', {
        provideHover: async (model, position) => {
            try {
                const params = getModelParams(model)
                if (!params) return null

                const lineLen = model.getLineLength(position.lineNumber)
                const col = Math.min(position.column - 1, Math.max(0, lineLen - 1))

                const result = await lspService.hover(
                    params.pp, params.fp,
                    position.lineNumber - 1, col,
                )
                if (!result || !result.contents) return null

                return {
                    contents: [{ value: result.contents }],
                }
            } catch (e) {
                console.warn('[lsp] hover provider error:', e)
                return null
            }
        },
    })
}

function registerDefinitionProvider(monaco: typeof import('monaco-editor')) {
    monaco.languages.registerDefinitionProvider('*', {
        provideDefinition: async (model, position) => {
            try {
                const params = getModelParams(model)
                if (!params) return null

                const lineLen = model.getLineLength(position.lineNumber)
                const col = Math.min(position.column - 1, Math.max(0, lineLen - 1))

                const locs = await lspService.goToDefinition(
                    params.pp, params.fp,
                    position.lineNumber - 1, col,
                )
                if (!locs || locs.length === 0) return null

                // Ensure Monaco models exist for cross-file targets so Peek widget doesn't crash
                for (const loc of locs) {
                    if (loc.path !== params.fp) {
                        await ensureModelExists(params.pp, loc.path)
                    }
                }

                return locs.map(loc => ({
                    uri: monaco.Uri.parse('monika://' + encodeURIComponent(params.pp) + '/' + encodeURIComponent(loc.path)),
                    range: {
                        startLineNumber: loc.line + 1,
                        startColumn: loc.col + 1,
                        endLineNumber: loc.line + 1,
                        endColumn: loc.col + 1,
                    },
                }))
            } catch (e) {
                console.warn('[lsp] definition provider error:', e)
                return null
            }
        },
    })
}
