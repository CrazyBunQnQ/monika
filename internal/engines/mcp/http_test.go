package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"monika/pkg/engine"
)

func TestHTTPMCPListTools(t *testing.T) {
	config := engine.MCPServerConfig{
		ID:   "web-search-prime",
		Type: "http",
		URL:  "https://open.bigmodel.cn/api/mcp/web_search_prime/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer 6bc34741204746f49fca49e664c6588d.RLUPsP3qhFZvkAas",
		},
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: initialize
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]string{
				"name":    "monika",
				"version": "1.0",
			},
		},
	}
	initBody, _ := json.Marshal(initReq)

	t.Logf("=== Step 1: initialize ===")
	result, sessionID, err := doPostRPC(client, config, initBody, 1)
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	t.Logf("session ID: %s", sessionID)
	t.Logf("initialize result: %s", truncate(string(result), 500))

	// Step 2: send initialized notification (with session ID)
	notif := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notifBody, _ := json.Marshal(notif)

	t.Logf("=== Step 2: notifications/initialized ===")
	httpReq, _ := http.NewRequest("POST", config.URL, strings.NewReader(string(notifBody)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range config.Headers {
		httpReq.Header.Set(k, v)
	}
	if sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", sessionID)
	}
	notifResp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("initialized notification failed: %v", err)
	}
	notifResp.Body.Close()
	t.Logf("initialized notification status: %d", notifResp.StatusCode)

	// Step 3: tools/list (with session ID)
	toolsReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}
	toolsBody, _ := json.Marshal(toolsReq)

	t.Logf("=== Step 3: tools/list ===")
	result, _, err = doPostRPC(client, config, toolsBody, 2)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	t.Logf("tools/list raw: %s", truncate(string(result), 2000))

	var toolsResp struct {
		Tools []struct {
			Name        string          `json:"name"`
			Title       string          `json:"title"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsResp); err != nil {
		t.Fatalf("parse tools: %v (raw: %s)", err, truncate(string(result), 500))
	}
	t.Logf("found %d tools:", len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		t.Logf("  - %s: %s", tool.Name, truncate(tool.Description, 120))
	}
}

// doPostRPC does a POST and handles both direct JSON and SSE responses,
// matching the behavior of httpConnection.postRPC.
func doPostRPC(client *http.Client, config engine.MCPServerConfig, body []byte, expectedID int) (json.RawMessage, string, error) {
	httpReq, err := http.NewRequest("POST", config.URL, strings.NewReader(string(body)))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range config.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("POST: %w", err)
	}
	defer resp.Body.Close()

	sid := resp.Header.Get("Mcp-Session-Id")

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		return nil, sid, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(raw), 500))
	}

	ct := resp.Header.Get("Content-Type")

	// SSE response
	if strings.Contains(ct, "text/event-stream") {
		result, err := parseSSE(raw, expectedID)
		return result, sid, err
	}

	// Direct JSON response
	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return nil, sid, fmt.Errorf("decode: %w (body: %s)", err, truncate(string(raw), 500))
	}
	if rpcResp.Error != nil {
		return nil, sid, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, sid, nil
}

func parseSSE(data []byte, expectedID int) (json.RawMessage, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var currentData string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			currentData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "" && currentData != "" {
			var rpcResp struct {
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
				ID int `json:"id"`
			}
			if err := json.Unmarshal([]byte(currentData), &rpcResp); err == nil && rpcResp.ID == expectedID {
				if rpcResp.Error != nil {
					return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
				}
				return rpcResp.Result, nil
			}
			currentData = ""
		}
	}
	return nil, fmt.Errorf("no SSE response found for request %d", expectedID)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
