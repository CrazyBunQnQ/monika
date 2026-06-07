# LSP File Preview — 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为文件预览面板添加完整 LSP 能力（跳转定义、引用、hover、符号、诊断、rename、code actions）

**Architecture:** 复用现有 `internal/lsp/Manager`，在 `App` struct 上新增 Wails IPC 方法暴露给前端。前端 CodeMirror 6 从只读升级为可编辑，通过新增的 IPC 方法调用 LSP 能力。用户编辑直接走 `WriteFile` + `LspDidChange`，不与 AI agent 的 tool pipeline 耦合。

**Tech Stack:** Go (existing LSP client), CodeMirror 6 (`@codemirror/view`, `@codemirror/tooltip`, `@codemirror/lint`), Wails IPC

---

## Phase 1: Backend LSP IPC 层

### Task 1: 添加 LSP API 类型到 types.go

**Files:**
- Modify: `internal/api/types.go` (追加类型定义)

**Step 1: 追加 LSP 结果类型**

在 `internal/api/types.go` 文件末尾追加以下类型。这些类型从 `lsp/protocol.go` 的 LSP 原生类型转换为前端友好的格式（行/列改为数字，URI 转为文件路径）：

```go
// LSP API types — flattened for Wails IPC, no raw LSP URIs

type LspLocation struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}

type LspHoverResult struct {
	Contents string `json:"contents"`
}

type LspSymbol struct {
	Name     string       `json:"name"`
	Kind     int          `json:"kind"`
	Path     string       `json:"path"`
	StartLine int         `json:"startLine"`
	StartCol  int         `json:"startCol"`
	EndLine   int         `json:"endLine"`
	EndCol    int         `json:"endCol"`
	Children  []LspSymbol `json:"children,omitempty"`
}

type LspDiagnostic struct {
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startCol"`
	EndLine   int    `json:"endLine"`
	EndCol    int    `json:"endCol"`
	Severity  int    `json:"severity"`
	Message   string `json:"message"`
	Source    string `json:"source"`
	Code      string `json:"code,omitempty"`
}

type LspTextEdit struct {
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startCol"`
	EndLine   int    `json:"endLine"`
	EndCol    int    `json:"endCol"`
	NewText   string `json:"newText"`
}

type LspFileEdit struct {
	Path  string        `json:"path"`
	Edits []LspTextEdit `json:"edits"`
}

type LspWorkspaceEdit struct {
	Changes []LspFileEdit `json:"changes"`
}

type LspCodeAction struct {
	Title string           `json:"title"`
	Kind  string           `json:"kind"`
	Edit  *LspWorkspaceEdit `json:"edit,omitempty"`
}
```

**Step 2: 运行 `wails3 generate bindings -ts` 生成 TypeScript 绑定**

```powershell
wails3 generate bindings -f "..." -ts
```

（完整命令取决于实际 flags，参考 `AGENTS.md` 中的说明）

**Step 3: 验证生成的绑定**

检查 `frontend/bindings/monika/` 目录下生成了新的类型定义。

---

### Task 2: 添加 getLspManager 辅助方法

**Files:**
- Modify: `internal/api/app.go`

**Step 1: 在 App struct 上添加私有辅助方法**

在 `GetLSPStatus` 方法附近（约第 3142 行之后），添加：

```go
func (a *App) getLspManager() *lsp.Manager {
	t, ok := a.registry.Get("lsp")
	if !ok {
		return nil
	}
	type lspTool interface {
		Manager() *lsp.Manager
	}
	if lt, ok := t.(lspTool); ok {
		return lt.Manager()
	}
	return nil
}
```

**Step 2: 重构 GetLSPStatus 使用新辅助方法**

将 `GetLSPStatus` 改为调用 `getLspManager`：

```go
func (a *App) GetLSPStatus() []lsp.LSPServerStatus {
	mgr := a.getLspManager()
	if mgr == nil {
		return nil
	}
	return mgr.ServerStatuses()
}
```

**Step 3: 编译验证**

```powershell
go build ./...
```

---

### Task 3: 添加文件生命周期方法

**Files:**
- Modify: `internal/api/app.go`

**Step 1: 添加 LspOpenFile 方法**

```go
func (a *App) LspOpenFile(projectPath, filePath string) error {
	mgr := a.getLspManager()
	if mgr == nil {
		return nil
	}
	absPath := filepath.Join(projectPath, filePath)
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(projectPath, filePath)
	}
	client, serverName, err := mgr.ClientForFile(a.ctx, absPath)
	if err != nil {
		return err
	}
	_, _, err = mgr.EnsureFileOpen(a.ctx, client, absPath, serverName)
	return err
}
```

**Step 2: 添加 LspCloseFile 方法**

```go
func (a *App) LspCloseFile(projectPath, filePath string) error {
	mgr := a.getLspManager()
	if mgr == nil {
		return nil
	}
	absPath := filepath.Join(projectPath, filePath)
	client, _, err := mgr.ClientForFile(a.ctx, absPath)
	if err != nil {
		return nil
	}
	return mgr.CloseFile(a.ctx, client, absPath)
}
```

**Step 3: 添加 LspDidChange 方法**

```go
func (a *App) LspDidChange(projectPath, filePath, content string, version int) error {
	mgr := a.getLspManager()
	if mgr == nil {
		return nil
	}
	absPath := filepath.Join(projectPath, filePath)
	client, serverName, err := mgr.ClientForFile(a.ctx, absPath)
	if err != nil {
		return err
	}
	return mgr.SyncContentFromMemory(a.ctx, client, absPath, content, serverName)
}
```

**Step 4: 编译验证**

```powershell
go build ./...
```

**注意：** `filepath.Join` 处理 `absPath` 的逻辑需要修正——如果 `filePath` 已是绝对路径则不拼接。看 `lsp/tool.go:170-173` 的模式来处理。

---

### Task 4: 添加读操作方法（Definition, References, Hover, Symbols）

**Files:**
- Modify: `internal/api/app.go`

**Step 1: 添加 resolveLspClient 辅助方法（减少重复代码）**

```go
func (a *App) resolveLspClient(projectPath, filePath string, pos *lsp.Position) (*lsp.Client, string, string, error) {
	mgr := a.getLspManager()
	if mgr == nil {
		return nil, "", "", fmt.Errorf("no LSP manager available")
	}
	absPath := filePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(projectPath, filePath)
	}
	client, serverName, err := mgr.ClientForFile(a.ctx, absPath)
	if err != nil {
		return nil, "", "", err
	}
	_, err = mgr.EnsureAndSync(a.ctx, client, absPath, serverName)
	if err != nil {
		return nil, "", "", err
	}
	return client, serverName, fileToURICall(a.ctx, absPath), nil
}

