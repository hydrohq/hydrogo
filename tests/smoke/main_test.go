//go:build smoke

package smoke

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binary string

// TestMain compiles the server once; every test starts its own copy of it.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "hydro-smoke-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	binary = filepath.Join(dir, "hydro")
	build := exec.Command("go", "build", "-o", binary, "./cmd/hydro")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "go build ./cmd/hydro: %v\n%s", err, out)
		os.Exit(2)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
