package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/config"
)

func TestValidateProductionOriginRequiresPublicOriginInProduction(t *testing.T) {
	t.Setenv("HYDRO_ENV", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("GO_ENV", "")

	empty := config.Config{}
	if err := validateProductionOrigin(empty); err != nil {
		t.Fatalf("dev/unset environment with no public_origin must not fail: %v", err)
	}

	t.Setenv("HYDRO_ENV", "production")
	if err := validateProductionOrigin(empty); err == nil {
		t.Fatal("production with an empty public_origin must be flagged")
	}

	withOrigin := config.Config{}
	withOrigin.HTTP.PublicOrigin = "https://shop.example.test"
	if err := validateProductionOrigin(withOrigin); err != nil {
		t.Fatalf("production with public_origin set: %v", err)
	}

	t.Setenv("HYDRO_ENV", "")
	t.Setenv("APP_ENV", "prod")
	if err := validateProductionOrigin(empty); err == nil {
		t.Fatal("APP_ENV=prod must also be treated as production")
	}
	_ = os.Unsetenv("APP_ENV")
}

func TestDoctorValidatesContentTypesAndDetectsPortConflict(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "doctor.sqlite")
	mediaDir := filepath.Join(tmp, "media")
	contentTypesDir := filepath.Join(tmp, "content-types")
	if err := os.MkdirAll(contentTypesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Run migrate first so DB schema is ready
	var mOut, mErr bytes.Buffer
	if err := run([]string{"migrate", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &mOut, &mErr); err != nil {
		t.Fatalf("migrate failed: %v, stderr: %s", err, mErr.String())
	}

	// 2. Put a valid content type
	validYAML := `
id: book
label: Book
fields:
  - id: title
    label: Title
    kind: text
`
	if err := os.WriteFile(filepath.Join(contentTypesDir, "book.yaml"), []byte(validYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Create a config file pointing to dirs
	configPath := filepath.Join(tmp, "hydro.yaml")
	cfgContent := fmt.Sprintf(`
profile: compact
content_types_dir: %q
media:
  dir: %q
database:
  dialect: sqlite
  dsn: %q
`, contentTypesDir, mediaDir, dbPath)
	if err := os.WriteFile(configPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. Bind a dummy listener to simulate port in use
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	boundAddr := ln.Addr().String()

	var stdout, stderr bytes.Buffer
	args := []string{
		"doctor",
		"--config", configPath,
		"--address", boundAddr,
		"--json",
	}

	if err := run(args, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor failed: %v, stderr: %s", err, stderr.String())
	}

	var report struct {
		Database          string   `json:"database"`
		ContentTypesCount int      `json:"content_types_count"`
		PortAvailable     bool     `json:"port_available"`
		Warnings          []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json unmarshal failed: %v, raw: %s", err, stdout.String())
	}
	if report.ContentTypesCount != 1 {
		t.Errorf("expected 1 content type, got %d", report.ContentTypesCount)
	}
	if report.PortAvailable {
		t.Errorf("expected port_available = false for bound address %s", boundAddr)
	}
	hasPortWarning := false
	for _, w := range report.Warnings {
		if strings.Contains(w, "cannot be bound") {
			hasPortWarning = true
			break
		}
	}
	if !hasPortWarning {
		t.Errorf("expected warning about port bindability, got: %v", report.Warnings)
	}

	// 4. Test invalid content type schema fails loudly
	invalidYAML := `
id: Invalid-ID-Caps
label: Invalid
fields:
  - id: id
    label: ID
    type: string
`
	if err := os.WriteFile(filepath.Join(contentTypesDir, "invalid.yaml"), []byte(invalidYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run(args, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("expected doctor to fail with invalid content type schema")
	} else if !strings.Contains(err.Error(), "content types schema validation") {
		t.Fatalf("expected content types schema validation error, got: %v", err)
	}
}