// fileToURICall is a thin wrapper — use the existing fileToURI from lsp package
// But since it's unexported, define a local helper
func filePathToURI(path string) string {
	return "file:///" + filepath.ToSlash(path)
}
```

**Step 2: 添加 LspGoToDefinition**

```go
func (a *App) LspGoToDefinition(projectPath, filePath string, line, col int) ([]LspLocation, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	locs, err := client.Definition(a.ctx, uri, lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	return locationsToLSPLocations(locs), nil
}
```

**Step 3: 添加 LspReferences**

```go
func (a *App) LspReferences(projectPath, filePath string, line, col int) ([]LspLocation, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	locs, err := client.References(a.ctx, uri, lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	return locationsToLSPLocations(locs), nil
}
```

**Step 4: 添加 LspHover**

```go
func (a *App) LspHover(projectPath, filePath string, line, col int) (*LspHoverResult, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	hover, err := client.Hover(a.ctx, uri, lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	if hover == nil {
		return nil, nil
	}
	text := hover.ContentText()
	if text == "" {
		return nil, nil
	}
	return &LspHoverResult{Contents: text}, nil
}
```

**Step 5: 添加 LspDocumentSymbols**

```go
func (a *App) LspDocumentSymbols(projectPath, filePath string) ([]LspSymbol, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, nil)
	if err != nil {
		return nil, err
	}
	syms, err := client.DocumentSymbols(a.ctx, uri)
	if err != nil {
		return nil, err
	}
	return documentSymbolsToLSP(syms, filePath), nil
}
```

**Step 6: 添加 LspTypeDefinition 和 LspImplementation（同模式）**

```go
func (a *App) LspTypeDefinition(projectPath, filePath string, line, col int) ([]LspLocation, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	locs, err := client.TypeDefinition(a.ctx, uri, lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	return locationsToLSPLocations(locs), nil
}

func (a *App) LspImplementation(projectPath, filePath string, line, col int) ([]LspLocation, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	locs, err := client.Implementation(a.ctx, uri, lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	return locationsToLSPLocations(locs), nil
}
```

**Step 7: 添加转换辅助函数**

```go
func locationsToLSPLocations(locs []lsp.Location) []LspLocation {
	result := make([]LspLocation, len(locs))
	for i, loc := range locs {
		result[i] = LspLocation{
			Path:  uriToFilePath(loc.URI),
			Line:  loc.Range.Start.Line,
			Col:   loc.Range.Start.Character,
		}
	}
	return result
}

func uriToFilePath(uri string) string {
	prefix := "file:///"
	if strings.HasPrefix(uri, prefix) {
		return uri[len(prefix):]
	}
	return uri
}

func documentSymbolsToLSP(syms []lsp.DocumentSymbol, filePath string) []LspSymbol {
	result := make([]LspSymbol, len(syms))
	for i, s := range syms {
		result[i] = LspSymbol{
			Name:      s.Name,
			Kind:      int(s.Kind),
			Path:      filePath,
			StartLine: s.Range.Start.Line,
			StartCol:  s.Range.Start.Character,
			EndLine:   s.Range.End.Line,
			EndCol:    s.Range.End.Character,
			Children:  documentSymbolsToLSP(s.Children, filePath),
		}
	}
	return result
}
```

**Step 8: 编译验证**

```powershell
go build ./...
```

---

### Task 5: 添加写操作方法（Rename, CodeActions, ExecuteCodeAction）

**Files:**
- Modify: `internal/api/app.go`

**Step 1: 添加 LspRename**

```go
func (a *App) LspRename(projectPath, filePath string, line, col int, newName string) (*LspWorkspaceEdit, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	wsEdit, err := client.Rename(a.ctx, uri, lsp.Position{Line: line, Character: col}, newName)
	if err != nil {
		return nil, err
	}
	return workspaceEditToLSP(wsEdit, projectPath), nil
}
```

**Step 2: 添加 LspCodeActions**

```go
func (a *App) LspCodeActions(projectPath, filePath string, line, col int) ([]LspCodeAction, error) {
	client, _, uri, err := a.resolveLspClient(projectPath, filePath, &lsp.Position{Line: line, Character: col})
	if err != nil {
		return nil, err
	}
	r := lsp.Range{
		Start: lsp.Position{Line: line, Character: col},
		End:   lsp.Position{Line: line, Character: col},
	}
	actions, err := client.CodeActions(a.ctx, uri, r, nil)
	if err != nil {
		return nil, err
	}
	result := make([]LspCodeAction, len(actions))
	for i, act := range actions {
		result[i] = LspCodeAction{
			Title: act.Title,
			Kind:  string(act.Kind),
			Edit:  workspaceEditToLSP(act.Edit, projectPath),
		}
	}
	return result, nil
}
```

**Step 3: 添加 LspExecuteCodeAction**

```go
func (a *App) LspExecuteCodeAction(projectPath string, action LspCodeAction) (*LspWorkspaceEdit, error) {
	// Code action 可以通过其 title/kind 查找对应 LSP server 并执行
	// 简化实现：如果 action.Edit 非空则直接返回
	if action.Edit != nil {
		return action.Edit, nil
	}
	return nil, fmt.Errorf("code action has no workspace edit to apply")
}
```

**Step 4: 添加 WorkspaceEdit 转换函数**

```go
func workspaceEditToLSP(edit *lsp.WorkspaceEdit, projectPath string) *LspWorkspaceEdit {
	if edit == nil {
		return nil
	}
	var changes []LspFileEdit
	for uri, edits := range edit.Changes {
		path := uriToFilePath(uri)
		lspEdits := make([]LspTextEdit, len(edits))
		for j, e := range edits {
			lspEdits[j] = LspTextEdit{
				StartLine: e.Range.Start.Line,
				StartCol:  e.Range.Start.Character,
				EndLine:   e.Range.End.Line,
				EndCol:    e.Range.End.Character,
				NewText:   e.NewText,
			}
		}
		changes = append(changes, LspFileEdit{Path: path, Edits: lspEdits})
	}
	// Also handle DocumentChanges for v2 workspace edit
	for _, dc := range edit.DocumentChanges {
		if dc.TextDocumentEdit != nil {
			path := uriToFilePath(dc.TextDocumentEdit.TextDocument.URI)
			lspEdits := make([]LspTextEdit, len(dc.TextDocumentEdit.Edits))
			for j, e := range dc.TextDocumentEdit.Edits {
				lspEdits[j] = LspTextEdit{
					StartLine: e.Range.Start.Line,
					StartCol:  e.Range.Start.Character,
					EndLine:   e.Range.End.Line,
					EndCol:    e.Range.End.Character,
					NewText:   e.NewText,
				}
			}
			changes = append(changes, LspFileEdit{Path: path, Edits: lspEdits})
		}
	}
	return &LspWorkspaceEdit{Changes: changes}
}
```

**Step 5: 编译验证**

```powershell
go build ./...
```

---

### Task 6: 添加 LspDiagnostics 方法

**Files:**
- Modify: `internal/api/app.go`

**Step 1: 添加 LspDiagnostics**

```go
func (a *App) LspDiagnostics(projectPath, filePath string) ([]LspDiagnostic, error) {
	mgr := a.getLspManager()
	if mgr == nil {
		return nil, nil
	}
	absPath := filePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(projectPath, filePath)
	}
	client, _, err := mgr.ClientForFile(a.ctx, absPath)
	if err != nil {
		return nil, err
	}
	uri := filePathToURI(absPath)
	diags := client.Diagnostics(uri)
	return diagnosticsToLSP(diags), nil
}

func diagnosticsToLSP(diags []lsp.Diagnostic) []LspDiagnostic {
	result := make([]LspDiagnostic, len(diags))
	for i, d := range diags {
		code := ""
		switch v := d.Code.(type) {
		case string:
			code = v
		}
		result[i] = LspDiagnostic{
			StartLine: d.Range.Start.Line,
			StartCol:  d.Range.Start.Character,
			EndLine:   d.Range.End.Line,
			EndCol:    d.Range.End.Character,
			Severity:  int(d.Severity),
			Message:   d.Message,
			Source:    d.Source,
			Code:      code,
		}
	}
	return result
}
```

**Step 2: 编译验证**

```powershell
go build ./...
```

---

### Task 7: 添加 LSP 诊断推送事件

**Files:**
- Modify: `internal/api/app.go`

**需求**: 后端 LSP server 通过 `textDocument/publishDiagnostics` 推送诊断时，通过 Wails Event 实时推送到前端。

**当前约束**: `Client.handleServerNotification` 在 `internal/lsp/client.go` 中处理推送，但 `Client` 没有回调/事件机制暴露给外部。需要通过 `Manager` 注册回调。

**替代方案（更简单）**: 不在 client 层做事件回调，而是由前端在编辑后 debounce 调用 `LspDiagnostics`。LSP server 的 push 会自然更新 client 缓存，下一次 pull 就能拿到最新结果。

**决定：** 不做 push 事件，改用前端主动 pull（编辑 debounce 500ms + `LspDiagnostics`）。这避免了修改 client 内部结构，且 LSP 诊断延迟 500ms 对预览场景完全可接受。

**无需代码变更。**

---

### Task 8: 生成 TypeScript 绑定

**Files:**
- Generated: `frontend/bindings/monika/types.ts` (auto-generated)

**Step 1: 运行绑定生成命令**

```powershell
wails3 generate bindings -f "..." -ts
```

**Step 2: 验证生成的类型**

检查 `frontend/bindings/monika/` 中是否包含了 `LspLocation`, `LspHoverResult`, `LspSymbol`, `LspDiagnostic`, `LspWorkspaceEdit`, `LspCodeAction` 等类型。

**Step 3: 如果生成不成功，手动在前端 store 中添加类型定义**

如果不能自动生成（`wails3 generate bindings` 可能需要特定的版本/参数），在 `frontend/src/store/index.ts` 或 `frontend/src/lib/lspService.ts` 中手动定义接口。

---

## Phase 2: 前端 LSP 基础设施

### Task 9: 安装 CodeMirror 相关 npm 包

**Files:**
- Modify: `frontend/package.json`

**Step 1: 安装依赖**

```powershell
cd frontend
npm install @codemirror/tooltip @codemirror/lint
```

**验证安装：**

```powershell
node -e "require('@codemirror/tooltip'); require('@codemirror/lint'); console.log('OK')"
```

---

### Task 10: 创建 LSP 服务模块

**Files:**
- Create: `frontend/src/lib/lspService.ts`

**Step 1: 创建文件**

```typescript
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
    severity: number  // 1=Error, 2=Warning, 3=Info, 4=Hint
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

const callLsp = async <T>(method: string, params: Record<string, any>): Promise<T> => {
    return Call.ByName('monika/internal/api.App.' + method, params)
}

const noop = ''

export const lspService = {
    openFile: (projectPath: string, filePath: string) =>
        callLsp<void>('LspOpenFile', { projectPath, filePath }),

    closeFile: (projectPath: string, filePath: string) =>
        callLsp<void>('LspCloseFile', { projectPath, filePath }),

    didChange: (projectPath: string, filePath: string, content: string, version: number) =>
        callLsp<void>('LspDidChange', { projectPath, filePath, content, version }),

    goToDefinition: (projectPath: string, filePath: string, line: number, col: number) =>
        callLsp<LspLocation[]>('LspGoToDefinition', { projectPath, filePath, line, col }),

    typeDefinition: (projectPath: string, filePath: string, line: number, col: number) =>
        callLsp<LspLocation[]>('LspTypeDefinition', { projectPath, filePath, line, col }),

    implementation: (projectPath: string, filePath: string, line: number, col: number) =>
        callLsp<LspLocation[]>('LspImplementation', { projectPath, filePath, line, col }),

    references: (projectPath: string, filePath: string, line: number, col: number) =>
        callLsp<LspLocation[]>('LspReferences', { projectPath, filePath, line, col }),

    hover: (projectPath: string, filePath: string, line: number, col: number) =>
        callLsp<LspHoverResult | null>('LspHover', { projectPath, filePath, line, col }),

    documentSymbols: (projectPath: string, filePath: string) =>
        callLsp<LspSymbol[]>('LspDocumentSymbols', { projectPath, filePath }),

    diagnostics: (projectPath: string, filePath: string) =>
        callLsp<LspDiagnostic[]>('LspDiagnostics', { projectPath, filePath }),

    rename: (projectPath: string, filePath: string, line: number, col: number, newName: string) =>
        callLsp<LspWorkspaceEdit | null>('LspRename', { projectPath, filePath, line, col, newName }),

    codeActions: (projectPath: string, filePath: string, line: number, col: number) =>
        callLsp<LspCodeAction[]>('LspCodeActions', { projectPath, filePath, line, col }),

    executeCodeAction: (projectPath: string, action: LspCodeAction) =>
        callLsp<LspWorkspaceEdit | null>('LspExecuteCodeAction', { projectPath, action }),
}
```

---

### Task 11: 添加 LSP 状态到 store

**Files:**
- Modify: `frontend/src/store/index.ts`

**Step 1: 添加 LSP 相关状态类型**

在 import 区域后添加：

```typescript
import { lspService, LspDiagnostic, LspSymbol } from '../lib/lspService'
```

**Step 2: 在 AppState 接口中添加字段**

```typescript
// 文件级别 LSP 状态
lspReady: Record<string, boolean>  // filePath -> LSP server ready
lspDiagnostics: Record<string, LspDiagnostic[]>  // filePath -> diagnostics
lspSymbols: Record<string, LspSymbol[]>  // filePath -> symbols
```

**Step 3: 添加初始化值**

```typescript
lspReady: {},
lspDiagnostics: {},
lspSymbols: {},
```

**Step 4: 添加 actions**

```typescript
// 文件生命周期
openLspFile: (projectPath: string, filePath: string) => {
    lspService.openFile(projectPath, filePath).catch(() => {
        get().lspReady[filePath] = false
    })
    get().lspReady[filePath] = true
},

closeLspFile: (projectPath: string, filePath: string) => {
    lspService.closeFile(projectPath, filePath).catch(() => {})
    delete get().lspReady[filePath]
},

// 设置诊断（从 push 事件或 pull）
setLspDiagnostics: (filePath: string, diags: LspDiagnostic[]) => {
    set({ lspDiagnostics: { ...get().lspDiagnostics, [filePath]: diags } })
},

// 设置符号
setLspSymbols: (filePath: string, syms: LspSymbol[]) => {
    set({ lspSymbols: { ...get().lspSymbols, [filePath]: syms } })
},
```

---

## Phase 3: 前端可编辑编辑器升级

### Task 12: 预览面板工具栏添加编辑模式切换

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加编辑模式状态**

在 `PreviewPanel` 组件内添加：

```typescript
const [editMode, setEditMode] = useState(false)
const [lspEnabled, setLspEnabled] = useState(false)
const versionRef = useRef(0)
```

**Step 2: 修改 CodeMirror 创建逻辑，支持可编辑**

将 `EditorView.editable.of(false)` 替换为 `EditorView.editable.of(editMode)`。

将 `'.cm-cursor': { display: 'none' }` 改为根据 editMode 动态控制：编辑模式下显示光标，只读模式隐藏。

**Step 3: 在 FilePreviewHeader 组件中添加编辑切换按钮**

在语言标签旁边添加一个编辑/只读切换图标按钮。点击时切换 `editMode` 状态，并重建 editor（因为 `editable` 是静态扩展，需要重建 EditorView）。

---

### Task 13: 编辑 → 保存同步流水线

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加 onChange 监听**

在构建 extensions 时添加：

```typescript
import { EditorView } from '@codemirror/view'
import { App } from '../../bindings/monika'

// 在 extensions 数组中添加：
EditorView.updateListener.of((update) => {
    if (!update.docChanged || !editMode) return
    const content = update.state.doc.toString()
    const store = useStore.getState()
    const fp = store.preview.filePath
    const pp = store.projectPath
    if (!fp || !pp) return

    // debounce 300ms 保存
    clearTimeout(saveTimeout)
    saveTimeout = setTimeout(async () => {
        versionRef.current++
        await App.WriteFile(pp, fp, content)
        if (lspEnabled) {
            lspService.didChange(pp, fp, content, versionRef.current)
                .catch(() => {})
        }
        // 触发文件树刷新
        store.refreshFileTree?.()
    }, 300)
})
```

**Step 2: 添加文件打开/关闭生命周期**

在 useEffect 中（创建 editor 的地方），当文件路径变化时：

```typescript
// 关闭旧文件的 LSP
if (prevFilePathRef.current && lspEnabled) {
    const store = useStore.getState()
    lspService.closeFile(store.projectPath, prevFilePathRef.current).catch(() => {})
}
// 打开新文件的 LSP
if (preview.filePath && !truncated && !showBinary) {
    const store = useStore.getState()
    lspService.openFile(store.projectPath, preview.filePath).catch(() => {})
    prevFilePathRef.current = preview.filePath
}
```

---

## Phase 4: LSP 功能实现

### Task 14: Ctrl+Click 跳转定义

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加 Ctrl+Click 事件处理**

在 extensions 数组中添加：

```typescript
import { keymap } from '@codemirror/view'

EditorView.domEventHandlers({
    mousedown: (event, view) => {
        if (!event.ctrlKey && !event.metaKey) return
        if (!lspEnabled) return
        event.preventDefault()

        const pos = view.posAtCoords({ x: event.clientX, y: event.clientY })
        if (pos === null) return
        const line = view.state.doc.lineAt(pos)
        const lineNum = line.number - 1  // 0-based
        const col = pos - line.from

        const store = useStore.getState()
        const fp = store.preview.filePath
        const pp = store.projectPath
        if (!fp || !pp) return

        lspService.goToDefinition(pp, fp, lineNum, col).then((locs) => {
            if (!locs || locs.length === 0) return
            if (locs.length === 1) {
                const loc = locs[0]
                // 同文件：滚动到目标行
                // 跨文件：打开新预览
                // ... 实现见下方
            }
        }).catch(() => {})
    },
})
```

**Step 2: 实现跳转逻辑**

```typescript
function navigateToLocation(loc: LspLocation) {
    const store = useStore.getState()
    const curPath = store.preview.filePath
    if (loc.path === curPath) {
        // 同文件跳转：滚动到目标行
        editorRef.current?.dispatch({
            effects: EditorView.scrollIntoView(
                editorRef.current.state.doc.line(loc.line + 1).from,
                { y: 'center' }
            )
        })
    } else {
        // 跨文件跳转：读取目标文件并设置预览
        App.ReadFile(store.projectPath, loc.path).then((f) => {
            store.setPreviewFile(loc.path, loc.path.split('/').pop() || '', f.content)
        })
    }
    // 记录导航历史
    pushNavHistory({ path: loc.path, line: loc.line, col: loc.col })
}
```

---

### Task 15: Hover Tooltip

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加 hover tooltip 扩展**

```typescript
import { hoverTooltip } from '@codemirror/toolkit'

hoverTooltip(async (view, pos) => {
    if (!lspEnabled) return null
    const store = useStore.getState()
    const fp = store.preview.filePath
    const pp = store.projectPath
    if (!fp || !pp) return null

    const line = view.state.doc.lineAt(pos)
    const lineNum = line.number - 1
    const col = pos - line.from

    const result = await lspService.hover(pp, fp, lineNum, col)
    if (!result || !result.contents) return null

    return {
        pos: line.from,
        end: line.to,
        above: true,
        create() {
            const dom = document.createElement('div')
            dom.className = 'lsp-hover-tooltip'
            dom.style.cssText = 'max-width:500px;padding:8px 12px;font-size:12px;font-family:var(--font-mono);'
            dom.innerHTML = formatMarkdown(result.contents)  // 需要 highlight.js 或简单 markdown 渲染
            return { dom }
        },
    }
}, { hoverTime: 500 })
```

**注意:** `@codemirror/tooltip` 的 `hoverTooltip` 实际导入路径需要确认——可能是 `@codemirror/tooltip` 或通过 `@codemirror/view`。

---

### Task 16: 右键上下文菜单

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加右键菜单扩展**

```typescript
EditorView.domEventHandlers({
    contextmenu: (event, view) => {
        if (!lspEnabled) return
        event.preventDefault()

        const pos = view.posAtCoords({ x: event.clientX, y: event.clientY })
        if (pos === null) return
        const line = view.state.doc.lineAt(pos)
        const lineNum = line.number - 1
        const col = pos - line.from

        showContextMenu(event.clientX, event.clientY, view, lineNum, col)
    },
})
```

**Step 2: 实现 showContextMenu 函数**

创建自定义 HTML 右键菜单，包含以下选项：
- "Go to Definition" (Ctrl+Click 等效)
- "Go to Type Definition"
- "Find Implementations"  
- "Find All References"
- "Rename Symbol..."
- "Code Actions..."

菜单样式参考现有项目的深色主题（`#08090d` 背景）。

---

### Task 17: 诊断装饰

**Files:**
- Create: `frontend/src/lib/lspDecorations.ts`
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 创建 LSP 装饰扩展**

```typescript
// lspDecorations.ts
import { StateField, StateEffect } from '@codemirror/state'
import { Decoration, EditorView } from '@codemirror/view'
import { LspDiagnostic } from './lspService'

const setDiagnostics = StateEffect.define<LspDiagnostic[]>()

const diagnosticDeco = (diag: LspDiagnostic): Decoration => {
    const severityClass = diag.severity === 1 ? 'lsp-error'
        : diag.severity === 2 ? 'lsp-warning'
        : diag.severity === 3 ? 'lsp-info'
        : 'lsp-hint'
    return Decoration.mark({ class: `lsp-diagnostic ${severityClass}`, attributes: { title: diag.message } })
}

export const lspDiagnosticField = StateField.define<Decoration[]>({
    create() { return [] },
    update(decos, tr) {
        for (const e of tr.effects) {
            if (e.is(setDiagnostics)) {
                return e.value.map(diagnosticDeco)
            }
        }
        return decos
    },
    provide: f => EditorView.decorations.from(f),
})

export function updateLspDiagnostics(view: EditorView, diags: LspDiagnostic[]) {
    view.dispatch({ effects: setDiagnostics.of(diags) })
}
```

**Step 2: 添加 CSS 样式**

在 `PreviewPanel.tsx` 的 `previewTheme` 中添加：

```typescript
'.lsp-diagnostic': { textDecoration: 'underline' },
'.lsp-error': { textDecorationColor: '#e06c75', textDecorationStyle: 'wavy' },
'.lsp-warning': { textDecorationColor: '#e5c07b', textDecorationStyle: 'wavy' },
'.lsp-info': { textDecorationColor: '#61afef', textDecorationStyle: 'dashed' },
'.lsp-hint': { textDecorationColor: '#5c6370', textDecorationStyle: 'dotted' },
```

**Step 3: 诊断拉取循环**

在编辑 onChange 中添加 debounce 的诊断拉取：

```typescript
// 编辑后 500ms 拉取诊断
clearTimeout(diagTimeout)
diagTimeout = setTimeout(async () => {
    const diags = await lspService.diagnostics(pp, fp)
    if (editorRef.current) {
        updateLspDiagnostics(editorRef.current, diags)
    }
}, 500)
```

---

### Task 18: 文档符号侧边栏

**Files:**
- Create: `frontend/src/components/Preview/LspSymbolSidebar.tsx`

**Step 1: 创建符号侧边栏组件**

```tsx
import { useState } from 'react'
import { LspSymbol } from '../../lib/lspService'

interface Props {
    symbols: LspSymbol[]
    onSymbolClick: (sym: LspSymbol) => void
    activeSymbolPath?: number[]
}

export function LspSymbolSidebar({ symbols, onSymbolClick, activeSymbolPath }: Props) {
    const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

    return (
        <div className="lsp-symbol-sidebar" style={{ width: 200, overflow: 'auto', borderLeft: '1px solid var(--border)' }}>
            {symbols.map((s, i) => (
                <SymbolNode key={i} sym={s} depth={0} onClick={onSymbolClick}
                    collapsed={collapsed} onToggle={(name) => {
                        const next = new Set(collapsed)
                        next.has(name) ? next.delete(name) : next.add(name)
                        setCollapsed(next)
                    }} />
            ))}
        </div>
    )
}
```

**Step 2: 添加 SymbolNode 子组件**

递归渲染符号树，使用 ▸/▾ 表示展开/折叠状态。种类用图标区分（函数 ƒ、类 ◇、变量 x、接口 △）。

**Step 3: 集成到 PreviewPanel**

在编辑器右侧添加可折叠的侧边栏。通过 `preview. filePath` 查 `store.lspSymbols[filePath]` 获取符号数据。文件打开时调用 `lspService.documentSymbols()` 填充。

---

### Task 19: 面包屑（合并到底部状态栏）

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加面包屑逻辑**

```typescript
function getBreadcrumbs(symbols: LspSymbol[], line: number): LspSymbol[] {
    const path: LspSymbol[] = []
    function walk(syms: LspSymbol[]): boolean {
        for (const s of syms) {
            if (line >= s.startLine && line <= s.endLine) {
                path.push(s)
                if (s.children && s.children.length > 0) walk(s.children)
                return true
            }
        }
        return false
    }
    walk(symbols)
    return path
}
```

**Step 2: 渲染面包屑**

在底部状态栏区域添加面包屑 HTML，与现有关标位置/语言信息合并为一行。监听光标位置变化（通过 `editor.domEventHandlers({ click: ... })` 或 `EditorView.updateListener`），更新面包屑显示。

面包屑项可点击弹出下拉，显示同级符号列表。

---

### Task 20: 引用查看 Peek Panel

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加 Peek Panel 状态和渲染**

```typescript
const [peekPanel, setPeekPanel] = useState<{
    title: string
    items: LspLocation[]
    range?: [number, number]
} | null>(null)
```

在编辑器下方渲染一个 peek panel（类似 VS Code 的 reference view），显示引用列表。每行显示文件路径、行号和代码片段。

点击某个引用项 → 调用 `navigateToLocation` 跳转。

---

### Task 21: 内联 Rename

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 实现内联 rename 功能**

```typescript
async function startRename(view: EditorView, line: number, col: number) {
    const store = useStore.getState()
    const fp = store.preview.filePath
    const pp = store.projectPath
    if (!fp || !pp) return

    // 获取当前行文本中光标处的符号
    const lineText = view.state.doc.line(line + 1).text
    const word = extractWordAt(lineText, col)
    if (!word) return

    // 这里可以创建一个 input 覆盖在符号上
    // 或者用浏览器的 prompt
    const newName = prompt('Rename symbol:', word)
    if (!newName || newName === word) return

    const result = await lspService.rename(pp, fp, line, col, newName)
    if (result) {
        await applyWorkspaceEdit(result, view, pp)
    }
}
```

**Step 2: 实现 applyWorkspaceEdit**

```typescript
import { App } from '../../bindings/monika'

async function applyWorkspaceEdit(edit: LspWorkspaceEdit, view: EditorView, projectPath: string) {
    const curPath = useStore.getState().preview.filePath

    for (const fileEdit of edit.changes) {
        if (fileEdit.path === curPath) {
            // 当前文件：直接 dispatch 到 CodeMirror
            const changes = fileEdit.edits.map(e => ({
                from: view.state.doc.line(e.startLine + 1).from + e.startCol,
                to: view.state.doc.line(e.endLine + 1).from + e.endCol,
            }))
            // 应用编辑 ...
        } else {
            // 其他文件：ReadFile → apply edits → WriteFile
            const fc = await App.ReadFile(projectPath, fileEdit.path)
            let content = fc.content
            // 从后往前应用编辑（避免位置偏移）
            const sorted = [...fileEdit.edits].sort((a, b) => b.startLine - a.startLine)
            // ... apply edits to content
            await App.WriteFile(projectPath, fileEdit.path, content)
        }
    }
}
```

---

### Task 22: Code Actions

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 在右键菜单中实现 Code Actions 功能**

右键菜单选择 "Code Actions..." → 调用 `lspService.codeActions()` → 弹出可用 action 列表 → 选择后调用 `lspService.executeCodeAction()` → 用 `applyWorkspaceEdit` 应用结果。

---

### Task 23: 导航历史（Alt+← →）

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

**Step 1: 添加导航历史状态**

```typescript
interface NavEntry { path: string; line: number; col: number }
const navHistory = useRef<NavEntry[]>([])
const navIndex = useRef(-1)

function pushNavHistory(entry: NavEntry) {
    navHistory.current = navHistory.current.slice(0, navIndex.current + 1)
    navHistory.current.push(entry)
    navIndex.current++
}

function goBack() {
    if (navIndex.current > 0) {
        navIndex.current--
        const entry = navHistory.current[navIndex.current]
        navigateToLocation(entry)
    }
}

function goForward() {
    if (navIndex.current < navHistory.current.length - 1) {
        navIndex.current++
        const entry = navHistory.current[navIndex.current]
        navigateToLocation(entry)
    }
}
```

**Step 2: 添加键盘快捷键**

```typescript
import { keymap } from '@codemirror/view'

keymap.of([
    { key: 'Alt-ArrowLeft', run: () => { goBack(); return true } },
    { key: 'Alt-ArrowRight', run: () => { goForward(); return true } },
])
```

---

### Task 24: AI 并发编辑处理

**Files:**
- Modify: `frontend/src/store/index.ts`

**Step 1: 在 setupWailsEvents 中处理 file_changed 事件**

在现有的 `processEvent` 的 `file_changed` case 中添加逻辑：

```typescript
case 'file_changed':
    // ... existing logic ...
    // 检查是否影响当前预览文件
    const fp = fileChange.path
    const curPreview = useStore.getState().preview
    if (curPreview.mode === 'file' && curPreview.filePath === fp) {
        // 弹出提示或标记需要刷新
        useStore.setState({ previewNeedsRefresh: fp })
    }
    break
```

**Step 2: 在 PreviewPanel 中处理刷新**

当 `previewNeedsRefresh` 状态变化时，显示 "文件已被修改，点击刷新" 的提示条。

---

## 验证清单

- [ ] 后端编译通过: `go build ./...`
- [ ] 后端测试通过: `go test ./...`
- [ ] 前端编译通过: `cd frontend && npm run build`
- [ ] LspOpenFile / LspCloseFile 正常开关 LSP server
- [ ] 打开 `.go` / `.ts` 文件后 LSP server 自动启动
- [ ] Ctrl+Click 跳转到定义（同文件和跨文件）
- [ ] 鼠标悬停显示类型信息
- [ ] 右键菜单显示所有 LSP 选项
- [ ] 诊断波浪线显示在代码上
- [ ] 符号侧边栏可展开/折叠/点击跳转
- [ ] 面包屑随光标移动更新
- [ ] Alt+← → 导航历史正确
- [ ] 编辑模式切换正常（只读 ↔ 可编辑）
- [ ] 编辑后自动保存 + 通知 LSP
- [ ] AI 编辑文件后预览提示刷新
