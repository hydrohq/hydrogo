package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// These optional modules are wired by serve; sectoral Extension types are not
// activated by adding their names to this CLI's module configuration.
var runtimeRouteOptionalModules = []string{"forms", "analytics", "search", "collections", "realtime", "push", "ai"}

// Exercise serve's real composition root, not a separately maintained list of
// routes. Both matrices re-exec this test binary: CGI constructs the runtime
// handler but finishes after one request (no listening port, no process-global
// signal handlers in the parent), while the http mode starts the standalone
// listener and must exit gracefully after SIGTERM.
func TestRuntimeRouteTableRegistration(t *testing.T) {
	if os.Getenv("HYDRO_ROUTE_TABLE_TEST_CHILD") == "cgi" {
		args := []string{"--config", "routes.json", "--database-dialect", "sqlite", "--database-dsn", "routes.sqlite"}
		if err := run(append([]string{"migrate"}, args...), strings.NewReader(""), os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
		if err := run(append([]string{"serve", "--cgi"}, args...), strings.NewReader(""), os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
		return
	}

	t.Run("default", func(t *testing.T) { testRuntimeRouteConfiguration(t, nil) })
	for _, module := range runtimeRouteOptionalModules {
		t.Run(module, func(t *testing.T) { testRuntimeRouteConfiguration(t, []string{module}) })
	}
	t.Run("all_optional", func(t *testing.T) { testRuntimeRouteConfiguration(t, runtimeRouteOptionalModules) })
}

// The standalone child is re-entered through this test; the parent below uses
// the same optional-module matrix and must observe a graceful SIGTERM exit.
func TestRuntimeStandaloneStartupAndShutdown(t *testing.T) {
	if os.Getenv("HYDRO_ROUTE_TABLE_TEST_CHILD") == "http" {
		base := []string{"--config", "routes.json", "--database-dialect", "sqlite", "--database-dsn", "routes.sqlite"}
		if err := run(append([]string{"migrate"}, base...), strings.NewReader(""), os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
		serveArgs := append([]string{"serve"}, base...)
		serveArgs = append(serveArgs, "--address", os.Getenv("HYDRO_ROUTE_TABLE_TEST_ADDRESS"))
		if err := run(serveArgs, strings.NewReader(""), os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
		return
	}

	t.Run("default", func(t *testing.T) { testRuntimeStandaloneShutdown(t, nil) })
	for _, module := range runtimeRouteOptionalModules {
		t.Run(module, func(t *testing.T) { testRuntimeStandaloneShutdown(t, []string{module}) })
	}
	t.Run("all_optional", func(t *testing.T) { testRuntimeStandaloneShutdown(t, runtimeRouteOptionalModules) })
}

func testRuntimeRouteConfiguration(t *testing.T, optional []string) {
	t.Helper()
	cmd := runtimeRouteChild(t, "cgi", optional)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("runtime route assembly failed: %v\nstdout:\n%s\nstderr:\n%s", err, &stdout, &stderr)
	}
	if !strings.Contains(stdout.String(), "Status: 200") {
		t.Fatalf("runtime did not serve health check:\n%s\nstderr:\n%s", &stdout, &stderr)
	}
}

func testRuntimeStandaloneShutdown(t *testing.T, optional []string) {
	t.Helper()
	for attempt := 0; attempt < 3; attempt++ {
		port := freeTCPPort(t)
		cmd := runtimeRouteChild(t, "http", optional)
		cmd.Env = append(cmd.Env, "HYDRO_ROUTE_TABLE_TEST_ADDRESS=127.0.0.1:"+strconv.Itoa(port))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		ready, exitErr, exitedEarly := waitRuntimeReady(t, port, exited)
		if exitedEarly {
			if isPortConflict(&stderr) && attempt < 2 {
				continue // the reserved port raced away between probe and bind
			}
			t.Fatalf("standalone runtime exited during startup: %v\nstdout:\n%s\nstderr:\n%s", exitErr, &stdout, &stderr)
		}
		if !ready {
			t.Fatalf("standalone runtime did not become ready within 15s\nstdout:\n%s\nstderr:\n%s", &stdout, &stderr)
		}
		if err := terminateRuntimeProcess(cmd); err != nil {
			t.Skipf("cannot deliver shutdown signals on this platform: %v", err)
		}
		select {
		case err := <-exited:
			if err != nil {
				t.Fatalf("standalone runtime did not shut down gracefully: %v\nstdout:\n%s\nstderr:\n%s", err, &stdout, &stderr)
			}
			if strings.Contains(stdout.String()+stderr.String(), "panic:") {
				t.Fatalf("shutdown output contained a panic\nstdout:\n%s\nstderr:\n%s", &stdout, &stderr)
			}
			return
		case <-time.After(30 * time.Second):
			_ = cmd.Process.Kill()
			t.Fatalf("standalone runtime ignored SIGTERM for 30s\nstdout:\n%s\nstderr:\n%s", &stdout, &stderr)
		}
	}
	t.Fatal("exhausted standalone startup retries")
}

// runtimeRouteChild builds the isolated self-re-exec'd child process for one
// route-registration configuration. mode selects the child branch ("cgi" or
// "http") and therefore which parent test the -test.run regex re-enters.
func runtimeRouteChild(t *testing.T, mode string, optional []string) *exec.Cmd {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	childTest := "TestRuntimeRouteTableRegistration"
	if mode == "http" {
		childTest = "TestRuntimeStandaloneStartupAndShutdown"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, executable, "-test.run=^"+childTest+"$", "-test.count=1")
	cmd.Dir = t.TempDir()
	modules := append([]string{"core", "auth", "members", "pages", "media", "site", "news"}, optional...)
	configuration, err := json.Marshal(map[string]any{"modules": modules})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmd.Dir, "routes.json"), configuration, 0o600); err != nil {
		t.Fatal(err)
	}
	// Do not inherit developer database, license, tenant, or external-service
	// settings. Defaults enable core/auth/members/pages/media/site/news together.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "HYDRO_") || strings.HasPrefix(key, "HTTP_") || strings.HasPrefix(key, "SERVER_") || key == "REQUEST_METHOD" || key == "REQUEST_URI" || key == "GATEWAY_INTERFACE" || key == "SCRIPT_NAME" || key == "QUERY_STRING" || key == "CONTENT_LENGTH" || key == "CONTENT_TYPE" || key == "PATH_INFO" || key == "REMOTE_ADDR" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env,
		"HYDRO_ROUTE_TABLE_TEST_CHILD="+mode,
		"HYDRO_THEME_DIR="+filepath.Join(root, "themes"),
	)
	if mode == "cgi" {
		cmd.Env = append(cmd.Env,
			"REQUEST_METHOD=GET", "REQUEST_URI=/healthz", "PATH_INFO=/healthz",
			"SERVER_PROTOCOL=HTTP/1.1", "SERVER_NAME=localhost", "SERVER_PORT=80",
			"GATEWAY_INTERFACE=CGI/1.1", "REMOTE_ADDR=127.0.0.1",
		)
	}
	for _, module := range optional {
		switch module {
		case "push":
			private, public, err := webpush.GenerateVAPIDKeys()
			if err != nil {
				t.Fatal(err)
			}
			cmd.Env = append(cmd.Env, "HYDRO_PUSH_VAPID_PRIVATE_KEY="+private, "HYDRO_PUSH_VAPID_PUBLIC_KEY="+public, "HYDRO_PUSH_VAPID_SUBJECT=mailto:routes@example.test")
		case "ai":
			// Startup validates configuration only; health requests never invoke
			// the provider. No real credentials or external service are needed.
			cmd.Env = append(cmd.Env, "HYDRO_AI_ENABLED=true", "HYDRO_AI_ENDPOINT=https://ai.example.test/v1/chat/completions", "HYDRO_AI_API_KEY=route-test-only", "HYDRO_AI_MODEL=route-test")
		}
	}
	return cmd
}

// waitRuntimeReady polls /healthz until it answers 200. It reports whether the
// child exited during startup so port-conflict races can be retried; the
// channel is consumed exactly once by the caller's shutdown select.
func waitRuntimeReady(t *testing.T, port int, exited <-chan error) (ready bool, exitErr error, exitedEarly bool) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			return false, err, true
		default:
		}
		response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return true, nil, false
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false, nil, false
}

func terminateRuntimeProcess(cmd *exec.Cmd) error {
	if err := cmd.Process.Signal(syscall.SIGTERM); err == nil {
		return nil
	}
	return cmd.Process.Signal(os.Interrupt)
}

func isPortConflict(stderr *bytes.Buffer) bool {
	return strings.Contains(stderr.String(), "address already in use") || strings.Contains(stderr.String(), "bind:")
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
