package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var lspLogFile *os.File

func init() {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		f, err := os.OpenFile(filepath.Join(dir, "lsp_debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			lspLogFile = f
		}
	}
	// fallback: try current directory
	if lspLogFile == nil {
		f, err := os.OpenFile("lsp_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			lspLogFile = f
		}
	}
}

func lspLog(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if lspLogFile != nil {
		lspLogFile.WriteString(time.Now().Format("15:04:05.000") + " " + msg + "\n")
		lspLogFile.Sync()
	}
}

func CloseLogFile() {
	if lspLogFile != nil {
		lspLogFile.Close()
		lspLogFile = nil
	}
}

type Client struct {
	name         string
	transport    *Transport
	cmd          *exec.Cmd
	nextID       atomic.Int64
	mu           sync.Mutex
	pending      map[int64]chan *jsonRPCResponse
	diags        map[string][]Diagnostic
	diagSeq      map[string]int64
	diagMu       sync.RWMutex
	serverCaps   ServerCapabilities
	settings     map[string]any
	ready        bool
	shutdownOnce sync.Once
	done         chan struct{}

	// $/progress tracking
	activeProgressTokens map[any]struct{}
	progressMu           sync.Mutex
	projectLoadedOnce    sync.Once
	projectLoadedChan    chan struct{}
}

func NewClient(ctx context.Context, name string, command string, args []string, workdir string) (*Client, error) {
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
		name:                 name,
		transport:            NewTransport(stdin, stdout),
		cmd:                  cmd,
		pending:              make(map[int64]chan *jsonRPCResponse),
		diags:                make(map[string][]Diagnostic),
		diagSeq:              make(map[string]int64),
		done:                 make(chan struct{}),
		activeProgressTokens: make(map[any]struct{}),
		projectLoadedChan:    make(chan struct{}),
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
				TypeDefinition:  &TypeDefinitionCapabilities{LinkSupport: true},
				Implementation:  &ImplementationCapabilities{LinkSupport: true},
				References:      &ReferencesCapabilities{},
				DocumentSymbol:  &DocumentSymbolCapabilities{HierarchicalDocumentSymbolSupport: true},
				Rename:          &RenameCapabilities{PrepareSupport: true},
				CodeAction: &CodeActionCapabilities{
					CodeActionLiteralSupport: &CodeActionLiteralSupport{
						CodeActionKind: CodeActionKindSupport{
							ValueSet: []string{"quickfix", "refactor", "refactor.extract", "refactor.inline", "refactor.rewrite", "source", "source.organizeImports"},
						},
					},
					ResolveSupport: &CodeActionResolveSupport{Properties: []string{"edit"}},
				},
				PublishDiagnostics: &PublishDiagnosticsCapabilities{
					RelatedInformation: true,
					VersionSupport:     true,
					CodeDescriptionSupport: true,
					DataSupport:        true,
					TagSupport:         &PublishDiagnosticsTagSupport{ValueSet: []int{1, 2}},
				},
			},
			Workspace: &WorkspaceClientCapabilities{
				ApplyEdit:     true,
				Configuration: true,
				Symbol:        &SymbolCapabilities{},
				WorkspaceEdit: &WorkspaceEditCapabilities{
					DocumentChanges:    true,
					ResourceOperations: []string{"create", "rename", "delete"},
					FailureHandling:    "textOnlyTransactional",
				},
				FileOperations: &FileOperationsCapabilities{
					WillRename: true,
					DidRename:  true,
				},
			},
			Window: &WindowClientCapabilities{WorkDoneProgress: true},
		},
		ClientInfo:            &ClientInfo{Name: "monika", Version: "0.1.0"},
		InitializationOptions: initOptions,
	}

	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("lsp initialize: %w", err)
	}
	c.serverCaps = result.Capabilities
	if s, ok := settings.(map[string]any); ok {
		c.settings = s
	}

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
	c.shutdownOnce.Do(func() {
		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = c.call(ctx2, "shutdown", nil, nil)
		_ = c.notify(ctx2, "exit", nil)
		c.transport.Close()

		done := make(chan struct{})
		go func() {
			c.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			c.cmd.Process.Kill()
		}
		close(c.done)
	})
}

func (c *Client) IsAlive() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
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

// Formatting sends a textDocument/formatting request and returns the text edits.
func (c *Client) Formatting(ctx context.Context, uri string, opts FormattingOptions) ([]TextEdit, error) {
	if !c.Ready() {
		return nil, nil
	}
	var result []TextEdit
	if err := c.call(ctx, "textDocument/formatting", DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Options:      opts,
	}, &result); err != nil {
		// Many servers don't support formatting — treat as non-fatal
		return nil, nil
	}
	return result, nil
}

