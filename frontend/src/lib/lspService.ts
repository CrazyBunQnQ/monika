import { Call } from '@wailsio/runtime'

export interface LspLocation {
    path: string
    line: number
    col: number
}

export interface LspHoverResult {
    contents: string
}

export interface LspSymbol {
    name: string
    kind: number
    path: string
    startLine: number
    startCol: number
    endLine: number
    endCol: number
    children: LspSymbol[]
}

export interface LspDiagnostic {
    startLine: number
    startCol: number
    endLine: number
    endCol: number
    severity: number
    message: string
    source: string
    code?: string
}

export interface LspTextEdit {
    startLine: number
    startCol: number
    endLine: number
    endCol: number
    newText: string
}

export interface LspFileEdit {
    path: string
    edits: LspTextEdit[]
}

export interface LspWorkspaceEdit {
    changes: LspFileEdit[]
}

export interface LspCodeAction {
    title: string
    kind: string
    edit?: LspWorkspaceEdit | null
}

const method = (name: string) => 'monika/internal/api.App.' + name

export const lspService = {
    openFile: (projectPath: string, filePath: string) =>
        Call.ByName(method('LspOpenFile'), projectPath, filePath),

    closeFile: (projectPath: string, filePath: string) =>
        Call.ByName(method('LspCloseFile'), projectPath, filePath),

    didChange: (projectPath: string, filePath: string, content: string, version: number) =>
        Call.ByName(method('LspDidChange'), projectPath, filePath, content, version),

    goToDefinition: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspGoToDefinition'), projectPath, filePath, line, col) as Promise<LspLocation[]>,

    typeDefinition: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspTypeDefinition'), projectPath, filePath, line, col) as Promise<LspLocation[]>,

    implementation: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspImplementation'), projectPath, filePath, line, col) as Promise<LspLocation[]>,

    references: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspReferences'), projectPath, filePath, line, col) as Promise<LspLocation[]>,

    hover: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspHover'), projectPath, filePath, line, col) as Promise<LspHoverResult | null>,

    completion: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspCompletion'), projectPath, filePath, line, col) as Promise<{
            isIncomplete: boolean;
            items: { label: string; kind?: number; detail?: string; documentation?: string; insertText?: string }[];
        } | null>,

    documentSymbols: (projectPath: string, filePath: string) =>
        Call.ByName(method('LspDocumentSymbols'), projectPath, filePath) as Promise<LspSymbol[]>,

    diagnostics: (projectPath: string, filePath: string) =>
        Call.ByName(method('LspDiagnostics'), projectPath, filePath) as Promise<LspDiagnostic[]>,

    rename: (projectPath: string, filePath: string, line: number, col: number, newName: string) =>
        Call.ByName(method('LspRename'), projectPath, filePath, line, col, newName) as Promise<LspWorkspaceEdit | null>,

    codeActions: (projectPath: string, filePath: string, line: number, col: number) =>
        Call.ByName(method('LspCodeActions'), projectPath, filePath, line, col) as Promise<LspCodeAction[]>,

    executeCodeAction: (projectPath: string, action: LspCodeAction) =>
        Call.ByName(method('LspExecuteCodeAction'), projectPath, action) as Promise<LspWorkspaceEdit | null>,
}
