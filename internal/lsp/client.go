package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Client struct {
	transport  *Transport
	cmd        *exec.Cmd
	nextID     atomic.Int64
	mu         sync.Mutex
	pending    map[int64]chan *jsonRPCResponse
	diags      map[string][]Diagnostic
	diagMu     sync.RWMutex
	serverCaps ServerCapabilities
	ready      bool
	done       chan struct{}
}

func NewClient(ctx context.Context, command string, args []string, workdir string) (*Client, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workdir
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("lsp: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("lsp: start %s: %w", command, err)
	}

	c := &Client{
		transport: NewTransport(stdin, stdout),
		cmd:       cmd,
		pending:   make(map[int64]chan *jsonRPCResponse),
		diags:     make(map[string][]Diagnostic),
		done:      make(chan struct{}),
	}

	go c.readLoop()

	return c, nil
}

func (c *Client) Initialize(ctx context.Context, rootURI string, initOptions any, settings any) error {
	params := InitializeParams{
		ProcessID: 0,
		RootURI:   rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: &TextDocumentClientCapabilities{
				Synchronization: &SynchronizationCapabilities{DidSave: true},
				Hover:           &HoverCapabilities{ContentFormat: []string{"markdown", "plaintext"}},
				Definition:      &DefinitionCapabilities{LinkSupport: true},
				References:      &ReferencesCapabilities{},
				DocumentSymbol:  &DocumentSymbolCapabilities{HierarchicalDocumentSymbolSupport: true},
				PublishDiagnostics: &PublishDiagnosticsCapabilities{RelatedInformation: true},
			},
			Workspace: &WorkspaceClientCapabilities{
				Symbol: &SymbolCapabilities{},
			},
		},
		ClientInfo:            &ClientInfo{Name: "monika", Version: "0.1.0"},
		InitializationOptions: initOptions,
	}

	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("lsp initialize: %w", err)
	}
	c.serverCaps = result.Capabilities

	if settings != nil {
		_ = c.notify(ctx, "workspace/didChangeConfiguration", map[string]any{
			"settings": settings,
		})
	}

	_ = c.notify(ctx, "initialized", map[string]any{})

	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	return nil
}

func (c *Client) Shutdown(ctx context.Context) {
	select {
	case <-c.done:
		return
	default:
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = c.call(ctx2, "shutdown", nil, nil)
	_ = c.notify(ctx2, "exit", nil)
	c.transport.Close()
	c.cmd.Wait()
	close(c.done)
}

func (c *Client) DidOpen(ctx context.Context, doc TextDocumentItem) error {
	return c.notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{TextDocument: doc})
}

func (c *Client) DidChange(ctx context.Context, uri string, version int, content string) error {
	return c.notify(ctx, "textDocument/didChange", DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{URI: uri, Version: version},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: content},
		},
	})
}

func (c *Client) DidClose(ctx context.Context, uri string) error {
	return c.notify(ctx, "textDocument/didClose", DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
}

func (c *Client) DidSave(ctx context.Context, uri string, text string) error {
	params := DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	if text != "" {
		params.Text = &text
	}
	return c.notify(ctx, "textDocument/didSave", params)
}

func (c *Client) Definition(ctx context.Context, uri string, pos Position) ([]Location, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "textDocument/definition", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}, &raw); err != nil {
		return nil, err
	}
	return parseLocations(raw)
}

func (c *Client) TypeDefinition(ctx context.Context, uri string, pos Position) ([]Location, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "textDocument/typeDefinition", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}, &raw); err != nil {
		return nil, err
	}
	return parseLocations(raw)
}

func (c *Client) Implementation(ctx context.Context, uri string, pos Position) ([]Location, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "textDocument/implementation", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}, &raw); err != nil {
		return nil, err
	}
	return parseLocations(raw)
}

func (c *Client) CodeActions(ctx context.Context, uri string, r Range, diags []Diagnostic) ([]CodeAction, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "textDocument/codeAction", CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        r,
		Context:      CodeActionContext{Diagnostics: diags},
	}, &raw); err != nil {
		return nil, err
	}
	return parseCodeActions(raw)
}

func (c *Client) Rename(ctx context.Context, uri string, pos Position, newName string) (*WorkspaceEdit, error) {
	var result WorkspaceEdit
	if err := c.call(ctx, "textDocument/rename", RenameParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
		NewName:      newName,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) References(ctx context.Context, uri string, pos Position) ([]Location, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "textDocument/references", ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
		Context:      ReferenceContext{IncludeDeclaration: true},
	}, &raw); err != nil {
		return nil, err
	}
	return parseLocations(raw)
}

