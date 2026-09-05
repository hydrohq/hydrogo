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
