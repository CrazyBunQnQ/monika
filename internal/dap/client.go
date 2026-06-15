package dap

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ---------------------------------------------------------------------------
// DapClient
// ---------------------------------------------------------------------------

// DapClient communicates with a DAP debug adapter over stdin/stdout using the
// Content-Length framing protocol.
type DapClient struct {
	adapter         *DapResolvedAdapter
	cwd             string
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          io.ReadCloser
	requestSeq      int
	mu              sync.Mutex
	pendingRequests map[int]*DapPendingRequest
	eventHandlers   map[DapEventType][]DapEventHandler
	disposed        bool
	lastActivity    time.Time
	capabilities    *DapCapabilities
}

// SpawnDapClient creates a new DAP client, starts the debug adapter process,
// and launches the message read-loop goroutine.
func SpawnDapClient(adapter *DapResolvedAdapter, cwd string) (*DapClient, error) {
	cmd := exec.Command(adapter.ResolvedCommand, adapter.Args...)
	cmd.Dir = cwd

	// Hide the cmd window on Windows
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	

	if len(adapter.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range adapter.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("dap: create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("dap: create stdout pipe: %w", err)
	}

	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("dap: start adapter: %w", err)
	}
	c := &DapClient{
		adapter:         adapter,
		cwd:             cwd,
		cmd:             cmd,
		stdin:           stdin,
		stdout:          stdout,
		requestSeq:      1,
		pendingRequests: make(map[int]*DapPendingRequest),
		eventHandlers:   make(map[DapEventType][]DapEventHandler),
		lastActivity:    time.Now(),
	}

	go c.readLoop()

	return c, nil
}

// ---------------------------------------------------------------------------
// SendRequest
// ---------------------------------------------------------------------------

// SendRequest sends a DAP request with the given command and optional
// arguments, and waits for the response or timeout.  Returns the body of the
// response on success.
func (c *DapClient) SendRequest(command string, args interface{}, timeout time.Duration) (interface{}, error) {
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}

	seq := c.nextSeq()

	// Build the arguments JSON.
	var argsRaw json.RawMessage
	if args != nil {
		data, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("dap: marshal %q args: %w", command, err)
		}
		argsRaw = data
	}

	req := DapRequestMessage{
		DapProtocolMessage: DapProtocolMessage{
			Seq:  seq,
			Type: "request",
		},
		Command:   command,
		Arguments: argsRaw,
	}

	// Wire up the pending request channel.
	pending := &DapPendingRequest{
		Command: command,
		Resolve: nil, // set below
		Reject:  nil,
	}

	// Use channels so we can select with timeout.
	respCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)

	pending.Resolve = func(body interface{}) {
		respCh <- body
	}
	pending.Reject = func(err error) {
		errCh <- err
	}

	c.mu.Lock()
	c.pendingRequests[seq] = pending
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pendingRequests, seq)
		c.mu.Unlock()
	}()

	// Write the frame.
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("dap: marshal request: %w", err)
	}

	frame := formatFrame(data)
	if _, err := c.stdin.Write(frame); err != nil {
		return nil, fmt.Errorf("dap: write request: %w", err)
	}

	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()

	// Wait for response or timeout.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case body := <-respCh:
		return body, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("dap: request %q (seq=%d) timed out after %v", command, seq, timeout)
	}
}

// Initialize sends the "initialize" request, unmarshals the capabilities from
// the response body, and stores them on the client.
func (c *DapClient) Initialize(args DapInitializeArguments, timeout time.Duration) (*DapCapabilities, error) {
	body, err := c.SendRequest("initialize", args, timeout)
	if err != nil {
		return nil, err
	}

	data, ok := body.(json.RawMessage)
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("dap: initialize response has no body")
	}

	var caps DapCapabilities
	if err := json.Unmarshal(data, &caps); err != nil {
		return nil, fmt.Errorf("dap: unmarshal capabilities: %w", err)
	}

	c.mu.Lock()
	c.capabilities = &caps
	c.mu.Unlock()

	return &caps, nil
}

// OnEvent registers a handler for the given DAP event type.
func (c *DapClient) OnEvent(eventType DapEventType, handler DapEventHandler) {
	c.mu.Lock()
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)
	c.mu.Unlock()
}

// Capabilities returns the capabilities received from the adapter after a
// successful Initialize call.
func (c *DapClient) Capabilities() *DapCapabilities {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.capabilities
}

// IsAlive reports whether the client has not been disposed and the adapter
// process is still running.
func (c *DapClient) IsAlive() bool {
	c.mu.Lock()
	disposed := c.disposed
	c.mu.Unlock()

	if disposed {
		return false
	}

	return c.cmd != nil && c.cmd.Process != nil && c.cmd.ProcessState == nil
}

