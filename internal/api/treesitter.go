package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TSRequest is the payload sent from Go → frontend via Wails event.
type TSRequest struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

type tsResult struct {
	OK   bool            `json:"ok"`
	Data json.RawMessage `json:"data,omitempty"`
	Err  string          `json:"error,omitempty"`
}

// TSQueryFunc is the function signature tools use to call tree-sitter.
// Returns (nil, nil) when tree-sitter is unavailable (graceful fallback).
type TSQueryFunc func(ctx context.Context, method string, params map[string]any) (json.RawMessage, error)

type tsBridge struct {
	mu       sync.Mutex
	pending  map[string]chan tsResult
	timeout  time.Duration
	app      *App
}

// NewTSBridge creates a standalone tsBridge (no App dependency needed for QueryFunc).
func NewTSBridge() *tsBridge {
	return &tsBridge{
		pending: make(map[string]chan tsResult),
		timeout: 10 * time.Second,
	}
}

// SetApp wires the bridge to an App for TSResponse callback routing.
func (b *tsBridge) SetApp(app *App) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.app = app
}

func (b *tsBridge) QueryFunc() TSQueryFunc {
	return func(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
		id := fmt.Sprintf("ts-%d", time.Now().UnixNano())
		ch := make(chan tsResult, 1)

		b.mu.Lock()
		b.pending[id] = ch
		b.mu.Unlock()

		defer func() {
			b.mu.Lock()
			delete(b.pending, id)
			b.mu.Unlock()
		}()

		req := TSRequest{ID: id, Method: method, Params: params}
		application.Get().Event.Emit("ts:request", req)

		select {
		case res := <-ch:
			if !res.OK {
				return nil, fmt.Errorf("tree-sitter: %s", res.Err)
			}
			return res.Data, nil
		case <-time.After(b.timeout):
			return nil, fmt.Errorf("tree-sitter: request timed out (%s)", method)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// InitTSBridge sets the tsBridge on the App and wires the back-reference.
func (a *App) InitTSBridge(b *tsBridge) {
	a.tsBridge = b
	b.SetApp(a)
}

// GetTSQueryFunc returns the TSQueryFunc for tools to call tree-sitter.
func (a *App) GetTSQueryFunc() TSQueryFunc {
	if a.tsBridge == nil {
		return nil
	}
	return a.tsBridge.QueryFunc()
}

// TSResponse is called by the frontend via Call.ByName after processing a ts:request.
func (a *App) TSResponse(args json.RawMessage) error {
	var resp struct {
		ID  string          `json:"id"`
		OK  bool            `json:"ok"`
		Data json.RawMessage `json:"data,omitempty"`
		Error string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(args, &resp); err != nil {
		return err
	}

	a.tsBridge.mu.Lock()
	ch, ok := a.tsBridge.pending[resp.ID]
	if ok {
		delete(a.tsBridge.pending, resp.ID)
	}
	a.tsBridge.mu.Unlock()

	if ok {
		ch <- tsResult{OK: resp.OK, Data: resp.Data, Err: resp.Error}
	}
	return nil
}
