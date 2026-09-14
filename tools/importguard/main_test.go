package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectHydroImports(t *testing.T) {
	root := t.TempDir()
	source := []byte(`package fixture
import (
	"fmt"
	"github.com/ardaozyildirim/hydrogo/extension"
	"github.com/ardaozyildirim/hydrogo/internal/server"
)
var _ = fmt.Println
`)
	if err := os.WriteFile(filepath.Join(root, "fixture.go"), source, 0o600); err != nil {
		t.Fatal(err)
	}

	violations, err := inspect(root, "github.com/ardaozyildirim/hydrogo", []string{"github.com/ardaozyildirim/hydrogo/extension"})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want one", violations)
	}
}

func TestInspectModuleBoundariesRejectsConcreteAdapters(t *testing.T) {
	root := t.TempDir()
	serviceDir := filepath.Join(root, "modules", "demo", "service")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "service.go"), []byte(`package service
import "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
var _ sqlstore.Queue
`), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, err := inspectModuleBoundaries(root, "github.com/ardaozyildirim/hydrogo")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want one", violations)
	}
}

func TestInspectModuleBoundariesIgnoresAdaptersAndTests(t *testing.T) {
	root := t.TempDir()
	serviceDir := filepath.Join(root, "modules", "demo", "service")
	repositoryDir := filepath.Join(root, "modules", "demo", "repository", "sqlstore")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repositoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, source := range map[string]string{
		filepath.Join(serviceDir, "service_test.go"): `package service
import "database/sql"
var _ *sql.DB
`,
		filepath.Join(repositoryDir, "store.go"): `package sqlstore
import "database/sql"
var _ *sql.DB
`,
	} {
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	violations, err := inspectModuleBoundaries(root, "github.com/ardaozyildirim/hydrogo")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations = %v", violations)
	}
}
