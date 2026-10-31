package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestServeStdio(t *testing.T) {
	handler := NewHandler(Config{})

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}, "\n") + "\n"

	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	ctx := context.Background()
	if err := handler.ServeStdio(ctx, in, out); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 responses (notification omitted), got %d: %v", len(lines), lines)
	}

	// 1. Check initialize response
	var initResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp.ID != float64(1) {
		t.Errorf("expected id 1, got %v", initResp.ID)
	}
	resMap, ok := initResp.Result.(map[string]any)
	if !ok || resMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected initialize result: %v", initResp.Result)
	}

	// 2. Check ping response
	var pingResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &pingResp); err != nil {
		t.Fatalf("unmarshal ping response: %v", err)
	}
	if pingResp.ID != float64(2) {
		t.Errorf("expected id 2, got %v", pingResp.ID)
	}

	// 3. Check tools/list response
	var toolsResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[2]), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	if toolsResp.ID != float64(3) {
		t.Errorf("expected id 3, got %v", toolsResp.ID)
	}
	toolsMap, ok := toolsResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected tools result: %v", toolsResp.Result)
	}
	toolsList, ok := toolsMap["tools"].([]any)
	if !ok || len(toolsList) == 0 {
		t.Fatalf("expected tools list, got: %v", toolsMap)
	}
}
