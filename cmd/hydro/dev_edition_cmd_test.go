package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/projectgen"
)

// discardWriter satisfies io.Writer without importing io/ioutil; the
// subprocess output isn't asserted on, only the HTTP-visible behavior is.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// pickLocalAddress reserves and immediately releases an ephemeral loopback
// port so runEditionDevContext can bind to a predictable, free address.
func pickLocalAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("loopback listener unavailable in this environment: %v", err)
		}
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

func waitForHTTP200(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			ok := resp.StatusCode == http.StatusOK
			_ = resp.Body.Close()
			if ok {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func waitForStatusMessageContains(proxyAddr, substring string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://" + proxyAddr + editionStatusPath)
		if err == nil {
			var payload struct {
				Message string `json:"message"`
			}
			if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr == nil && strings.Contains(payload.Message, substring) {
				_ = resp.Body.Close()
				return true
			}
			_ = resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func appendToFile(t *testing.T, path, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func TestIsCoreCheckoutRequiresGoModAndDevserverDir(t *testing.T) {
	root := t.TempDir()
	if isCoreCheckout(root) {
		t.Fatal("empty directory must not be treated as a Core checkout")
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isCoreCheckout(root) {
		t.Fatal("go.mod alone must not be enough to detect a Core checkout")
	}
	if err := os.MkdirAll(filepath.Join(root, "tools", "devserver"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !isCoreCheckout(root) {
		t.Fatal("go.mod + tools/devserver must be detected as a Core checkout")
	}
}

func TestIsClientEditionRequiresLockAndConfig(t *testing.T) {
	root := t.TempDir()
	if isClientEdition(root) {
		t.Fatal("empty directory must not be treated as a Client Edition project")
	}
	if err := os.WriteFile(filepath.Join(root, "hydrogo.lock"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isClientEdition(root) {
		t.Fatal("hydrogo.lock alone must not be enough to detect a Client Edition project")
	}
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("profile: compact\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !isClientEdition(root) {
		t.Fatal("hydrogo.lock + config.yaml must be detected as a Client Edition project")
	}
}

func TestDetectDevModePrefersCoreCheckoutWhenBothSignalsArePresent(t *testing.T) {
	// The HydroGo Core repository itself carries a root-level config.yaml
	// and hydrogo.lock (used for its own local dev/testing), so
	// isCoreCheckout and isClientEdition are not mutually exclusive in
	// practice. detectDevMode must still pick devModeCore here: go.mod +
	// tools/devserver wins over hydrogo.lock + config.yaml, so existing
	// Core-repo `hydro dev` workflows never change.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "..", "..")
	if !isCoreCheckout(root) {
		t.Skip("repository root does not look like a Core checkout in this environment")
	}
	if !isClientEdition(root) {
		t.Skip("repository root has no root-level config.yaml/hydrogo.lock in this environment")
	}
	if mode := detectDevMode(root); mode != devModeCore {
		t.Fatalf("detectDevMode(%q) = %v, want devModeCore even though isClientEdition is also true", root, mode)
	}
}

func TestDetectDevModeReturnsUnknownOutsideAnyRecognizedLayout(t *testing.T) {
	root := t.TempDir()
	if mode := detectDevMode(root); mode != devModeUnknown {
		t.Fatalf("detectDevMode(%q) = %v, want devModeUnknown", root, mode)
	}
}

// buildHydroBinary compiles the current cmd/hydro source tree once per test
// binary run and returns the path to the resulting executable. Building (not
// `go run`) mirrors what runEditionDev actually subprocesses in production.
func buildHydroBinary(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join(wd, "..", "..")
	name := "hydro-edition-test"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binPath, "./cmd/hydro")
	build.Dir = moduleRoot
	output, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build hydro binary: %v\n%s", err, output)
	}
	return binPath
}

// scaffoldClientEditionProject generates a Client Edition project with
// projectgen (the same code `hydro new` uses) and disables license
// enforcement in the generated config.yaml. License verification against a
// vendor-signed hydro.lic is a separate, already-tracked gap (see
// TASKS.md "Scaffold edilen Client Edition projeleri lisanssız yerel e2e
// doğrulamaya kapalı"); this test exercises the dev-mode orchestration
// (migrate/seed/restart/proxy/watch), not license enforcement.
func scaffoldClientEditionProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path, err := projectgen.Generate(projectgen.Options{
		Root:         root,
		Name:         "devedition",
		Preset:       "editorial",
		Profile:      "compact",
		Database:     "sqlite",
		Presentation: "integrated",
		Edition:      "starter",
		Demo:         true,
	})
	if err != nil {
		t.Fatalf("scaffold project: %v", err)
	}
	configPath := filepath.Join(path, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	patched := strings.Replace(string(data), "license:\n  required: true", "license:\n  required: false", 1)
	if patched == string(data) {
		t.Fatal("expected to find license.required: true in generated config.yaml")
	}
	if err := os.WriteFile(configPath, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunEditionDevServesHealthzAndReactsToThemeChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary and runs a live server; skipped in -short mode")
	}
	binary := buildHydroBinary(t)
	projectRoot := scaffoldClientEditionProject(t)

	t.Setenv(editionSelfBinaryEnv, binary)
	t.Setenv("HYDRO_REDIS_URL", "")
	t.Setenv("HYDRO_DEV_INDICATOR", "0")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	proxyAddr := pickLocalAddress(t)
	appAddr := pickLocalAddress(t)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		errCh <- runEditionDevContext(ctx, []string{
			"--config", "config.yaml",
			"--address", proxyAddr,
			"--app-address", appAddr,
			"--seed-demo=false",
			"--dev-indicator=false",
		}, &discardWriter{}, &discardWriter{})
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	healthzURL := "http://" + proxyAddr + "/healthz"
	if !waitForHTTP200(healthzURL, 30*time.Second) {
		t.Fatalf("hydro dev did not serve a healthy /healthz at %s within timeout", healthzURL)
	}

	// Modify a theme static asset: must be handled as an asset-only update
	// (no migrate/restart), per the plan's file-watch scoping.
	cssPath := filepath.Join(projectRoot, "themes", "newsroom", "static", "theme.css")
	if _, err := os.Stat(cssPath); err != nil {
		t.Fatalf("expected generated theme css at %s: %v", cssPath, err)
	}
	appendToFile(t, cssPath, "\n/* dev test asset update */\n")

	if !waitForStatusMessageContains(proxyAddr, "Asset updated", 10*time.Second) {
		resp, err := http.Get("http://" + proxyAddr + editionStatusPath)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("final status payload: %s", body)
			_ = resp.Body.Close()
		} else {
			t.Logf("status fetch failed: %v", err)
		}
		t.Fatal("expected an asset-update status message after editing themes/*/static/theme.css")
	}

	// A non-asset content change (theme layout) must trigger migrate+restart
	// and still leave /healthz reachable afterwards.
	layoutPath := filepath.Join(projectRoot, "themes", "newsroom", "layouts", "base.html")
	if _, err := os.Stat(layoutPath); err != nil {
		t.Fatalf("expected generated theme layout at %s: %v", layoutPath, err)
	}
	appendToFile(t, layoutPath, "\n<!-- dev test layout update -->\n")

	if !waitForHTTP200(healthzURL, 30*time.Second) {
		t.Fatal("hydro dev did not become healthy again after a theme layout change (migrate+restart path)")
	}

	select {
	case err := <-errCh:
		t.Fatalf("runEditionDevContext returned early: %v", err)
	default:
	}
}
