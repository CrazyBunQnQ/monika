package lsp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSON-RPC 2.0 types

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// LSP base types

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Initialization

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Synchronization *SynchronizationCapabilities `json:"synchronization,omitempty"`
	Hover           *HoverCapabilities           `json:"hover,omitempty"`
	Definition      *DefinitionCapabilities      `json:"definition,omitempty"`
	References      *ReferencesCapabilities      `json:"references,omitempty"`
	DocumentSymbol  *DocumentSymbolCapabilities  `json:"documentSymbol,omitempty"`
	PublishDiagnostics *PublishDiagnosticsCapabilities `json:"publishDiagnostics,omitempty"`
}

type SynchronizationCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	WillSave            bool `json:"willSave"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil"`
	DidSave             bool `json:"didSave"`
}

type HoverCapabilities struct {
	DynamicRegistration bool     `json:"dynamicRegistration"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

type DefinitionCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	LinkSupport         bool `json:"linkSupport"`
}

type ReferencesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type DocumentSymbolCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	HierarchicalDocumentSymbolSupport bool `json:"hierarchicalDocumentSymbolSupport"`
}

type PublishDiagnosticsCapabilities struct {
	RelatedInformation bool `json:"relatedInformation"`
}

type WorkspaceClientCapabilities struct {
	Symbol *SymbolCapabilities `json:"symbol,omitempty"`
}

type SymbolCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type InitializeParams struct {
	ProcessID     int                `json:"processId"`
	RootURI       string             `json:"rootUri,omitempty"`
	Capabilities  ClientCapabilities `json:"capabilities"`
	ClientInfo     *ClientInfo        `json:"clientInfo,omitempty"`
	InitializationOptions any         `json:"initializationOptions,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	DefinitionProvider           any  `json:"definitionProvider,omitempty"`
	ReferencesProvider           any  `json:"referencesProvider,omitempty"`
	HoverProvider                any  `json:"hoverProvider,omitempty"`
	DocumentSymbolProvider       any  `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider      any  `json:"workspaceSymbolProvider,omitempty"`
	TextDocumentSync             any  `json:"textDocumentSync,omitempty"`
}

// Document sync

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string               `json:"text,omitempty"`
}

// Diagnostics

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Version     int          `json:"version,omitempty"`
}

type Diagnostic struct {
	Range    Range        `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Code     any          `json:"code,omitempty"`
	Source   string       `json:"source,omitempty"`
	Message  string       `json:"message"`
}

type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// Hover

type Hover struct {
	Contents json.RawMessage `json:"contents"`
	Range    *Range          `json:"range,omitempty"`
}

func (h *Hover) ContentText() string {
	if len(h.Contents) == 0 {
		return ""
	}

	// Try MarkupContent: {"kind":"...","value":"..."}
	var mc struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if json.Unmarshal(h.Contents, &mc) == nil && mc.Value != "" {
		return mc.Value
	}

	// Try MarkedString: "plain text"
	var s string
	if json.Unmarshal(h.Contents, &s) == nil {
		return s
	}

	// Try MarkedString: {"language":"...","value":"..."}
	var ms struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(h.Contents, &ms) == nil && ms.Value != "" {
		return ms.Value
	}

	// Try MarkedString[]
	var mss []json.RawMessage
	if json.Unmarshal(h.Contents, &mss) == nil {
		var parts []string
		for _, raw := range mss {
			var v string
			if json.Unmarshal(raw, &v) == nil {
				parts = append(parts, v)
			} else {
				var ms2 struct{ Value string `json:"value"` }
				if json.Unmarshal(raw, &ms2) == nil {
					parts = append(parts, ms2.Value)
				}
			}
		}
		return strings.Join(parts, "\n\n")
	}

	return string(h.Contents)
}

// Document symbols

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type SymbolInformation struct {
	Name     string     `json:"name"`
	Kind     SymbolKind `json:"kind"`
	Location Location   `json:"location"`
}

type SymbolKind int

const (
	SKFile SymbolKind = 1 + iota
	SKModule
	SKNamespace
	SKPackage
	SKClass
	SKMethod
	SKProperty
	SKField
	SKConstructor
	SKEnum
	SKInterface
	SKFunction
	SKVariable
	SKConstant
	SKString
	SKNumber
	SKBoolean
	SKArray
	SKObject
	SKKey
	SKNull
	SKEnumMember
	SKStruct
	SKEvent
	SKOperator
	SKTypeParameter
)

// Reference context

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext        `json:"context"`
}

// Text edits

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type TextDocumentEdit struct {
	TextDocument OptionalVersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                              `json:"edits"`
}

type OptionalVersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version *int   `json:"version,omitempty"`
}

type WorkspaceEdit struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []DocumentChange       `json:"documentChanges,omitempty"`
}

