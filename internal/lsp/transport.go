package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

const headerSep = "\r\n\r\n"

type Transport struct {
	stdin  io.WriteCloser
	stdout *bufio.Reader
	wmu    sync.Mutex
}

func NewTransport(stdin io.WriteCloser, stdout io.Reader) *Transport {
	return &Transport{
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
}

func (t *Transport) WriteMessage(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("lsp transport marshal: %w", err)
	}
	t.wmu.Lock()
	defer t.wmu.Unlock()
	header := fmt.Sprintf("Content-Length: %d%s", len(data), headerSep)
	if _, err := io.WriteString(t.stdin, header); err != nil {
		return fmt.Errorf("lsp transport write header: %w", err)
	}
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("lsp transport write body: %w", err)
	}
	return nil
}

func (t *Transport) ReadMessage() (json.RawMessage, error) {
	var contentLength int
	for {
		line, err := t.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("lsp transport read header: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("lsp transport parse content-length: %w", err)
			}
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("lsp transport: missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(t.stdout, body); err != nil {
		return nil, fmt.Errorf("lsp transport read body: %w", err)
	}
	return json.RawMessage(body), nil
}

func (t *Transport) Close() error {
	return t.stdin.Close()
}