func (c *Client) Hover(ctx context.Context, uri string, pos Position) (*Hover, error) {
	var result Hover
	if err := c.call(ctx, "textDocument/hover", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "textDocument/documentSymbol", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}, &raw); err != nil {
		return nil, err
	}

	var dsyms []DocumentSymbol
	if err := json.Unmarshal(raw, &dsyms); err == nil && len(dsyms) > 0 {
		return dsyms, nil
	}

	var sinfos []SymbolInformation
	if err := json.Unmarshal(raw, &sinfos); err != nil {
		return nil, err
	}
	result := make([]DocumentSymbol, 0, len(sinfos))
	for _, si := range sinfos {
		result = append(result, DocumentSymbol{
			Name:           si.Name,
			Kind:           si.Kind,
			Range:          si.Location.Range,
			SelectionRange: si.Location.Range,
		})
	}
	return result, nil
}

func (c *Client) WorkspaceSymbols(ctx context.Context, query string) ([]SymbolInformation, error) {
	var result []SymbolInformation
	if err := c.call(ctx, "workspace/symbol", WorkspaceSymbolParams{Query: query}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Diagnostics(uri string) []Diagnostic {
	c.diagMu.RLock()
	defer c.diagMu.RUnlock()
	return c.diags[uri]
}

func (c *Client) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready
}

// readLoop runs in a background goroutine, reading JSON-RPC messages
// and dispatching responses to pending callers or caching diagnostics.
func (c *Client) readLoop() {
	for {
		msg, err := c.transport.ReadMessage()
		if err != nil {
			return
		}

		var envelope struct {
			ID     int64           `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result,omitempty"`
			Error  *jsonRPCError   `json:"error,omitempty"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}

		if envelope.ID != 0 && envelope.Method == "" {
			c.mu.Lock()
			ch, ok := c.pending[envelope.ID]
			if ok {
				delete(c.pending, envelope.ID)
			}
			c.mu.Unlock()
			if ok {
				resp := &jsonRPCResponse{ID: envelope.ID, Result: envelope.Result, Error: envelope.Error}
				ch <- resp
			}
			continue
		}

		if envelope.Method == "textDocument/publishDiagnostics" && envelope.Params != nil {
			var params PublishDiagnosticsParams
			if json.Unmarshal(envelope.Params, &params) == nil {
				c.diagMu.Lock()
				if len(params.Diagnostics) == 0 {
					delete(c.diags, params.URI)
				} else {
					c.diags[params.URI] = params.Diagnostics
				}
				c.diagMu.Unlock()
			}
		}
	}
}

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	id := c.nextID.Add(1)
	ch := make(chan *jsonRPCResponse, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	if err := c.transport.WriteMessage(req); err != nil {
		return fmt.Errorf("lsp send %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return fmt.Errorf("lsp %s: [%d] %s", method, resp.Error.Code, resp.Error.Message)
		}
		if result != nil && resp.Result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("lsp %s: decode result: %w", method, err)
			}
		}
		return nil
	}
}

func (c *Client) notify(ctx context.Context, method string, params any) error {
	n := jsonRPCNotification{JSONRPC: "2.0", Method: method, Params: params}
	if err := c.transport.WriteMessage(n); err != nil {
		return fmt.Errorf("lsp notify %s: %w", method, err)
	}
	return nil
}

// parseLocations handles LSP returning Location, Location[], or LocationLink[].
func parseLocations(raw json.RawMessage) ([]Location, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var locs []Location
	if err := json.Unmarshal(raw, &locs); err == nil {
		return locs, nil
	}

	var loc Location
	if err := json.Unmarshal(raw, &loc); err == nil {
		return []Location{loc}, nil
	}

	var links []struct {
		TargetURI string `json:"targetUri"`
		TargetRange Range `json:"targetRange"`
	}
	if err := json.Unmarshal(raw, &links); err == nil {
		result := make([]Location, 0, len(links))
		for _, l := range links {
			result = append(result, Location{URI: l.TargetURI, Range: l.TargetRange})
		}
		return result, nil
	}

	return nil, nil
}

// parseCodeActions handles the CodeAction response which can be
// CodeAction[] or (CodeAction | Command)[].
func parseCodeActions(raw json.RawMessage) ([]CodeAction, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	// Try as CodeAction[]
	var actions []CodeAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		return nil, err
	}

	// Filter out Command-only entries (they have no title field in CodeAction context)
	result := make([]CodeAction, 0, len(actions))
	for _, a := range actions {
		if a.Edit != nil || a.Command != nil || len(a.Diagnostics) > 0 {
			result = append(result, a)
		}
	}
	return result, nil
}

func fileToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if runtime.GOOS == "windows" {
		abs = filepath.ToSlash(abs)
	}
	return "file:///" + abs
}

func uriToPath(uri string) string {
	if len(uri) > 8 && uri[:8] == "file:///" {
		return filepath.FromSlash(uri[8:])
	}
	if len(uri) > 7 && uri[:7] == "file://" {
		return filepath.FromSlash(uri[7:])
	}
	return uri
}
