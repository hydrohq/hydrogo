package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestMCPCmdConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := mcpCommand([]string{"config"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("mcpCommand config: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	servers, ok := parsed["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("missing mcpServers: %v", parsed)
	}
	hydro, ok := servers["hydrogo"].(map[string]any)
	if !ok {
		t.Fatalf("missing hydrogo server config: %v", servers)
	}
	if hydro["command"] == "" {
		t.Errorf("empty command")
	}
}

func TestMCPCmdStdio(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Open(context.Background(), database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	cfgContent := `
database:
  dialect: sqlite
  dsn: ` + dbPath + `
content_types_dir: ` + filepath.Join(tempDir, "content-types") + `
`
	cfgPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "content-types"), 0o750); err != nil {
		t.Fatal(err)
	}

	stdin := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\"}}\n")
	var stdout, stderr bytes.Buffer

	err = mcpCommand([]string{"--config", cfgPath}, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("mcpCommand: %v", err)
	}

	outStr := strings.TrimSpace(stdout.String())
	var resp map[string]any
	if err := json.Unmarshal([]byte(outStr), &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", outStr, err)
	}
	if resp["id"] != float64(1) {
		t.Errorf("expected id 1, got %v", resp["id"])
	}
}
