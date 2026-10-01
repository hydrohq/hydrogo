// runEditionDev implements `hydro dev` for scaffolded Client Edition
// projects (produced by `hydro new` / `hydro generate scaffold`). Unlike
// tools/devserver, which owns the HydroGo Core development flow and rebuilds
// ./cmd/hydro from source on every change, a Client Edition project ships no
// Go source tree at all (see docs/architecture/0028-core-isolation-and-licensing.md).
// There is nothing to `go build`: the already-compiled hydro binary that is
// running `hydro dev` is reused as the child "serve" process via subprocess,
// and only migrations/demo seeding/config reload happen on file changes.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/fsnotify/fsnotify"
)

const (
	editionStatusPath = "/__hydrogo/dev/status"
	editionClientPath = "/__hydrogo/dev/client.js"
	editionStylePath  = "/__hydrogo/dev/client.css"
)

// editionSelfBinaryEnv lets tests (and, in principle, operators) point
// `hydro dev` at a specific already-built hydro binary instead of resolving
// the currently running executable through os.Executable().
const editionSelfBinaryEnv = "HYDRO_DEV_SELF_BINARY"

// editionWatchedTopDirs are the only directories a Client Edition project
// carries application-affecting content in. There is no Go source anywhere
// in the tree, so unlike tools/devserver there is no reason to watch the
// whole project (and no "rebuild the binary" step at all).
var editionWatchedTopDirs = map[string]bool{
	"themes":     true,
	"migrations": true,
	"custom":     true,
	"seed":       true,
}

