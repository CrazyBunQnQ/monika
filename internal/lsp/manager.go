package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const idleTimeout = 5 * time.Minute

type LSPServerStatus struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	FileTypes []string `json:"fileTypes"`
	Running   bool     `json:"running"`
}

type openFile struct {
	version    int
	content    string
	serverName string
}

type managedClient struct {
	client   *Client
	lastUsed time.Time
}

type Manager struct {
	workdir    string
	servers    map[string]ServerConfig
	clients    map[string]*managedClient
	openFiles  map[string]*openFile // uri -> version + content
	mu         sync.Mutex
	clientLocks map[string]*sync.Mutex // per-server lock for getOrStart
	fileLocks  map[string]*sync.Mutex // per-file lock for open/close/change
	cancel     context.CancelFunc
}

func NewManager(workdir string) *Manager {
	return &Manager{
		workdir:     workdir,
		servers:     ResolveServers(workdir),
		clients:     make(map[string]*managedClient),
		openFiles:   make(map[string]*openFile),
		clientLocks: make(map[string]*sync.Mutex),
		fileLocks:   make(map[string]*sync.Mutex),
	}
}

func (m *Manager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.idleLoop(ctx)
	// Warm up LSP servers in the background so first use is fast.
	go m.Warmup(context.Background())
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, mc := range m.clients {
		mc.client.Shutdown(context.Background())
		delete(m.clients, name)
	}
	CloseLogFile()
}

func (m *Manager) ClientForFile(ctx context.Context, filePath string) (*Client, string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return nil, "", fmt.Errorf("lsp: no file extension for %q", filePath)
	}

	names := FileTypeToServer(ext, m.servers)
	if len(names) == 0 {
		return nil, "", fmt.Errorf("lsp: no server configured for %q files", ext)
	}

	name := names[0]
	return m.getOrStart(ctx, name)

}

// ReadyForFile returns true if an LSP client is running and initialized for this file.
func (m *Manager) ReadyForFile(ctx context.Context, filePath string) bool {
	client, _, err := m.ClientForFile(ctx, filePath)
	if err != nil {
		return false
	}
	return client.Ready()
}

func (m *Manager) Status() string {

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.servers) == 0 {
		return "No LSP servers configured for this project."
	}

	var sb strings.Builder
	sb.WriteString("LSP servers:\n")
	for name, cfg := range m.servers {
		mc, running := m.clients[name]
		status := "available"
		if running {
			status = fmt.Sprintf("running (last used %s)", time.Since(mc.lastUsed).Round(time.Second))
		}
		sb.WriteString(fmt.Sprintf("  %s: %s [%s] (%s)\n", name, cfg.Command, strings.Join(cfg.FileTypes, ","), status))
	}
	return sb.String()
}

func (m *Manager) ServerStatuses() []LSPServerStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]LSPServerStatus, 0, len(m.servers))
	for name, cfg := range m.servers {
		_, running := m.clients[name]
		result = append(result, LSPServerStatus{
			Name:      name,
			Command:   cfg.Command,
			FileTypes: cfg.FileTypes,
			Running:   running,
		})
	}
	return result
}

func (m *Manager) clientLock(name string) *sync.Mutex {
	m.mu.Lock()
	if lk, ok := m.clientLocks[name]; ok {
		m.mu.Unlock()
		return lk
	}
	lk := &sync.Mutex{}
	m.clientLocks[name] = lk
	m.mu.Unlock()
	return lk
}

func (m *Manager) fileLock(uri string) *sync.Mutex {
	m.mu.Lock()
	if lk, ok := m.fileLocks[uri]; ok {
		m.mu.Unlock()
		return lk
	}
	lk := &sync.Mutex{}
	m.fileLocks[uri] = lk
	m.mu.Unlock()
	return lk
}

func (m *Manager) getOrStart(ctx context.Context, name string) (*Client, string, error) {
	lk := m.clientLock(name)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	if mc, ok := m.clients[name]; ok {
		if mc.client.IsAlive() {
			mc.lastUsed = time.Now()
			m.mu.Unlock()
			return mc.client, name, nil
		}
// Client died; clean up and reconnect
		delete(m.clients, name)
		m.clearOpenFiles(name)
		m.mu.Unlock()
		return m.startClient(ctx, name)
	}
	m.mu.Unlock()
	return m.startClient(ctx, name)
}

func (m *Manager) startClient(ctx context.Context, name string) (*Client, string, error) {
	cfg, ok := m.servers[name]
	if !ok {
		return nil, "", fmt.Errorf("lsp: unknown server %q", name)
	}

	cmd := ResolveCommand(cfg.Command, m.workdir)

	args := make([]string, len(cfg.Args))
	for i, a := range cfg.Args {
		if a == "$PID" {
			args[i] = strconv.Itoa(os.Getpid())
		} else {
			args[i] = a
		}
	}

	client, err := NewClient(ctx, name, cmd, args, m.workdir)
	if err != nil {
		return nil, "", fmt.Errorf("lsp: start %s: %w", name, err)
	}

	rootURI := fileToURI(m.workdir)
	if err := client.Initialize(ctx, rootURI, cfg.InitOptions, cfg.Settings); err != nil {
		client.Shutdown(ctx)
		return nil, "", fmt.Errorf("lsp: initialize %s: %w", name, err)
	}

	m.mu.Lock()
	m.clients[name] = &managedClient{client: client, lastUsed: time.Now()}
	m.mu.Unlock()

	return client, name, nil
}

