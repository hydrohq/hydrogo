package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEnforceDevModeLocalBind(t *testing.T) {
	cases := []struct {
		name    string
		address string
		wantErr string
	}{
		{"loopback ipv4", "127.0.0.1:8080", ""},
		{"loopback hostname", "localhost:8080", ""},
		{"loopback ipv6", "[::1]:8080", ""},
		{"non-loopback", "0.0.0.0:8080", "refused"},
		{"public address", "203.0.113.5:8080", "refused"},
		{"malformed address", "not-a-host-port", "requires a valid --address"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := enforceDevModeLocalBind(tc.address)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPrintDevModeWarning(t *testing.T) {
	var buf bytes.Buffer
	printDevModeWarning(&buf)
	if !strings.Contains(buf.String(), "DEV MODE - NOT FOR PRODUCTION") {
		t.Fatalf("expected DEV MODE - NOT FOR PRODUCTION banner, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "NOT FOR PRODUCTION") {
		t.Fatalf("expected NOT FOR PRODUCTION banner, got: %s", buf.String())
	}
}

// TestServeDevModeRejections covers paths that return synchronously (before
// any listener starts), so they can run in-process against the real serve()
// entry point via run().
func TestServeDevModeRejections(t *testing.T) {
	t.Run("dev flag rejected on non-loopback address", func(t *testing.T) {
		dsn := filepath.Join(t.TempDir(), "devmode.sqlite")
		var stdout, stderr bytes.Buffer
		err := run([]string{"serve", "--dev", "--address", "0.0.0.0:0", "--database-dialect", "sqlite", "--database-dsn", dsn}, strings.NewReader(""), &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "refused") {
			t.Fatalf("expected non-loopback rejection, got err=%v stderr=%s", err, stderr.String())
		}
	})

	t.Run("HYDRO_DEV_MODE=1 rejected on non-loopback address", func(t *testing.T) {
		t.Setenv("HYDRO_DEV_MODE", "1")
		dsn := filepath.Join(t.TempDir(), "devmode.sqlite")
		var stdout, stderr bytes.Buffer
		err := run([]string{"serve", "--address", "0.0.0.0:0", "--database-dialect", "sqlite", "--database-dsn", dsn}, strings.NewReader(""), &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "refused") {
			t.Fatalf("expected non-loopback rejection, got err=%v stderr=%s", err, stderr.String())
		}
	})

}

// A server pointed at a database nobody migrated must say what to do instead
// of failing on whichever table it happens to read first.
func TestServeRefusesAnUnmigratedDatabaseWithAHint(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "unmigrated.sqlite")
	var stdout, stderr bytes.Buffer
	err := run([]string{"serve", "--dev", "--address", "127.0.0.1:0", "--database-dialect", "sqlite", "--database-dsn", dsn}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "hydro migrate") {
		t.Fatalf("expected a hint to run hydro migrate, got err=%v stderr=%s", err, stderr.String())
	}
}

// TestDevModeServeLifecycle re-execs this test binary as a child process to
// exercise the full --dev/HYDRO_DEV_MODE bypass path end to end: a real HTTP
// listener bound to 127.0.0.1, serving /healthz without any hydro.lic, and
// shutting down gracefully. A subprocess is required (mirroring
// runtime_routes_test.go) because a successful bypass starts serve()'s
// blocking signal-driven HTTP server for real.
func TestDevModeServeLifecycle(t *testing.T) {
	if os.Getenv("HYDRO_DEV_MODE_TEST_CHILD") == "1" {
		runDevModeServeChild(t)
		return
	}

	t.Run("--dev flag", func(t *testing.T) {
		testDevModeServeStartup(t, true, false)
	})
	t.Run("HYDRO_DEV_MODE=1 env", func(t *testing.T) {
		testDevModeServeStartup(t, false, true)
	})
}

func testDevModeServeStartup(t *testing.T, useDevFlag, useDevEnv bool) {
	t.Helper()
	for attempt := 0; attempt < 3; attempt++ {
		port := freeTCPPort(t)
		cmd := devModeServeChild(t, port, useDevFlag, useDevEnv)
		var stdout, stderr synchronizedBuffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		ready, exitErr, exitedEarly := waitRuntimeReady(t, port, exited)
		if exitedEarly {
			if isPortConflict(&stderr) && attempt < 2 {
				continue
			}
			t.Fatalf("dev mode server exited during startup: %v\nstdout:\n%s\nstderr:\n%s", exitErr, &stdout, &stderr)
		}
		if !ready {
			t.Fatalf("dev mode server did not become ready within 15s\nstdout:\n%s\nstderr:\n%s", &stdout, &stderr)
		}
		if !strings.Contains(stderr.String(), "DEV MODE - NOT FOR PRODUCTION") {
			t.Fatalf("expected DEV MODE - NOT FOR PRODUCTION warning on stderr, got:\n%s", &stderr)
		}
		if err := terminateRuntimeProcess(cmd); err != nil {
			_ = cmd.Process.Kill()
			<-exited
			t.Skipf("cannot deliver shutdown signals on this platform: %v", err)
		}
		select {
		case err := <-exited:
			if err != nil {
				t.Fatalf("dev mode server did not shut down gracefully: %v\nstdout:\n%s\nstderr:\n%s", err, &stdout, &stderr)
			}
			return
		case <-time.After(30 * time.Second):
			_ = cmd.Process.Kill()
			t.Fatalf("dev mode server ignored SIGTERM for 30s\nstdout:\n%s\nstderr:\n%s", &stdout, &stderr)
		}
	}
	t.Fatal("exhausted dev mode startup retries")
}

type synchronizedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *synchronizedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(data)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func devModeServeChild(t *testing.T, port int, useDevFlag, useDevEnv bool) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestDevModeServeLifecycle$", "-test.count=1")
	cmd.Dir = t.TempDir()
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "HYDRO_") {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env,
		"HYDRO_DEV_MODE_TEST_CHILD=1",
		"HYDRO_DEV_MODE_TEST_ADDRESS=127.0.0.1:"+strconv.Itoa(port),
	)
	if useDevEnv {
		cmd.Env = append(cmd.Env, "HYDRO_DEV_MODE=1")
	}
	if useDevFlag {
		cmd.Env = append(cmd.Env, "HYDRO_DEV_MODE_TEST_USE_FLAG=1")
	}
	return cmd
}

func runDevModeServeChild(t *testing.T) {
	t.Helper()
	address := os.Getenv("HYDRO_DEV_MODE_TEST_ADDRESS")
	if address == "" {
		t.Fatal("HYDRO_DEV_MODE_TEST_ADDRESS not set")
	}
	configPath := filepath.Join(".", "config.yaml")
	licenseYAML := "license:\n  required: true\n  file: hydro.lic\n  project_id: test-project\n  edition: test-edition\n"
	if err := os.WriteFile(configPath, []byte(licenseYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	dbArgs := []string{"--database-dialect", "sqlite", "--database-dsn", "devmode.sqlite"}
	if err := run(append([]string{"migrate"}, dbArgs...), strings.NewReader(""), os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "dev mode child migrate failed: %v\n", err)
		os.Exit(1)
	}
	args := append([]string{"serve", "--config", configPath, "--address", address}, dbArgs...)
	if os.Getenv("HYDRO_DEV_MODE_TEST_USE_FLAG") == "1" {
		args = append(args, "--dev")
	}
	if err := run(args, strings.NewReader(""), os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "dev mode child serve failed: %v\n", err)
		os.Exit(1)
	}
}