type editionState struct {
	mu       sync.RWMutex
	Phase    string `json:"phase"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	Error    string `json:"error,omitempty"`
	Revision int    `json:"revision"`
}

func (s *editionState) set(phase string, progress int, message, buildError string) {
	s.mu.Lock()
	s.Phase, s.Progress, s.Message, s.Error = phase, progress, message, buildError
	s.Revision++
	s.mu.Unlock()
}

func (s *editionState) snapshot() editionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return editionState{Phase: s.Phase, Progress: s.Progress, Message: s.Message, Error: s.Error, Revision: s.Revision}
}

// editionSupervisor restarts the already-built hydro binary as a child
// "serve" process after migrations/config/theme/content changes. It does not
// compile anything and does not keep a "previous binary" fallback, because
// the binary itself never changes while `hydro dev` is running.
type editionSupervisor struct {
	root, selfBinary, configPath, appAddress string
	state                                    *editionState
	stdout, stderr                           io.Writer

	mu      sync.Mutex
	readyMu sync.Mutex
	child   *exec.Cmd
}

func (s *editionSupervisor) restart(ctx context.Context, reason string) {
	s.state.set("migrating", 40, "Running database migrations…", "")
	migration := exec.CommandContext(ctx, s.selfBinary, "migrate", "--config", s.configPath)
	migration.Dir = s.root
	var migrationOutput bytes.Buffer
	migration.Stdout, migration.Stderr = &migrationOutput, &migrationOutput
	if err := migration.Run(); err != nil {
		s.state.set("error", 100, "Migration failed; preserving active version.", shorten(migrationOutput.String(), 12000))
		return
	}

	s.state.set("restarting", 75, "Restarting application server…", "")
	s.stopChild()
	if err := s.startChild(ctx); err != nil {
		s.state.set("error", 100, "Application failed to start.", err.Error())
		return
	}

	if !s.waitForReady(4 * time.Second) {
		s.state.set("error", 100, "App not ready; check hydro.lic/license configuration.", "The server process did not respond to /healthz. If config.yaml has license.required: true, a missing or invalid hydro.lic is the most likely cause.")
		return
	}

	s.state.set("ready", 100, "Ready · "+reason, "")
}

func (s *editionSupervisor) waitForReady(timeout time.Duration) bool {
	// Serialize readiness probes the same way tools/devserver does, so a
	// burst of browser requests during a restart does not create its own
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

func (s *editionSupervisor) stopChild() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil && s.child.Process != nil {
		_ = s.child.Process.Kill()
		_, _ = s.child.Process.Wait()
	}
	s.child = nil
	// On Windows, TCP sockets can linger in TIME_WAIT / CLOSE_WAIT for a
	// moment after process exit. Wait up to 1.5s for the port to free up.
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

func (s *editionSupervisor) startChild(ctx context.Context) error {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		child := exec.Command(s.selfBinary, "serve", "--config", s.configPath, "--address", s.appAddress)
		child.Dir = s.root
		child.Env = append(os.Environ(), "HYDRO_DEV_MODE=1")
		child.Stdout, child.Stderr = s.stdout, s.stderr

		if err := child.Start(); err != nil {
			return err
		}

		s.mu.Lock()
		s.child = child
		s.mu.Unlock()

		done := make(chan error, 1)
		go func(process *exec.Cmd) { done <- process.Wait() }(child)

		select {
		case err := <-done:
			lastErr = fmt.Errorf("child exited prematurely: %v", err)
			time.Sleep(200 * time.Millisecond)
			continue
		case <-time.After(150 * time.Millisecond):
			return nil
		}
	}
	return lastErr
}

func (s *editionSupervisor) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil && s.child.Process != nil {
		_ = s.child.Process.Kill()
		_, _ = s.child.Process.Wait()
	}
}

// resolveEditionSelfBinary returns the already-compiled hydro binary to run
// `migrate`/`seed-demo`/`serve` subcommands against. Client Edition dev mode
// never invokes `go build`.
func resolveEditionSelfBinary() (string, error) {
	if override := strings.TrimSpace(os.Getenv(editionSelfBinaryEnv)); override != "" {
		return override, nil
	}
	return os.Executable()
}

// ensureEditionSQLiteDir creates the directory holding the configured SQLite
// database file if it does not exist yet. A freshly scaffolded project has
// no `data/` directory, and migrate/serve fail outright without it.
func ensureEditionSQLiteDir(configPath string) error {
	cfg, err := config.Load(configPath, os.LookupEnv, config.Overrides{})
	if err != nil {
		return fmt.Errorf("load %s: %w", configPath, err)
	}
	if cfg.Database.Dialect != database.SQLite {
		return nil
	}
	dbPath, err := sqliteDatabasePath(cfg.Database.DSN)
	if err != nil {
		return nil // non-file-backed DSNs (e.g. :memory:) need no directory.
	}
	return os.MkdirAll(filepath.Dir(dbPath), 0o750)
}

func runEditionDev(args []string, stdout, stderr io.Writer) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return runEditionDevContext(ctx, args, stdout, stderr)
}

// runEditionDevContext is the testable core of runEditionDev: tests supply a
// cancellable context instead of a process-wide signal.NotifyContext so the
// proxy server and the "serve" child process shut down deterministically at
// the end of the test instead of leaking past it.
func runEditionDevContext(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "config.yaml", "YAML config file")
	address := flags.String("address", "0.0.0.0:8090", "development proxy listen address")
	appAddress := flags.String("app-address", "127.0.0.1:8091", "internal HydroGo application address")
	poll := flags.Duration("poll", 600*time.Millisecond, "file watch polling interval")
	seedDemo := flags.Bool("seed-demo", true, "seed the disposable local demo data before starting")
	devIndicator := flags.Bool("dev-indicator", true, "show the browser development status indicator")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	selfBinary, err := resolveEditionSelfBinary()
	if err != nil {
		return fmt.Errorf("resolve hydro binary: %w", err)
	}
	if err := ensureEditionSQLiteDir(*configPath); err != nil {
		return err
	}

	status := &editionState{Phase: "starting", Progress: 5, Message: "Preparing development server…"}
	supervisor := &editionSupervisor{
		root:       root,
		selfBinary: selfBinary,
		configPath: *configPath,
		appAddress: *appAddress,
		state:      status,
		stdout:     stdout,
		stderr:     stderr,
	}
	supervisor.restart(ctx, "initial start complete")

	if *seedDemo {
		status.set("seeding", 88, "Preparing demo data…", "")
		seed := exec.CommandContext(ctx, selfBinary, "seed-demo", "--confirm-demo", "--config", *configPath)
		seed.Dir = root
		var seedOutput bytes.Buffer
		seed.Stdout, seed.Stderr = &seedOutput, &seedOutput
		if err := seed.Run(); err != nil {
			// Demo seeding is a convenience, not a hard requirement; keep the
			// already-running server up and surface the failure.
			status.set("error", 100, "Demo data setup failed; service is running.", shorten(seedOutput.String(), 12000))
		} else {
			status.set("ready", 100, "Ready · demo data loaded", "")
		}
	}
	defer supervisor.stop()

	target, err := url.Parse("http://" + *appAddress)
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		if supervisor.waitForReady(4 * time.Second) {
			proxy.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Rebuilding</title><link rel="stylesheet" href="`+editionStylePath+`"><script src="`+editionClientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card"><div class="hdev-brand"><span class="hdev-brand-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></span><span>Hydro<b>Go</b></span></div><div class="hdev-badge"><span class="hdev-pulse-dot"></span> DEV COMPILER</div><h2>Updating Server</h2><p>Running migrations and restarting the application server…</p><div class="hdev-bar-wrap"><div class="hdev-bar-indeterminate"></div></div></main></body></html>`)
	}
	indicatorEnabled := *devIndicator && !envBoolDisabled("HYDRO_DEV_INDICATOR")
	proxy.ModifyResponse = func(res *http.Response) error {
		res.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		res.Header.Set("Pragma", "no-cache")
		res.Header.Set("Expires", "0")
		if indicatorEnabled {
			return editionInjectClient(res)
		}
		return nil
	}

	holdWhileRebuilding := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			snap := status.snapshot()
			if snap.Phase == "ready" || snap.Phase == "error" {
				break
			}
			time.Sleep(30 * time.Millisecond)
		}

		snap := status.snapshot()
		if snap.Phase == "error" && snap.Error != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.WriteHeader(http.StatusInternalServerError)
			errorEscaped := template.HTMLEscapeString(snap.Error)
			_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Dev Error</title><link rel="stylesheet" href="`+editionStylePath+`"><script src="`+editionClientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card hdev-error" style="max-width:720px;text-align:left"><div class="hdev-brand"><span>Hydro<b>Go</b> Dev Server</span></div><div class="hdev-badge" style="color:#ef4444;border-color:rgba(239,68,68,0.3);background:rgba(239,68,68,0.1)">SERVER ERROR</div><h2>Not Ready</h2><pre style="width:100%;overflow:auto;background:#05080f;color:#fca5a5;padding:14px;border-radius:10px;font-size:12px;border:1px solid rgba(255,255,255,0.1);max-height:400px">`+errorEscaped+`</pre><p style="margin-top:14px;font-size:12px;color:#94a3b8">Fix the issue (for example, place a valid hydro.lic in the project root) and this page will refresh automatically.</p></main></body></html>`)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	mux := http.NewServeMux()
	mux.HandleFunc(editionStatusPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(status.snapshot())
	})
	if indicatorEnabled {
		mux.HandleFunc(editionClientPath, editionServeClient)
		mux.HandleFunc(editionStylePath, editionServeStyles)
	}
	mux.Handle("/", holdWhileRebuilding)

	go editionWatch(ctx, root, *poll, func(change editionFileChange) {
		if change.rebuild {
			supervisor.restart(ctx, "updated")
			return
		}
		rel, err := filepath.Rel(root, change.path)
		if err != nil {
			rel = filepath.Base(change.path)
		}
		status.set("ready", 100, "Asset updated: "+filepath.ToSlash(rel), "")
	})

	server := &http.Server{
		Addr:              *address,
		Handler:           noStore(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelShutdown()
		_ = server.Shutdown(shutdownCtx)
	}()
	_, _ = fmt.Fprintf(stdout, "HydroGo dev proxy hazır: http://%s (uygulama: %s)\n", *address, *appAddress)
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func editionInjectClient(response *http.Response) error {
	if !strings.Contains(response.Header.Get("Content-Type"), "text/html") || response.Body == nil {
		return nil
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	_ = response.Body.Close()
	needle := []byte("</body>")
	client := []byte(`<script src="` + editionClientPath + `"></script></body>`)
	if bytes.Contains(body, needle) {
		body = bytes.Replace(body, needle, client, 1)
	} else {
		body = append(body, []byte(`<script src="`+editionClientPath+`"></script>`)...)
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	return nil
}

func editionServeClient(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = io.WriteString(w, editionDevClient)
}

func editionServeStyles(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = io.WriteString(w, editionDevStyles)
}

type editionFileChange struct {
	path    string
	rebuild bool
}

func editionWatch(ctx context.Context, root string, interval time.Duration, changed func(editionFileChange)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		editionPollWatch(ctx, root, interval, changed)
		return
	}
	defer watcher.Close()

	if err := editionAddWatchDirectories(watcher, root); err != nil {
		editionPollWatch(ctx, root, interval, changed)
		return
	}

	pending := make(map[string]struct{})
	var debounce *time.Timer
	var debounceC <-chan time.Time
	flush := func() {
		if len(pending) == 0 {
			return
		}
		paths := make([]string, 0, len(pending))
		for path := range pending {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		assetOnly := true
		for _, path := range paths {
			if !editionIsAsset(path) {
				assetOnly = false
				break
			}
		}
		pending = make(map[string]struct{})
		changed(editionFileChange{path: paths[0], rebuild: !assetOnly})
	}

	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() && editionShouldWatchDirectory(event.Name, root) {
					_ = watcher.Add(event.Name)
					continue
				}
			}
			// A file write inside a watched directory also updates that
			// directory's own mtime, which some platforms report as a
			// separate Write event on the directory path itself. Directory
			// paths have no file extension to match against, which would
			// otherwise spuriously mark a pure asset change as needing a
			// full migrate+restart; skip them explicitly.
			if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
				continue
			}
			if !editionWatchedFile(event.Name, root) {
				continue
			}
			pending[event.Name] = struct{}{}
			if debounce == nil {
				debounce = time.NewTimer(150 * time.Millisecond)
			} else {
				if !debounce.Stop() {
					select {
					case <-debounce.C:
					default:
					}
				}
				debounce.Reset(150 * time.Millisecond)
			}
			debounceC = debounce.C
		case <-debounceC:
			debounceC = nil
			flush()
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func editionPollWatch(ctx context.Context, root string, interval time.Duration, changed func(editionFileChange)) {
	known := editionScan(root)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := editionScan(root)
			paths := editionChangedPaths(known, current)
			known = current
			if len(paths) == 0 {
				continue
			}
			assetOnly := true
			for _, path := range paths {
				if !editionIsAsset(path) {
					assetOnly = false
					break
				}
			}
			changed(editionFileChange{path: paths[0], rebuild: !assetOnly})
		}
	}
}