func (m *Manager) idleLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.shutdownIdle()
		}
	}
}

func (m *Manager) shutdownIdle() {
	var toClose []*Client
	m.mu.Lock()
	now := time.Now()
	for name, mc := range m.clients {
		if now.Sub(mc.lastUsed) > idleTimeout {
			toClose = append(toClose, mc.client)
			delete(m.clients, name)
			m.clearOpenFiles(name)
		}
	}
	m.mu.Unlock()

	for _, c := range toClose {
		c.Shutdown(context.Background())
	}
}


func (m *Manager) clearOpenFiles(serverName string) {
	for uri, of := range m.openFiles {
		if of.serverName == serverName {
			delete(m.openFiles, uri)
		}
	}
}

// EnsureFileOpen opens the file in the LSP server if not already open.
// Returns the current file content and version.
func (m *Manager) EnsureFileOpen(ctx context.Context, client *Client, filePath string, serverName string) (string, int, error) {
	uri := fileToURI(filePath)

	lk := m.fileLock(uri)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	of, alreadyOpen := m.openFiles[uri]
	m.mu.Unlock()

	if alreadyOpen && of.serverName == serverName {
		return of.content, of.version, nil
	}

	if alreadyOpen {
		// Server changed (reconnect); stale entry, re-open
		_ = client.DidClose(ctx, uri)
	}

	content, err := m.ReadFileContent(filePath)
	if err != nil {
		return "", 0, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	langID := extToLanguageID(ext)

	if err := client.DidOpen(ctx, TextDocumentItem{
		URI:        uri,
		LanguageID: langID,
		Version:    1,
		Text:       content,
	}); err != nil {
		return "", 0, fmt.Errorf("didOpen failed: %w", err)
	}

	m.mu.Lock()
	m.openFiles[uri] = &openFile{version: 1, content: content, serverName: serverName}
	m.mu.Unlock()

	return content, 1, nil
}

// SyncContent re-reads the file and sends didChange if content differs.
func (m *Manager) SyncContent(ctx context.Context, client *Client, filePath string) error {
	uri := fileToURI(filePath)

	lk := m.fileLock(uri)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	of, ok := m.openFiles[uri]
	m.mu.Unlock()

	if !ok {
		return nil
	}

	content, err := m.ReadFileContent(filePath)
	if err != nil {
		return err
	}

	if content == of.content {
		lspLog("SyncContent: uri=%s content_unchanged", uri)
		return nil
	}

	lspLog("SyncContent: uri=%s content_CHANGED sending didChange", uri)
	client.ClearDiagnostics(uri)
	of.version++
	of.content = content

	return client.DidChange(ctx, uri, of.version, content)
}

// EnsureAndSync atomically ensures the file is open in the LSP server and
// synced with the current on-disk content. Returns true if content was
// actually sent or updated to the server.
func (m *Manager) EnsureAndSync(ctx context.Context, client *Client, filePath string, serverName string) (bool, error) {
	uri := fileToURI(filePath)

	lk := m.fileLock(uri)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	of, alreadyOpen := m.openFiles[uri]
	m.mu.Unlock()

	if alreadyOpen && of.serverName == serverName {
		content, err := m.ReadFileContent(filePath)
		if err != nil {
			return false, err
		}
		if content == of.content {
			return false, nil
		}
		of.version++
		of.content = content
		client.ClearDiagnostics(uri)
		return true, client.DidChange(ctx, uri, of.version, content)
	}

	if alreadyOpen {
		_ = client.DidClose(ctx, uri)
	}

	content, err := m.ReadFileContent(filePath)
	if err != nil {
		return false, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	langID := extToLanguageID(ext)

	client.ClearDiagnostics(uri)
	if err := client.DidOpen(ctx, TextDocumentItem{
		URI:        uri,
		LanguageID: langID,
		Version:    1,
		Text:       content,
	}); err != nil {
		return false, fmt.Errorf("didOpen failed: %w", err)
	}

	m.mu.Lock()
	m.openFiles[uri] = &openFile{version: 1, content: content, serverName: serverName}
	m.mu.Unlock()

	return true, nil
}

// NotifySaved sends didSave for a file if it is currently open.
func (m *Manager) NotifySaved(ctx context.Context, client *Client, filePath string) error {
	uri := fileToURI(filePath)

	lk := m.fileLock(uri)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	of, ok := m.openFiles[uri]
	m.mu.Unlock()

	if !ok {
		return nil
	}

	return client.DidSave(ctx, uri, of.content)
}

// CloseFile sends didClose and removes the file from the open set.
func (m *Manager) CloseFile(ctx context.Context, client *Client, filePath string) error {
	uri := fileToURI(filePath)

	lk := m.fileLock(uri)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	_, ok := m.openFiles[uri]
	if ok {
		delete(m.openFiles, uri)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}

	return client.DidClose(ctx, uri)
}

func (m *Manager) ReadFileContent(filePath string) (string, error) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(m.workdir, filePath)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("lsp: read %s: %w", filePath, err)
	}
	return string(data), nil
}

