package dbbridge

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed scripts/*
var scripts embed.FS

type BridgeManager struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      *json.Encoder
	stdout     *bufio.Scanner
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrBuf  bytes.Buffer
	cancelKeep context.CancelFunc
	projectDir string
	runtime    string
	scriptPath string
	running    atomic.Bool
	reqID      atomic.Int64
	maxRetries int
	retries    int
}

func NewBridgeManager() *BridgeManager {
	return &BridgeManager{
		maxRetries: 3,
	}
}

func (b *BridgeManager) Start(ctx context.Context, projectDir, runtime string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running.Load() {
		return nil
	}

	b.projectDir = projectDir
	b.runtime = runtime
	b.retries = 0

	return b.startLocked(ctx)
}

func (b *BridgeManager) startLocked(ctx context.Context) error {
	bin, err := b.findBinary(b.runtime)
	if err != nil {
		return fmt.Errorf("dbbridge: find binary: %w", err)
	}

	script, err := b.extractScript()
	if err != nil {
		return fmt.Errorf("dbbridge: extract script: %w", err)
	}
	b.scriptPath = script

	b.cmd = exec.CommandContext(ctx, bin, script)
	b.cmd.Dir = b.projectDir

	stdinPipe, err := b.cmd.StdinPipe()
	if err != nil {
		os.Remove(script)
		return fmt.Errorf("dbbridge: stdin pipe: %w", err)
	}

	stdoutPipe, err := b.cmd.StdoutPipe()
	if err != nil {
		stdinPipe.Close()
		os.Remove(script)
		return fmt.Errorf("dbbridge: stdout pipe: %w", err)
	}

	b.stderrBuf.Reset()
	b.cmd.Stderr = &b.stderrBuf

	if err := b.cmd.Start(); err != nil {
		stdinPipe.Close()
		stdoutPipe.Close()
		os.Remove(script)
		if stderr := b.stderrBuf.String(); stderr != "" {
			return fmt.Errorf("dbbridge: start process: %w (stderr: %s)", err, stderr)
		}
		return fmt.Errorf("dbbridge: start process: %w", err)
	}

	b.stdin = json.NewEncoder(stdinPipe)
	b.stdout = bufio.NewScanner(stdoutPipe)
	b.stdinPipe = stdinPipe
	b.stdoutPipe = stdoutPipe
	b.stdout.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	b.running.Store(true)

	keepCtx, cancel := context.WithCancel(context.Background())
	b.cancelKeep = cancel
	go b.keepalive(keepCtx)

	return nil
}

// Send sends a request and waits for the response.
// Note: concurrent calls are serialized because the bridge protocol
// uses a single stdin/stdout pipe pair. This is acceptable since
// db queries are typically sequential in the agent loop.
func (b *BridgeManager) Send(req Request) (Response, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running.Load() {
		return Response{}, fmt.Errorf("dbbridge: bridge not running")
	}

	if err := b.stdin.Encode(req); err != nil {
		b.running.Store(false)
		return Response{}, fmt.Errorf("dbbridge: send: %w", err)
	}

	if !b.stdout.Scan() {
		b.running.Store(false)
		return Response{}, fmt.Errorf("dbbridge: connection closed")
	}

	var resp Response
	if err := json.Unmarshal(b.stdout.Bytes(), &resp); err != nil {
		return Response{}, fmt.Errorf("dbbridge: decode response: %w", err)
	}

	return resp, nil
}

func (b *BridgeManager) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stopLocked()
}

func (b *BridgeManager) stopLocked() {
	if b.cancelKeep != nil {
		b.cancelKeep()
		b.cancelKeep = nil
	}

	if b.stdinPipe != nil {
		b.stdinPipe.Close()
		b.stdinPipe = nil
	}
	if b.stdoutPipe != nil {
		b.stdoutPipe.Close()
		b.stdoutPipe = nil
	}

	if b.cmd != nil && b.cmd.Process != nil {
		b.cmd.Process.Kill()
		b.cmd.Wait()
	}

	b.running.Store(false)

	if b.scriptPath != "" {
		os.Remove(b.scriptPath)
		b.scriptPath = ""
	}
}

func (b *BridgeManager) IsRunning() bool {
	return b.running.Load()
}

func (b *BridgeManager) NextID() int {
	return int(b.reqID.Add(1))
}

func (b *BridgeManager) findBinary(runtime string) (string, error) {
	switch runtime {
	case "node":
		if p, err := exec.LookPath("node"); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("node not found in PATH")
	case "python":
		if p, err := exec.LookPath("python3"); err == nil {
			return p, nil
		}
		if p, err := exec.LookPath("python"); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("python3/python not found in PATH")
	default:
		return "", fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

func (b *BridgeManager) extractScript() (string, error) {
	entries, err := fs.ReadDir(scripts, "scripts")
	if err != nil {
		return "", err
	}

	var target string
	switch b.runtime {
	case "node":
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".mjs" || filepath.Ext(e.Name()) == ".js" {
				target = e.Name()
				break
			}
		}
	case "python":
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".py" {
				target = e.Name()
				break
			}
		}
	}

	if target == "" {
		return "", fmt.Errorf("no bridge script found for runtime %s", b.runtime)
	}

	data, err := fs.ReadFile(scripts, "scripts/"+target)
	if err != nil {
		return "", err
	}

	tmp, err := os.CreateTemp("", "dbbridge-*-"+target)
	if err != nil {
		return "", err
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

func (b *BridgeManager) keepalive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !b.running.Load() {
				return
			}

			id := int(b.reqID.Add(1))
			req := Request{ID: id, Action: "ping"}

			b.mu.Lock()
			if !b.running.Load() {
				b.mu.Unlock()
				return
			}

			if err := b.stdin.Encode(req); err != nil {
				b.mu.Unlock()
				log.Printf("[dbbridge] keepalive send failed: %v", err)
				b.handleCrash(ctx)
				return
			}

			if !b.stdout.Scan() {
				b.mu.Unlock()
				log.Printf("[dbbridge] keepalive read failed")
				b.handleCrash(ctx)
				return
			}
			b.mu.Unlock()
		}
	}
}

func (b *BridgeManager) handleCrash(ctx context.Context) {
	b.mu.Lock()
	b.stopLocked()

	if b.retries >= b.maxRetries {
		b.mu.Unlock()
		log.Printf("[dbbridge] max retries (%d) reached, giving up", b.maxRetries)
		return
	}

	b.retries++
	log.Printf("[dbbridge] restarting bridge (attempt %d/%d)", b.retries, b.maxRetries)

	if err := b.startLocked(ctx); err != nil {
		b.mu.Unlock()
		log.Printf("[dbbridge] restart failed: %v", err)
		return
	}
	b.mu.Unlock()
}