// DocumentChange is a union: TextDocumentEdit | CreateFile | RenameFile | DeleteFile
type DocumentChange struct {
	TextDocument *TextDocumentEdit `json:"-"`
	CreateFile   *CreateFile       `json:"-"`
	RenameFile   *RenameFile       `json:"-"`
	DeleteFile   *DeleteFile       `json:"-"`
	kind         string
}

func (dc DocumentChange) MarshalJSON() ([]byte, error) {
	switch dc.kind {
	case "textDocument":
		return json.Marshal(dc.TextDocument)
	case "create":
		return json.Marshal(dc.CreateFile)
	case "rename":
		return json.Marshal(dc.RenameFile)
	case "delete":
		return json.Marshal(dc.DeleteFile)
	}
	return nil, fmt.Errorf("unknown document change kind: %s", dc.kind)
}

func (dc *DocumentChange) UnmarshalJSON(data []byte) error {
	var probe struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(data, &probe) == nil && probe.Kind != "" {
		switch probe.Kind {
		case "create":
			dc.kind = "create"
			dc.CreateFile = &CreateFile{}
			return json.Unmarshal(data, dc.CreateFile)
		case "rename":
			dc.kind = "rename"
			dc.RenameFile = &RenameFile{}
			return json.Unmarshal(data, dc.RenameFile)
		case "delete":
			dc.kind = "delete"
			dc.DeleteFile = &DeleteFile{}
			return json.Unmarshal(data, dc.DeleteFile)
		}
	}

	dc.kind = "textDocument"
	dc.TextDocument = &TextDocumentEdit{}
	return json.Unmarshal(data, dc.TextDocument)
}

func ChangeTextDocument(edit TextDocumentEdit) DocumentChange {
	return DocumentChange{kind: "textDocument", TextDocument: &edit}
}

func ChangeCreateFile(cf CreateFile) DocumentChange {
	return DocumentChange{kind: "create", CreateFile: &cf}
}

func ChangeRenameFile(rf RenameFile) DocumentChange {
	return DocumentChange{kind: "rename", RenameFile: &rf}
}

func ChangeDeleteFile(df DeleteFile) DocumentChange {
	return DocumentChange{kind: "delete", DeleteFile: &df}
}

type CreateFile struct {
	Kind    string            `json:"kind"`
	URI     DocumentURI       `json:"uri"`
	Options *CreateFileOptions `json:"options,omitempty"`
}

type CreateFileOptions struct {
	Overwrite      bool `json:"overwrite,omitempty"`
	IgnoreIfExists bool `json:"ignoreIfExists,omitempty"`
}

type RenameFile struct {
	Kind    string            `json:"kind"`
	OldURI  DocumentURI       `json:"oldUri"`
	NewURI  DocumentURI       `json:"newUri"`
	Options *RenameFileOptions `json:"options,omitempty"`
}

type RenameFileOptions struct {
	Overwrite      bool `json:"overwrite,omitempty"`
	IgnoreIfExists bool `json:"ignoreIfExists,omitempty"`
}

type DeleteFile struct {
	Kind    string            `json:"kind"`
	URI     DocumentURI       `json:"uri"`
	Options *DeleteFileOptions `json:"options,omitempty"`
}

type DeleteFileOptions struct {
	Recursive          bool `json:"recursive,omitempty"`
	IgnoreIfNotExists  bool `json:"ignoreIfNotExists,omitempty"`
}

type DocumentURI = string

// Code Actions

type CodeActionKind string

const (
	CAKEmpty           CodeActionKind = ""
	CAKQuickFix        CodeActionKind = "quickfix"
	CAKRefactor        CodeActionKind = "refactor"
	CAKRefactorExtract CodeActionKind = "refactor.extract"
	CAKRefactorInline  CodeActionKind = "refactor.inline"
	CAKRefactorRewrite CodeActionKind = "refactor.rewrite"
	CAKSource          CodeActionKind = "source"
	CAKSourceOrganize  CodeActionKind = "source.organizeImports"
)

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

type CodeActionContext struct {
	Diagnostics []Diagnostic   `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

type CodeAction struct {
	Title       string        `json:"title"`
	Kind        CodeActionKind `json:"kind,omitempty"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
	IsPreferred *bool         `json:"isPreferred,omitempty"`
	Disabled    *CodeActionDisabled `json:"disabled,omitempty"`
 Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

type CodeActionDisabled struct {
	Reason string `json:"reason"`
}

type Command struct {
	Title     string   `json:"title"`
	Command   string   `json:"command"`
	Arguments []any    `json:"arguments,omitempty"`
}

type ExecuteCommandParams struct {
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

// Rename

type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

// Definition response can be Location, Location[], or LocationLink[]
// We unify into []Location for simplicity.

// Workspace symbol query

type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}
