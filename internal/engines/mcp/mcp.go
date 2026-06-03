package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"monika/pkg/engine"
)

func init() {
	engine.Register(&MCPEngine{})
}

type MCPEngine struct {
	mu          sync.RWMutex
	connections map[string]mcpConn
}

type mcpConn interface {
	engine.MCPServerConnection
	close() error
}

func (e *MCPEngine) ID() string { return "mcp" }

func (e *MCPEngine) NewInstance() engine.Engine { return &MCPEngine{} }

func (e *MCPEngine) Capabilities() []engine.Capability {
	return []engine.Capability{engine.CapMCP}
}

func (e *MCPEngine) Init(_ context.Context, _ map[string]any) error {
	if e.connections == nil {
		e.connections = make(map[string]mcpConn)
	}
	return nil
}

func (e *MCPEngine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id := range e.connections {
		_ = e.disconnectLocked(id)
	}
	return nil
}

func (e *MCPEngine) ConnectServer(ctx context.Context, config engine.MCPServerConfig) (engine.MCPServerConnection, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.connections[config.ID]; exists {
		return nil, fmt.Errorf("mcp: server %q already connected", config.ID)
	}

	if config.Type == "http" || config.Type == "sse" {
		conn, err := newHTTPConnection(ctx, config)
		if err != nil {
			return nil, err
		}
		e.connections[config.ID] = conn
		return conn, nil
	}

	cmd := exec.CommandContext(ctx, config.Command, config.Args...)
	hideWindow(cmd)
	cmd.Env = os.Environ()
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp: start server: %w", err)
	}

	conn := &serverConnection{
		id:     config.ID,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		enc:    json.NewEncoder(stdin),
		dec:    json.NewDecoder(stdout),
		stderr: &stderr,
	}

	// MCP protocol requires initialization handshake.
	if err := conn.initialize(); err != nil {
		stderrStr := stderr.String()
		_ = conn.close()
		if stderrStr != "" {
			return nil, fmt.Errorf("mcp: initialize: %w\nserver stderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("mcp: initialize: %w", err)
	}

	e.connections[config.ID] = conn
	return conn, nil
}

func (e *MCPEngine) DisconnectServer(_ context.Context, serverID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.disconnectLocked(serverID)
}

func (e *MCPEngine) IsConnected(serverID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.connections[serverID]
	return ok
}

func (e *MCPEngine) disconnectLocked(serverID string) error {
	conn, ok := e.connections[serverID]
	if !ok {
		return fmt.Errorf("mcp: server %q not found", serverID)
	}
	delete(e.connections, serverID)
	return conn.close()
}

type serverConnection struct {
	mu     sync.Mutex
	id     string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *bytes.Buffer
	enc    *json.Encoder
	dec    *json.Decoder
	nextID int
	meta   engine.MCPServerMeta
}

func (c *serverConnection) ServerMeta() engine.MCPServerMeta {
	return c.meta
}

func (c *serverConnection) ListTools(ctx context.Context) ([]engine.MCPTool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  "tools/list",
	}
	if err := c.enc.Encode(req); err != nil {
		return nil, err
	}

	var resp toolsListResponse
	if err := c.dec.Decode(&resp); err != nil {
		return nil, err
	}

	return convertRawTools(resp.Result.Tools), nil
}

func (c *serverConnection) CallTool(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	params, _ := json.Marshal(map[string]any{
		"name":      name,
		"arguments": json.RawMessage(args),
	})
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  "tools/call",
		Params:  params,
	}
	if err := c.enc.Encode(req); err != nil {
		return nil, err
	}

	var resp callToolResponse
	if err := c.dec.Decode(&resp); err != nil {
		return nil, err
	}
	return resp.Result.Content, nil
}

func (c *serverConnection) close() error {
	_ = c.stdin.Close()
	_ = c.stdout.Close()
	return c.cmd.Process.Kill()
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// initializeResponse represents the JSON-RPC response from an MCP initialize call.
type initializeResponse struct {
	Result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		Instructions string `json:"instructions"`
	} `json:"result"`
}

func (c *serverConnection) initialize() error {
	c.nextID++
	initParams, _ := json.Marshal(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "monika",
			"version": "1.0",
		},
	})
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  "initialize",
		Params:  initParams,
	}
	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("send initialize: %w", err)
	}

	var resp initializeResponse
	if err := c.dec.Decode(&resp); err != nil {
		return fmt.Errorf("read initialize response: %w", err)
	}

	// Store server metadata.
	c.meta = engine.MCPServerMeta{
		ID:           c.id,
		Name:         resp.Result.ServerInfo.Name,
		Version:      resp.Result.ServerInfo.Version,
		Instructions: resp.Result.Instructions,
	}

	// Send initialized notification (no ID)
	notif, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	if err := c.enc.Encode(json.RawMessage(notif)); err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}

	return nil
}

// rawMCPTool captures all fields from the MCP tools/list response.
type rawMCPTool struct {
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Annotations *struct {
		ReadOnly    bool `json:"readOnlyHint"`
		Destructive bool `json:"destructiveHint"`
		Idempotent  bool `json:"idempotentHint"`
		OpenWorld   bool `json:"openWorldHint"`
	} `json:"annotations"`
}

func convertRawTools(raw []rawMCPTool) []engine.MCPTool {
	out := make([]engine.MCPTool, len(raw))
	for i, r := range raw {
		t := engine.MCPTool{
			Name:        r.Name,
			Title:       r.Title,
			Description: r.Description,
			InputSchema: r.InputSchema,
		}
		if r.Annotations != nil {
			t.Annotations = engine.MCPAnnotations{
				ReadOnly:    r.Annotations.ReadOnly,
				Destructive: r.Annotations.Destructive,
				Idempotent:  r.Annotations.Idempotent,
				OpenWorld:   r.Annotations.OpenWorld,
			}
		}
		out[i] = t
	}
	return out
}

type toolsListResponse struct {
	Result struct {
		Tools []rawMCPTool `json:"tools"`
	} `json:"result"`
}

type callToolResponse struct {
	Result struct {
		Content json.RawMessage `json:"content"`
	} `json:"result"`
}