// Dispose marks the client as disposed, rejects all pending requests, and
// kills the adapter process.
func (c *DapClient) Dispose() {
	c.mu.Lock()
	if c.disposed {
		c.mu.Unlock()
		return
	}
	c.disposed = true

	// Reject all pending requests.
	err := fmt.Errorf("dap: client disposed")
	for seq, pending := range c.pendingRequests {
		if pending.Reject != nil {
			pending.Reject(err)
		}
		delete(c.pendingRequests, seq)
	}
	// Clear event handlers.
	for k := range c.eventHandlers {
		delete(c.eventHandlers, k)
	}
	stdin := c.stdin
	stdout := c.stdout
	cmd := c.cmd
	c.mu.Unlock()

	// Close pipes.
	if stdin != nil {
		stdin.Close()
	}
	if stdout != nil {
		stdout.Close()
	}

	// Kill the process.
	if cmd != nil && cmd.Process != nil && cmd.ProcessState == nil {
		cmd.Process.Kill()
		// Wait to avoid zombies.
		go cmd.Wait()
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// nextSeq returns the next request sequence number.
func (c *DapClient) nextSeq() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	seq := c.requestSeq
	c.requestSeq++
	return seq
}

// readLoop reads from stdout, parses Content-Length framed messages, and
// dispatches them via processMessage.
func (c *DapClient) readLoop() {
	reader := bufio.NewReader(c.stdout)

	for {
		content, err := readFrame(reader)
		if err != nil {
			// If the client is disposed or the process exited, that's
			// expected — return silently.
			c.mu.Lock()
			disposed := c.disposed
			c.mu.Unlock()

			if !disposed {
				c.mu.Lock()
				for _, pending := range c.pendingRequests {
					if pending.Reject != nil {
						pending.Reject(fmt.Errorf("dap: read error: %w", err))
					}
				}
				c.pendingRequests = make(map[int]*DapPendingRequest)
				c.mu.Unlock()
			}
			return
		}

		c.processMessage(content)

		c.mu.Lock()
		c.lastActivity = time.Now()
		c.mu.Unlock()
	}
}

// readFrame reads one Content-Length framed message from r.  It parses
// headers until an empty line, then reads exactly N bytes of JSON body.
func readFrame(r *bufio.Reader) ([]byte, error) {
	var contentLength int

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}
		line = trimCRLF(line)

		if line == "" {
			// End of headers.
			break
		}

		if _, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength); err == nil {
			continue
		}
		// Ignore unknown headers (e.g. Content-Type).
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing or invalid Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

// processMessage unmarshals a JSON message and dispatches it to the
// appropriate pending request or event handler.
func (c *DapClient) processMessage(data []byte) {
	// Peek at the type field to decide how to unmarshal.
	var msg DapProtocolMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		// Malformed data — ignore.
		return
	}

	switch msg.Type {
	case "response":
		var resp DapResponseMessage
		if err := json.Unmarshal(data, &resp); err != nil {
			return
		}
		c.dispatchResponse(&resp)

	case "event":
		var evt DapEventMessage
		if err := json.Unmarshal(data, &evt); err != nil {
			return
		}
		c.dispatchEvent(&evt)

	default:
		// Unknown message type — ignore.
	}
}

// dispatchResponse resolves the pending request matching the response's
// request_seq.
func (c *DapClient) dispatchResponse(resp *DapResponseMessage) {
	c.mu.Lock()
	pending, ok := c.pendingRequests[resp.RequestSeq]
	if !ok {
		c.mu.Unlock()
		return
	}
	delete(c.pendingRequests, resp.RequestSeq)
	c.mu.Unlock()

	if !resp.Success {
		errMsg := resp.Message
		if errMsg == "" {
			errMsg = fmt.Sprintf("adapter error for %q", resp.Command)
		}
		pending.Reject(fmt.Errorf("dap: %s: %s", resp.Command, errMsg))
		return
	}

	pending.Resolve(resp.Body)
}

// dispatchEvent notifies all registered handlers for the given event type.
func (c *DapClient) dispatchEvent(evt *DapEventMessage) {
	eventType := DapEventType(evt.Event)

	c.mu.Lock()
	handlers := c.eventHandlers[eventType]
	// Also notify handlers registered for the "all events" sentinel.
	allHandlers := c.eventHandlers[""]
	c.mu.Unlock()

	dispatch := func(h DapEventHandler) {
		h(evt.Body, evt)
	}

	for _, h := range handlers {
		dispatch(h)
	}
	for _, h := range allHandlers {
		dispatch(h)
	}
}

// ---------------------------------------------------------------------------
// Framing helpers
// ---------------------------------------------------------------------------

// formatFrame prepends the Content-Length header to JSON data.
func formatFrame(data []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(data))
	buf.Write(data)
	return buf.Bytes()
}

// trimCRLF removes a trailing \r or \r\n from s.
func trimCRLF(s string) string {
	if len(s) >= 2 && s[len(s)-2] == '\r' && s[len(s)-1] == '\n' {
		return s[:len(s)-2]
	}
	if len(s) >= 1 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}