func editionAddWatchDirectories(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root && !editionShouldWatchDirectory(path, root) {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

// editionShouldWatchDirectory limits recursion to the top-level directories
// a Client Edition project actually ships content-affecting files in. The
// project root itself is also watched (non-recursively covers config.yaml
// and hydrogo.lock).
func editionShouldWatchDirectory(path, root string) bool {
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if !editionWatchedTopDirs[parts[0]] {
		return false
	}
	for _, part := range parts {
		switch part {
		case ".git", "data", "node_modules", "vendor", ".tmp":
			return false
		}
	}
	return true
}

// editionWatchedFile reports whether a changed path is one `hydro dev`
// cares about: config.yaml/hydrogo.lock at the project root, or any file
// under themes/, migrations/, custom/, seed/.
func editionWatchedFile(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == "config.yaml" || rel == "hydrogo.lock" {
		return true
	}
	parts := strings.Split(rel, "/")
	if !editionWatchedTopDirs[parts[0]] {
		return false
	}
	for _, part := range parts {
		switch part {
		case ".git", "data", "node_modules", "vendor", ".tmp":
			return false
		}
	}
	return true
}

func editionIsAsset(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".css" && extension != ".js" {
		return false
	}
	rel := "/" + filepath.ToSlash(path) + "/"
	return strings.Contains(rel, "/themes/") && strings.Contains(rel, "/static/")
}

type editionFileStamp struct {
	modified time.Time
	size     int64
}

func editionScan(root string) map[string]editionFileStamp {
	files := make(map[string]editionFileStamp)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && !editionShouldWatchDirectory(path, root) {
				return filepath.SkipDir
			}
			return nil
		}
		if !editionWatchedFile(path, root) {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr == nil {
			files[path] = editionFileStamp{modified: info.ModTime(), size: info.Size()}
		}
		return nil
	})
	return files
}

