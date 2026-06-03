package engine

import (
	"context"
	"encoding/json"
	"regexp"
	"sync"
)

type MCPServerConfig struct {
	ID      string
	Type    string // "stdio" or "http"
	Command string
	Args    []string
	Env     map[string]string
	URL     string
	Headers map[string]string
}

// MCPServerMeta holds server metadata parsed from the initialize response.
type MCPServerMeta struct {
	ID           string
	Name         string // serverInfo.name
	Version      string // serverInfo.version
	Instructions string // initialize response instructions
}

// MCPAnnotations holds behavioral hints from the MCP protocol tool annotations.
type MCPAnnotations struct {
	ReadOnly    bool
	Destructive bool
	Idempotent  bool
	OpenWorld   bool
}

type MCPTool struct {
	Name        string
	Title       string          // human-readable name
	Description string
	InputSchema json.RawMessage
	ServerID    string          // owning server ID
	Annotations MCPAnnotations  // behavioral hints
}

type MCPServerConnection interface {
	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
	ServerMeta() MCPServerMeta
}

type MCPEngine interface {
	Engine
	ConnectServer(ctx context.Context, config MCPServerConfig) (MCPServerConnection, error)
	DisconnectServer(ctx context.Context, serverID string) error
	IsConnected(serverID string) bool
}

// sanitizePrefix replaces any non [a-zA-Z0-9_-] character with _.
var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func sanitizePrefix(s string) string {
	return sanitizeRe.ReplaceAllString(s, "_")
}

// MCPRegistry provides thread-safe access to MCP connections, tools, and server metadata.
// Tools are stored with prefixed names: sanitize(serverID) + "_" + sanitize(toolName).
type MCPRegistry struct {
	mu          sync.RWMutex
	servers     map[string]MCPServerMeta       // serverID → meta
	connections map[string]MCPServerConnection // serverID → connection
	tools       map[string]MCPTool             // prefixed name → MCPTool
}

func NewMCPRegistry() *MCPRegistry {
	return &MCPRegistry{
		servers:     make(map[string]MCPServerMeta),
		connections: make(map[string]MCPServerConnection),
		tools:       make(map[string]MCPTool),
	}
}

// AddServer registers a server's meta, connection, and tools.
// Tool names are automatically prefixed with the sanitized server ID.
func (r *MCPRegistry) AddServer(meta MCPServerMeta, conn MCPServerConnection, tools []MCPTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[meta.ID] = meta
	r.connections[meta.ID] = conn
	prefix := sanitizePrefix(meta.ID)
	for i := range tools {
		tools[i].ServerID = meta.ID
		prefixed := prefix + "_" + sanitizePrefix(tools[i].Name)
		r.tools[prefixed] = tools[i]
	}
}

// GetTools returns all registered tools with prefixed names.
func (r *MCPRegistry) GetTools() []MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MCPTool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// GetServers returns metadata for all registered servers.
func (r *MCPRegistry) GetServers() []MCPServerMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MCPServerMeta, 0, len(r.servers))
	for _, s := range r.servers {
		out = append(out, s)
	}
	return out
}

// GetToolsByServer returns prefixed tools belonging to a specific server.
func (r *MCPRegistry) GetToolsByServer(serverID string) []MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []MCPTool
	for _, t := range r.tools {
		if t.ServerID == serverID {
			out = append(out, t)
		}
	}
	return out
}

// GetConnection returns the connection for a given server ID.
func (r *MCPRegistry) GetConnection(serverID string) (MCPServerConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.connections[serverID]
	return c, ok
}

// ServerMeta returns metadata for a specific server.
func (r *MCPRegistry) ServerMeta(serverID string) (MCPServerMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.servers[serverID]
	return m, ok
}

// Resolve decomposes a prefixed tool name into server ID and original tool name.
func (r *MCPRegistry) Resolve(prefixedName string) (serverID string, origName string, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, exists := r.tools[prefixedName]
	if !exists {
		return "", "", false
	}
	return t.ServerID, t.Name, true
}

// LenTools returns the number of registered tools.
func (r *MCPRegistry) LenTools() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// GetConnections returns a copy of the connections map (for backward compatibility).
func (r *MCPRegistry) GetConnections() map[string]MCPServerConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]MCPServerConnection, len(r.connections))
	for k, v := range r.connections {
		out[k] = v
	}
	return out
}

// RemoveServer removes a server and its tools from the registry.
func (r *MCPRegistry) RemoveServer(serverID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.servers, serverID)
	delete(r.connections, serverID)
	for k, t := range r.tools {
		if t.ServerID == serverID {
			delete(r.tools, k)
		}
	}
}