// WillRenameFiles sends a workspace/willRenameFiles request to the server
// and returns the workspace edit response (which may contain refactoring changes).
func (c *Client) WillRenameFiles(ctx context.Context, files []FileRename) (*WorkspaceEdit, error) {
	var result WorkspaceEdit
	if err := c.call(ctx, "workspace/willRenameFiles", RenameFilesParams{Files: files}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DidRenameFiles notifies the server that files were renamed.
func (c *Client) DidRenameFiles(ctx context.Context, files []FileRename) error {
	return c.notify(ctx, "workspace/didRenameFiles", RenameFilesParams{Files: files})
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

func (c *Client) ExecuteCodeAction(ctx context.Context, action CodeAction) (*WorkspaceEdit, error) {
	if action.Edit != nil {
		return action.Edit, nil
	}
	if action.Command != nil {
		var raw json.RawMessage
		if err := c.call(ctx, "workspace/executeCommand", ExecuteCommandParams{
			Command:   action.Command.Command,
			Arguments: action.Command.Arguments,
		}, &raw); err != nil {
			return nil, err
		}
		var edit WorkspaceEdit
		if json.Unmarshal(raw, &edit) == nil && (len(edit.Changes) > 0 || len(edit.DocumentChanges) > 0) {
			return &edit, nil
		}
		return nil, nil
	}
	return nil, nil
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
	if err := c.call(ctx, "textDocument/documentSymbol", struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}{TextDocument: TextDocumentIdentifier{URI: uri}}, &raw); err != nil {
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
	return dedupeAndSortDiagnostics(c.diags[normalizeURI(uri)])
}

// dedupeAndSortDiagnostics removes duplicates and sorts by severity then position.
func dedupeAndSortDiagnostics(diags []Diagnostic) []Diagnostic {
	if len(diags) <= 1 {
		return diags
	}

	seen := make(map[string]bool)
	result := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		key := fmt.Sprintf("%d:%d-%d:%d:%s", d.Range.Start.Line, d.Range.Start.Character, d.Range.End.Line, d.Range.End.Character, d.Message)
		if !seen[key] {
			seen[key] = true
			result = append(result, d)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Severity != result[j].Severity {
			return result[i].Severity < result[j].Severity
		}
		if result[i].Range.Start.Line != result[j].Range.Start.Line {
			return result[i].Range.Start.Line < result[j].Range.Start.Line
		}
		return result[i].Range.Start.Character < result[j].Range.Start.Character
	})

	return result
}

func (c *Client) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready
}

func (c *Client) DiagSeq(uri string) int64 {
	c.diagMu.RLock()
	defer c.diagMu.RUnlock()
	return c.diagSeq[normalizeURI(uri)]
}

// ClearDiagnostics removes cached diagnostics for a URI.
// Call before DidChange to avoid accepting stale diagnostics.
func (c *Client) ClearDiagnostics(uri string) {
	c.diagMu.Lock()
	delete(c.diags, normalizeURI(uri))
	c.diagMu.Unlock()
}

func (c *Client) WaitForDiagUpdate(ctx context.Context, uri string, afterSeq int64, timeout time.Duration) bool {
	normURI := normalizeURI(uri)
	deadline := time.Now().Add(timeout)
	for {
		c.diagMu.RLock()
		cur := c.diagSeq[normURI]
		c.diagMu.RUnlock()
		if cur > afterSeq {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		default:
		}
		time.Sleep(30 * time.Millisecond)
	}
}

// readLoop runs in a background goroutine, reading JSON-RPC messages
// and dispatching responses to pending callers or caching diagnostics.
func (c *Client) readLoop() {
	defer func() {
		exitCode := -1
		if c.cmd.ProcessState != nil {
			exitCode = c.cmd.ProcessState.ExitCode()
		}
		errMsg := fmt.Sprintf("lsp %s: connection closed (exit code %d)", c.name, exitCode)

		c.mu.Lock()
		pending := make(map[int64]chan *jsonRPCResponse, len(c.pending))
		for id, ch := range c.pending {
			pending[id] = ch
			delete(c.pending, id)
		}
		c.mu.Unlock()
		for id, ch := range pending {
			ch <- &jsonRPCResponse{ID: id, Error: &jsonRPCError{Code: -32000, Message: errMsg}}
		}
		c.shutdownOnce.Do(func() {
			close(c.done)
			c.cmd.Process.Kill()
		})
	}()

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
		if envelope.Method != "" {
			lspLog("recv notification: method=%s params_len=%d", envelope.Method, len(envelope.Params))
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

		// Server-initiated request — route to appropriate handler
		if envelope.ID != 0 && envelope.Method != "" {
			c.handleServerRequest(envelope.ID, envelope.Method, envelope.Params)
			continue
		}

		// Server notification — route to appropriate handler
		if envelope.Method != "" && envelope.ID == 0 {
			c.handleServerNotification(envelope.Method, envelope.Params)
		}
	}
}

// handleServerRequest responds to server-initiated requests.
func (c *Client) handleServerRequest(id int64, method string, params json.RawMessage) {
	lspLog("recv server request: method=%s id=%d", method, id)

	switch method {
	case "workspace/configuration":
		var cfgReq struct {
			Items []struct {
				Section string `json:"section"`
			} `json:"items"`
		}
		result := make([]any, 0)
		if json.Unmarshal(params, &cfgReq) == nil && c.settings != nil {
			for _, item := range cfgReq.Items {
				if item.Section == "" {
					result = append(result, c.settings)
				} else {
					result = append(result, c.settings[item.Section])
				}
			}
		}
		c.sendResponse(id, result, nil)
	case "workspace/applyEdit":
		var editReq struct {
			Edit WorkspaceEdit `json:"edit"`
		}
		if json.Unmarshal(params, &editReq) == nil {
			_, err := ApplyWorkspaceEdit(editReq.Edit)
			if err != nil {
				c.sendResponse(id, map[string]any{"applied": false, "failureReason": err.Error()}, nil)
				return
			}
			c.sendResponse(id, map[string]any{"applied": true}, nil)
			return
		}
		c.sendResponse(id, map[string]any{"applied": false, "failureReason": "invalid params"}, nil)
	case "window/workDoneProgress/create":
		c.sendResponse(id, nil, nil)
	default:
		c.sendResponse(id, nil, &jsonRPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", method)})
	}
}

// handleServerNotification processes server notifications.
func (c *Client) handleServerNotification(method string, params json.RawMessage) {
	switch method {
	case "textDocument/publishDiagnostics":
		var diagParams PublishDiagnosticsParams
		if json.Unmarshal(params, &diagParams) == nil {
			lspLog("publishDiagnostics: raw_uri=%s diag_count=%d", diagParams.URI, len(diagParams.Diagnostics))
			uri := normalizeURI(diagParams.URI)
			lspLog("publishDiagnostics: normalized_uri=%s", uri)
			c.diagMu.Lock()
			if len(diagParams.Diagnostics) == 0 {
				delete(c.diags, uri)
			} else {
				c.diags[uri] = diagParams.Diagnostics
			}
			c.diagSeq[uri]++
			c.diagMu.Unlock()
		}
	case "$/progress":
		var progressParams struct {
			Token any    `json:"token"`
			Value struct {
				Kind string `json:"kind"`
			} `json:"value"`
		}
		if json.Unmarshal(params, &progressParams) == nil {
			c.progressMu.Lock()
			switch progressParams.Value.Kind {
			case "begin":
				c.activeProgressTokens[progressParams.Token] = struct{}{}
			case "end":
				delete(c.activeProgressTokens, progressParams.Token)
				if len(c.activeProgressTokens) == 0 {
					c.projectLoadedOnce.Do(func() {
						close(c.projectLoadedChan)
					})
				}
			}
			c.progressMu.Unlock()
		}
	}
}

// sendResponse writes a JSON-RPC response to the server.
func (c *Client) sendResponse(id int64, result any, err *jsonRPCError) {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: err}
	if err == nil && result != nil {
		raw, marshalErr := json.Marshal(result)
		if marshalErr == nil {
			resp.Result = raw
		}
	}
	if err == nil && result == nil {
		resp.Result = json.RawMessage("null")
	}
	_ = c.transport.WriteMessage(resp)
}

// WaitForProjectLoaded blocks until the server reports project loading
// is complete (via $/progress end), or until the timeout expires.
func (c *Client) WaitForProjectLoaded(ctx context.Context, timeout time.Duration) {
	select {
	case <-c.projectLoadedChan:
		return
	case <-ctx.Done():
		return
	case <-time.After(timeout):
		c.projectLoadedOnce.Do(func() {
			close(c.projectLoadedChan)
		})
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
		_ = c.notify(context.Background(), "$/cancelRequest", map[string]any{"id": id})
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

	// Return all actions with a non-empty title
	result := make([]CodeAction, 0, len(actions))
	for _, a := range actions {
		if a.Title != "" {
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
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs
	}
	if runtime.GOOS == "windows" && len(abs) > 2 && abs[0] == '/' && abs[2] == ':' {
		abs = "/" + strings.ToLower(abs[1:2]) + abs[2:]
	}
	u := &url.URL{Scheme: "file", Path: abs}
	return u.String()
}
// normalizeURI ensures consistent URI formatting for map lookups.
// TS server returns URIs with percent-encoded colons (e.g. d%3A),
// while fileToURI produces unencoded colons (e.g. d:).
// We decode and build a plain string to avoid url.URL.String() re-encoding.
func normalizeURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	if u.Scheme != "file" {
		return uri
	}
	path := u.Path
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}
	if runtime.GOOS == "windows" && len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = "/" + strings.ToLower(path[1:2]) + path[2:]
	}
	return "file://" + path
}


func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	p := u.Path
	// On Windows, url.Parse gives "/C:/foo" — strip leading /
	if runtime.GOOS == "windows" && len(p) > 2 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}