func editionChangedPaths(before, after map[string]editionFileStamp) []string {
	paths := make([]string, 0)
	for path, stamp := range after {
		if old, ok := before[path]; !ok || old != stamp {
			paths = append(paths, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func envBoolDisabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func shorten(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}

const editionDevClient = `(()=>{if(window.__hydrogoDev)return;window.__hydrogoDev=true;const link=document.createElement('link');link.rel='stylesheet';link.href='` + editionStylePath + `';document.head.append(link);const box=document.createElement('aside');box.id='hydrogo-dev-status';box.innerHTML='<button class="hdev-pill" type="button" aria-expanded="false" aria-label="Show HydroGo development status"><span class="hdev-dot"></span><span class="hdev-title">HYDROGO</span><span class="hdev-phase-tag">READY</span><span class="hdev-value" hidden>Ready</span><span class="hdev-chevron" aria-hidden="true">⌄</span></button><section class="hdev-panel" hidden><div class="hdev-panel-head"><strong>HydroGo Development Status</strong><button class="hdev-close" type="button" aria-label="Close">×</button></div><p class="hdev-message">Ready</p><div class="hdev-track"><i></i></div><pre hidden></pre><div class="hdev-panel-actions"><button class="hdev-hide" type="button">Hide badge</button></div></section>';document.documentElement.append(box);const pill=box.querySelector('.hdev-pill'),panel=box.querySelector('.hdev-panel'),close=box.querySelector('.hdev-close'),hideButton=box.querySelector('.hdev-hide'),value=box.querySelector('.hdev-value'),message=box.querySelector('.hdev-message'),bar=box.querySelector('i'),details=box.querySelector('pre'),phaseTag=box.querySelector('.hdev-phase-tag');let revision=-1,initialized=false,reloadOnReady=false,isReloading=false;const isRebuildingScreen=document.body.classList.contains('hydrogo-dev-body');const hiddenKey='hydrogo-dev-indicator-hidden';let userHidden=localStorage.getItem(hiddenKey)==='1';const show=()=>{if(!userHidden)box.classList.remove('hdev-hidden')};const togglePanel=(open)=>{panel.hidden=!open;pill.setAttribute('aria-expanded',String(open));box.classList.toggle('hdev-open',open)};pill.addEventListener('click',()=>togglePanel(panel.hidden));close.addEventListener('click',()=>togglePanel(false));hideButton.addEventListener('click',()=>{userHidden=true;localStorage.setItem(hiddenKey,'1');togglePanel(false);box.classList.add('hdev-hidden')});if(userHidden&&!isRebuildingScreen)box.classList.add('hdev-hidden');const doReload=()=>{if(isReloading)return;isReloading=true;try{window.location.replace(window.location.href);}catch(_){window.location.reload();}};let pollTimer=null;async function poll(){if(isReloading)return;try{const response=await fetch('` + editionStatusPath + `?t='+Date.now(),{cache:'no-store'});if(!response.ok){show();value.textContent='Compiling…';message.textContent='Preparing runtime server…';phaseTag.textContent='STARTING';reloadOnReady=true;scheduleNext(250);return;}const status=await response.json();const phase=(status.phase||'ready').toLowerCase();const error=phase==='error'||Boolean(status.error);value.textContent=error?'Error':phase==='ready'?'Ready':phase.toUpperCase();message.textContent=status.error||status.message;bar.style.width=status.progress+'%';phaseTag.textContent=error?'ERROR':phase.toUpperCase();box.classList.toggle('hdev-error',error);box.classList.toggle('hdev-busy',!error&&phase!=='ready');details.hidden=!status.error;details.textContent=status.error||'';show();if(isRebuildingScreen){if(phase==='ready'&&!error){doReload();return;}}else{if(initialized&&status.revision!==revision){if(phase==='ready'&&!error){doReload();return;}reloadOnReady=true;}if(initialized&&reloadOnReady&&phase==='ready'&&!error){doReload();return;}}initialized=true;revision=status.revision;scheduleNext(phase==='ready'&&!reloadOnReady?1200:250);}catch(_){show();value.textContent='Offline';message.textContent='Reconnecting to dev server…';phaseTag.textContent='OFFLINE';box.classList.add('hdev-error');reloadOnReady=true;scheduleNext(350);}}function scheduleNext(ms){clearTimeout(pollTimer);pollTimer=setTimeout(poll,ms);}poll();})();`

const editionDevStyles = `body.hydrogo-dev-body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#090d16;color:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased;padding:20px;box-sizing:border-box}
.hydrogo-dev-card{display:flex;flex-direction:column;align-items:center;text-align:center;padding:32px 28px 28px;max-width:360px;width:100%;background:#111927;border:1px solid rgba(255,255,255,0.08);border-radius:18px;box-shadow:0 20px 40px -10px rgba(0,0,0,0.5);position:relative;overflow:hidden}
.hdev-brand{display:inline-flex;align-items:center;gap:8px;font-size:1.15rem;font-weight:700;letter-spacing:-.02em;color:#f8fafc;margin-bottom:16px}
.hdev-brand-icon{display:grid;place-items:center;width:28px;height:28px;border-radius:8px;background:rgba(14,165,233,0.14);color:#38bdf8;border:1px solid rgba(14,165,233,0.25)}
.hdev-brand b{color:#38bdf8}
.hdev-badge{display:inline-flex;align-items:center;gap:6px;font-size:.68rem;font-weight:700;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#38bdf8;background:rgba(14,165,233,0.1);border:1px solid rgba(14,165,233,0.25);padding:3px 10px;border-radius:9999px;margin-bottom:12px;letter-spacing:.03em}
.hdev-pulse-dot{width:6px;height:6px;border-radius:50%;background:#38bdf8;box-shadow:0 0 8px #38bdf8;animation:hdev-pulse 1.8s infinite ease-in-out}
@keyframes hdev-pulse{0%,100%{opacity:1;transform:scale(1)}50%{opacity:.4;transform:scale(.85)}}
.hydrogo-dev-card h2{margin:0 0 6px;font-size:1.2rem;font-weight:700;letter-spacing:-.02em;color:#f8fafc}
.hydrogo-dev-card p{margin:0 0 20px;font-size:.84rem;color:#94a3b8;line-height:1.45}
.hdev-bar-wrap{width:100%;height:4px;border-radius:9999px;background:rgba(255,255,255,0.06);overflow:hidden;position:relative}
.hdev-bar-indeterminate{position:absolute;height:100%;background:#0ea5e9;border-radius:9999px;animation:hdev-progress 1.4s infinite ease-in-out}
@keyframes hdev-progress{0%{left:-40%;width:40%}50%{left:20%;width:60%}100%{left:100%;width:40%}}

#hydrogo-dev-status{position:fixed;right:16px;bottom:16px;z-index:2147483647;width:auto;max-width:min(340px,calc(100vw - 32px));color:#f8fafc;font:12px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;transition:all .18s ease}
#hydrogo-dev-status.hdev-hidden{opacity:0;pointer-events:none;transform:translateY(8px)}
.hdev-pill{display:inline-flex;align-items:center;gap:7px;padding:5px 12px;border:1px solid rgba(255,255,255,0.12);border-radius:9999px;background:#111927;color:#f8fafc;box-shadow:0 4px 14px rgba(0,0,0,.35);cursor:pointer;text-align:left;font-family:inherit}
.hdev-pill:hover{background:#182235;border-color:rgba(255,255,255,0.2)}
.hdev-pill:focus-visible,.hdev-close:focus-visible,.hdev-hide:focus-visible{outline:2px solid #0ea5e9;outline-offset:2px}
.hdev-dot{width:7px;height:7px;flex:0 0 auto;border-radius:50%;background:#10b981}
.hdev-busy .hdev-dot{background:#f59e0b;animation:hdev-pulse 1.4s infinite ease-in-out}
.hdev-error .hdev-dot{background:#ef4444}
.hdev-title{font-weight:700;font-size:11px;letter-spacing:.02em}
.hdev-phase-tag{color:#94a3b8;font-size:10px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
.hdev-error .hdev-phase-tag{color:#f87171}
.hdev-value{display:none!important}
.hdev-chevron{color:#64748b;font-size:14px;line-height:10px}
.hdev-panel{margin-top:6px;padding:12px 14px;border:1px solid rgba(255,255,255,0.1);border-radius:12px;background:#111927;box-shadow:0 10px 28px rgba(0,0,0,.45)}
.hdev-panel[hidden]{display:none}
.hdev-panel-head{display:flex;align-items:center;justify-content:space-between;color:#f8fafc;font-weight:700;font-size:12px}
.hdev-close{border:0;background:transparent;color:#94a3b8;font-size:18px;line-height:1;cursor:pointer;padding:0 2px}
.hdev-close:hover{color:#fff}
.hdev-message{margin:8px 0;color:#cbd5e1;font-size:11.5px}
.hdev-track{height:3px;overflow:hidden;border-radius:9999px;background:rgba(255,255,255,0.08)}
.hdev-track i{display:block;width:5%;height:100%;border-radius:inherit;background:#0ea5e9;transition:width .25s ease}
.hdev-error .hdev-track i{background:#ef4444}
.hdev-panel pre{max-height:200px;margin:8px 0 0;padding:8px 10px;overflow:auto;border-radius:8px;background:#070b12;border:1px solid rgba(255,255,255,0.06);color:#fca5a5;font:11px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre-wrap}
.hdev-panel-actions{margin-top:8px;display:flex;justify-content:flex-end}
.hdev-hide{padding:0;border:0;background:transparent;color:#64748b;font-size:11px;text-decoration:none;cursor:pointer}
.hdev-hide:hover{color:#cbd5e1}`
