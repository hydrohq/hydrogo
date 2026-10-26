//go:build smoke

package smoke

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	name := "hydro"
	if runtime.GOOS == "windows" {
		// exec.LookPath cannot resolve an extensionless file on Windows, even
		// given its full path.
		name += ".exe"
	}
	binary = filepath.Join(dir, name)
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
