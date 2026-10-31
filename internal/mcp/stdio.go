package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type stdioResponseWriter struct {
	header http.Header
	out    io.Writer
	mu     *sync.Mutex
	status int
}

func (s *stdioResponseWriter) Header() http.Header {
	if s.header == nil {
		s.header = make(http.Header)
	}
	return s.header
}

func (s *stdioResponseWriter) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out.Write(b)
}

func (s *stdioResponseWriter) WriteHeader(statusCode int) {
	s.status = statusCode
}

// ServeStdio executes the Model Context Protocol loop over standard input and output.
// Input messages are line-delimited JSON-RPC 2.0 objects.
// Responses are line-delimited JSON-RPC 2.0 objects written to out.
func (h *Handler) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 4*1024*1024) // up to 4MB per message

	var mu sync.Mutex

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var check struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(line, &check)

		// If it's a notification without an ID (e.g. notifications/initialized),
		// clients like Claude Desktop do not expect a response.
		if check.ID == nil && check.Method == "notifications/initialized" {
			continue
		}

		w := &stdioResponseWriter{
			out: out,
			mu:  &mu,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/mcp", bytes.NewReader(line))
		if err != nil {
			return fmt.Errorf("create mcp stdio request: %w", err)
		}

		h.handlePost(w, req)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	return nil
}