// Warmup starts all detected LSP servers in parallel.
// Servers that fail to start will be lazily retried on first use.
func (m *Manager) Warmup(ctx context.Context) {
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for name := range m.servers {
		wg.Add(1)
		go func(serverName string) {
			defer wg.Done()
			_, _, _ = m.getOrStart(ctx2, serverName)
		}(name)
	}
	wg.Wait()
}

// SyncContentFromMemory sends didChange (or didOpen) with the provided content
// without reading from disk. Use this when the caller already holds the file
// content (e.g. after a file edit).
func (m *Manager) SyncContentFromMemory(ctx context.Context, client *Client, filePath string, content string, serverName string) error {
	uri := fileToURI(filePath)

	lk := m.fileLock(uri)
	lk.Lock()
	defer lk.Unlock()

	m.mu.Lock()
	of, alreadyOpen := m.openFiles[uri]
	m.mu.Unlock()

	if alreadyOpen {
		// Only skip if the same server; otherwise treat as new open
		if of.serverName == serverName {
			client.ClearDiagnostics(uri)
			of.version++
			of.content = content
			return client.DidChange(ctx, uri, of.version, content)
		}
		_ = client.DidClose(ctx, uri)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	langID := extToLanguageID(ext)

	client.ClearDiagnostics(uri)
	if err := client.DidOpen(ctx, TextDocumentItem{
		URI:        uri,
		LanguageID: langID,
		Version:    1,
		Text:       content,
	}); err != nil {
		return fmt.Errorf("didOpen failed: %w", err)
	}

	m.mu.Lock()
	m.openFiles[uri] = &openFile{version: 1, content: content, serverName: serverName}
	m.mu.Unlock()

	return nil
}

// WriteThroughOptions controls the WriteThrough pipeline behavior.
type WriteThroughOptions struct {
	FormatOnWrite    bool
	DiagnosticsOnEdit bool
}

// FormatContent sends a textDocument/formatting request, applies the edits,
// and writes the formatted content back to disk. Returns the formatted content.
func (m *Manager) FormatContent(ctx context.Context, client *Client, filePath string) (string, error) {
	uri := fileToURI(filePath)

	edits, err := client.Formatting(ctx, uri, FormattingOptions{TabSize: 4, InsertSpaces: true})
	if err != nil || len(edits) == 0 {
		// Read original content if nothing changed
		content, err := m.ReadFileContent(filePath)
		if err != nil {
			return "", err
		}
		return content, nil
	}

	content, err := m.ReadFileContent(filePath)
	if err != nil {
		return "", err
	}

	newContent, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		return content, nil // fall back to original
	}

	if newContent == content {
		return content, nil
	}

	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		return content, nil // fall back to original if write fails
	}

	return newContent, nil
}

// WriteThrough runs the full LSP writethrough pipeline:
// 1. Sync content to LSP (didOpen/didChange)
// 2. Optionally format via LSP and write back to disk
// 3. Notify server of save (didSave)
// 4. Optionally wait for fresh diagnostics
// Returns the final content and any diagnostics text.
func (m *Manager) WriteThrough(ctx context.Context, client *Client, filePath string, serverName string, content string, opts WriteThroughOptions) (string, string) {
	uri := fileToURI(filePath)
	beforeDiagSeq := client.DiagSeq(uri)

	// Step 1: Sync content (in-memory, no disk read)
	if err := m.SyncContentFromMemory(ctx, client, filePath, content, serverName); err != nil {
		lspLog("WriteThrough: sync failed: %v", err)
	}

	finalContent := content

	// Step 2: Optional format
	if opts.FormatOnWrite {
		formatted, err := m.FormatContent(ctx, client, filePath)
		if err == nil && formatted != content {
			// Content changed — re-sync with formatted content
			finalContent = formatted
			_ = m.SyncContentFromMemory(ctx, client, filePath, formatted, serverName)
		}
	}

	// Step 3: Write to disk
	if err := os.WriteFile(filePath, []byte(finalContent), 0o644); err != nil {
		lspLog("WriteThrough: write failed: %v", err)
	}

	// Step 4: Notify saved
	_ = m.NotifySaved(ctx, client, filePath)

	// Step 5: Wait for diagnostics
	if opts.DiagnosticsOnEdit {
		client.WaitForDiagUpdate(ctx, uri, beforeDiagSeq, 3*time.Second)
	}

	return finalContent, ""
}
