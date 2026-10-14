package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type state struct {
	mu       sync.RWMutex
	Phase    string `json:"phase"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	Error    string `json:"error,omitempty"`
	Revision int    `json:"revision"`
}

func (s *state) set(phase string, progress int, message, buildError string) {
	s.mu.Lock()
	s.Phase, s.Progress, s.Message, s.Error = phase, progress, message, buildError
	s.Revision++
	s.mu.Unlock()
}

func (s *state) snapshot() state {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return state{Phase: s.Phase, Progress: s.Progress, Message: s.Message, Error: s.Error, Revision: s.Revision}
}

type supervisor struct {
	root, binary, appAddress, databaseDSN, redisURL string
	state                                           *state
	mu                                              sync.Mutex
	readyMu                                         sync.Mutex
	child                                           *exec.Cmd
}

// startLocalDependencies starts only the disposable local services declared
// by the repository. "auto" is best-effort so Compact development still works
// without Docker; "required" makes missing Docker/compose an explicit error.
func startLocalDependencies(root, mode, composeFile string) (func(), string, error) {
	if mode == "none" {
		return func() {}, "", nil
	}
	if mode != "auto" && mode != "required" {
		return nil, "", fmt.Errorf("dependencies must be auto, none, or required")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		if mode == "required" {
			return nil, "", fmt.Errorf("docker is required for local dependencies: %w", err)
		}
		return func() {}, "", nil
	}
	path := composeFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	command := exec.Command("docker", "compose", "-f", path, "up", "-d", "--wait", "redis", "minio")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		if mode == "required" {
			return nil, "", fmt.Errorf("start local dependencies: %w: %s", err, shorten(string(output), 4000))
		}
		return func() {}, "", nil
	}
	cleanup := func() {
		down := exec.Command("docker", "compose", "-f", path, "down")
		down.Dir = root
		_ = down.Run()
	}
	return cleanup, "redis://127.0.0.1:6379/0", nil
}

func (s *supervisor) rebuild(ctx context.Context, reason string, rebuild bool) {
	previousBinary := s.binary + ".previous"
	if rebuild {
		// On Windows, a running process locks its executable file (ETXTBSY / Access is denied).
		// We stop the running child before compilation so go build can overwrite the binary.
		s.stopChild()
		if _, err := os.Stat(s.binary); err == nil {
			_ = copyFile(s.binary, previousBinary)
		}
		s.state.set("building", 20, "Compiling changes…", "")
		command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", s.binary, "./cmd/hydro")
		command.Dir = s.root
		var output bytes.Buffer
		command.Stdout, command.Stderr = &output, &output
		if err := command.Run(); err != nil {
			s.state.set("error", 100, "Build failed; check console or source code.", shorten(output.String(), 12000))
			return
		}
	}

	// Keep the local database schema aligned with the binary before restarting.
	// A failed migration must not take down the currently running child.
	s.state.set("migrating", 48, "Running database migrations…", "")
	migration := exec.CommandContext(ctx, s.binary, "migrate", "--database-dsn", s.databaseDSN)
	migration.Dir = s.root
	var migrationOutput bytes.Buffer
	migration.Stdout, migration.Stderr = &migrationOutput, &migrationOutput
	if err := migration.Run(); err != nil {
		// SQLite may be locked by the running child. Retry once after stopping
		// it; if migration still fails, restore the previous binary so the
		// working version remains available.
		s.stopChild()
		migrationOutput.Reset()
		migration = exec.CommandContext(ctx, s.binary, "migrate", "--database-dsn", s.databaseDSN)
		migration.Dir = s.root
		migration.Stdout, migration.Stderr = &migrationOutput, &migrationOutput
		if retryErr := migration.Run(); retryErr != nil {
			if _, restoreErr := os.Stat(previousBinary); restoreErr == nil {
				_ = copyFile(previousBinary, s.binary)
				_ = s.startChild(s.binary)
			}
			s.state.set("error", 100, "Migration failed; preserving active version.", shorten(migrationOutput.String(), 12000))
			return
		}
	}

	s.state.set("restarting", 75, "Restarting application server…", "")
	s.stopChild()
	if err := s.startChild(s.binary); err != nil {
		s.state.set("error", 100, "Application failed to start.", err.Error())
		return
	}

	// Ping the internal app address before declaring the restart complete. A
	// child can start successfully and exit immediately (for example after a
	// startup validation error), so a successful Process.Start is not enough.
	ready := s.waitForReady(4 * time.Second)
	if !ready {
		// Restore the last known-good binary when the newly built binary cannot
		// serve. This keeps the dev environment usable while exposing the real
		// failure in the overlay instead of falsely reporting "Ready".
		s.stopChild()
		if _, restoreErr := os.Stat(previousBinary); restoreErr == nil {
			if copyErr := copyFile(previousBinary, s.binary); copyErr == nil {
				if startErr := s.startChild(s.binary); startErr == nil && s.waitForReady(4*time.Second) {
					s.state.set("ready", 100, "Previous binary restored · "+reason, "New build failed to start; running previous build.")
					return
				}
			}
		}
		s.state.set("error", 100, "App not ready; active version preserved.", "New binary did not respond to /healthz or /.")
		return
	}

	s.state.set("ready", 100, "Ready · "+reason, "")
}

func (s *supervisor) waitForReady(timeout time.Duration) bool {
	// Several browser requests can fail together while the child is restarting.
	// Serialize readiness probes so each failed request does not create its own
	// health-check storm against the application process.
	s.readyMu.Lock()
	defer s.readyMu.Unlock()

	pingClient := &http.Client{Timeout: 300 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := pingClient.Get("http://" + s.appAddress + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < http.StatusInternalServerError {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func (s *supervisor) stopChild() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil && s.child.Process != nil {
		_ = s.child.Process.Kill()
		_, _ = s.child.Process.Wait()
	}
	s.child = nil
	// On Windows, TCP sockets can linger in TIME_WAIT / CLOSE_WAIT for a moment after process exit.
	// Wait up to 1.5s for the port to be fully unbound before proceeding.
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		ln, err := net.Listen("tcp", s.appAddress)
		if err == nil {
			_ = ln.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *supervisor) startChild(binary string) error {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		child := exec.Command(binary, "serve", "--address", s.appAddress, "--database-dsn", s.databaseDSN)
		child.Dir = s.root
		env := make([]string, 0, len(os.Environ())+1)
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "HYDRO_REDIS_URL=") || strings.HasPrefix(kv, "REDIS_URL=") {
				continue
			}
			env = append(env, kv)
		}
		child.Env = append(env, "HYDRO_DEV_MODE=1")
		if s.redisURL != "" {
			child.Env = append(child.Env, "HYDRO_REDIS_URL="+s.redisURL)
		}
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			return err
		}

		s.mu.Lock()
		s.child = child
		s.mu.Unlock()

		// Run wait in background, tracking exit
		done := make(chan error, 1)
		go func(process *exec.Cmd) {
			done <- process.Wait()
		}(child)

		// Give it 150ms to ensure it didn't fail immediately on socket bind
		select {
		case err := <-done:
			lastErr = fmt.Errorf("child exited prematurely: %v", err)
			time.Sleep(200 * time.Millisecond)
			continue
		case <-time.After(150 * time.Millisecond):
			// Child is running and didn't crash on startup
			return nil
		}
	}
	return lastErr
}

func (s *supervisor) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil && s.child.Process != nil {
		_ = s.child.Process.Kill()
		_, _ = s.child.Process.Wait()
	}
}

func shorten(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o750)
}
