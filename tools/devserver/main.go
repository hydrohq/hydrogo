// Command devserver provides a local-only live development proxy for HydroGo.
// It deliberately stays outside the production hydro binary: production keeps
// its explicit, single-process deployment model while local development gets
// rebuilds, browser reloads and visible compiler errors.
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
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

const (
	statusPath = "/__hydrogo/dev/status"
	clientPath = "/__hydrogo/dev/client.js"
	stylePath  = "/__hydrogo/dev/client.css"
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
	pingClient := &http.Client{Timeout: 300 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, path := range []string{"/healthz", "/"} {
			resp, err := pingClient.Get("http://" + s.appAddress + path)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode < http.StatusInternalServerError {
					return true
				}
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

func main() {
	address := flag.String("address", "127.0.0.1:8090", "development proxy listen address")
	appAddress := flag.String("app-address", "127.0.0.1:8091", "internal HydroGo application address")
	databaseDSN := flag.String("database-dsn", "", "SQLite/PostgreSQL DSN passed to the child server")
	redisURL := flag.String("redis-url", "", "Optional Redis URL passed to the child server")
	dependencies := flag.String("dependencies", "auto", "Local dependencies: auto, none, or required")
	composeFile := flag.String("compose-file", "deploy/local/docker-compose.storage.yml", "Docker Compose file for local dependencies")
	poll := flag.Duration("poll", 600*time.Millisecond, "file watch polling interval")
	seedDemo := flag.Bool("seed-demo", true, "seed disposable local demo data before starting")
	devIndicator := flag.Bool("dev-indicator", true, "show the browser development status indicator")
	openBrowser := flag.Bool("open", true, "open the local development URL in the default browser")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if *databaseDSN == "" {
		*databaseDSN = database.SQLiteDSN(filepath.Join(root, "data", "demo.sqlite"))
	}
	cleanupDependencies, discoveredRedisURL, err := startLocalDependencies(root, *dependencies, *composeFile)
	if err != nil {
		fatal(err)
	}
	defer cleanupDependencies()
	if *redisURL == "" {
		*redisURL = discoveredRedisURL
	}
	binaryName := "hydro-dev"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binary := filepath.Join(root, ".tmp", "dev", binaryName)
	if err := os.MkdirAll(filepath.Dir(binary), 0o750); err != nil {
		fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	status := &state{Phase: "starting", Progress: 5, Message: "Preparing development server…"}
	supervisor := &supervisor{root: root, binary: binary, appAddress: *appAddress, databaseDSN: *databaseDSN, redisURL: *redisURL, state: status}
	supervisor.rebuild(ctx, "initial build complete", true)
	if *seedDemo {
		status.set("seeding", 88, "Preparing demo data…", "")
		seed := exec.CommandContext(ctx, binary, "seed-demo", "--confirm-demo", "--database-dsn", *databaseDSN)
		seed.Dir = root
		var seedOutput bytes.Buffer
		seed.Stdout, seed.Stderr = &seedOutput, &seedOutput
		if err := seed.Run(); err != nil {
			status.set("error", 100, "Demo data setup failed; service is running.", shorten(seedOutput.String(), 12000))
		} else {
			status.set("ready", 100, "Ready · demo data loaded", "")
		}
	}
	defer supervisor.stop()

	target, err := url.Parse("http://" + *appAddress)
	if err != nil {
		fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			if supervisor.waitForReady(150 * time.Millisecond) {
				proxy.ServeHTTP(w, r)
				return
			}
			time.Sleep(60 * time.Millisecond)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Rebuilding</title><link rel="stylesheet" href="`+stylePath+`"><script src="`+clientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card"><div class="hdev-brand"><span class="hdev-brand-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></span><span>Hydro<b>Go</b></span></div><div class="hdev-badge"><span class="hdev-pulse-dot"></span> DEV COMPILER</div><h2>Updating Server</h2><p>Compiling code changes and preparing runtime services…</p><div class="hdev-bar-wrap"><div class="hdev-bar-indeterminate"></div></div></main></body></html>`)
	}
	indicatorEnabled := *devIndicator && !envBoolDisabled("HYDRO_DEV_INDICATOR")
	proxy.ModifyResponse = func(res *http.Response) error {
		res.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		res.Header.Set("Pragma", "no-cache")
		res.Header.Set("Expires", "0")
		if indicatorEnabled {
			return injectClient(res)
		}
		return nil
	}

	holdWhileRebuilding := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If server is currently building, migrating or restarting, hold the request up to 12s so the browser seamlessly gets the freshly compiled page
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
			_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Build Error</title><link rel="stylesheet" href="`+stylePath+`"><script src="`+clientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card hdev-error" style="max-width:720px;text-align:left"><div class="hdev-brand"><span>Hydro<b>Go</b> Dev Compiler</span></div><div class="hdev-badge" style="color:#ef4444;border-color:rgba(239,68,68,0.3);background:rgba(239,68,68,0.1)">COMPILER ERROR</div><h2>Build Failed</h2><pre style="width:100%;overflow:auto;background:#05080f;color:#fca5a5;padding:14px;border-radius:10px;font-size:12px;border:1px solid rgba(255,255,255,0.1);max-height:400px">`+errorEscaped+`</pre><p style="margin-top:14px;font-size:12px;color:#94a3b8">Fix the error in your editor; the compiler will automatically reload when saved.</p></main></body></html>`)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	mux := http.NewServeMux()
	mux.HandleFunc(statusPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(status.snapshot())
	})
	if indicatorEnabled {
		mux.HandleFunc(clientPath, serveClient)
		mux.HandleFunc(stylePath, serveStyles)
	}
	mux.Handle("/", holdWhileRebuilding)

	go watch(ctx, root, *poll, func(change fileChange) {
		if change.rebuild {
			supervisor.rebuild(ctx, "updated", true)
			return
		}
		rel, err := filepath.Rel(root, change.path)
		if err != nil {
			rel = filepath.Base(change.path)
		}
		status.set("ready", 100, "Asset updated: "+filepath.ToSlash(rel), "")
	})

	// Keep the development proxy bounded just like the application server. This
	// prevents slow-client/header abuse from consuming a dev process indefinitely
	// while still allowing normal hot-reload requests to complete.
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
		fatal(err)
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelShutdown()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Printf("HydroGo dev proxy hazır: http://%s (uygulama: %s)\n", *address, *appAddress)
	if *openBrowser {
		go openDevelopmentURL(*address)
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fatal(err)
	}
}

func openDevelopmentURL(address string) {
	time.Sleep(250 * time.Millisecond)
	host, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, port)
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	_ = command.Start()
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

func injectClient(response *http.Response) error {
	if !strings.Contains(response.Header.Get("Content-Type"), "text/html") || response.Body == nil {
		return nil
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	_ = response.Body.Close()
	needle := []byte("</body>")
	client := []byte(`<script src="` + clientPath + `"></script></body>`)
	if bytes.Contains(body, needle) {
		body = bytes.Replace(body, needle, client, 1)
	} else {
		body = append(body, []byte(`<script src="`+clientPath+`"></script>`)...)
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	return nil
}

func serveClient(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = io.WriteString(w, devClient)
}

func serveStyles(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = io.WriteString(w, devStyles)
}

type fileChange struct {
	path    string
	rebuild bool
}

func watch(ctx context.Context, root string, interval time.Duration, changed func(fileChange)) {
	known := scan(root)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := scan(root)
			paths := changedPaths(known, current)
			known = current
			if len(paths) == 0 {
				continue
			}
			assetOnly := true
			for _, path := range paths {
				if !isAsset(path) {
					assetOnly = false
					break
				}
			}
			changed(fileChange{path: paths[0], rebuild: !assetOnly})
		}
	}
}

type fileStamp struct {
	modified time.Time
	size     int64
}

func scan(root string) map[string]fileStamp {
	files := make(map[string]fileStamp)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".tmp", "data", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !watched(path) {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr == nil {
			files[path] = fileStamp{modified: info.ModTime(), size: info.Size()}
		}
		return nil
	})
	return files
}

func watched(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".templ", ".html", ".css", ".js", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func isAsset(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css", ".js":
		return true
	default:
		return false
	}
}

func changedPaths(before, after map[string]fileStamp) []string {
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

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "hydro dev:", err)
	os.Exit(1)
}

const devClient = `(()=>{if(window.__hydrogoDev)return;window.__hydrogoDev=true;const link=document.createElement('link');link.rel='stylesheet';link.href='` + stylePath + `';document.head.append(link);const box=document.createElement('aside');box.id='hydrogo-dev-status';box.innerHTML='<button class="hdev-pill" type="button" aria-expanded="false" aria-label="Show HydroGo development status"><span class="hdev-dot"></span><span class="hdev-title">HYDROGO</span><span class="hdev-phase-tag">READY</span><span class="hdev-value" hidden>Ready</span><span class="hdev-chevron" aria-hidden="true">⌄</span></button><section class="hdev-panel" hidden><div class="hdev-panel-head"><strong>HydroGo Development Status</strong><button class="hdev-close" type="button" aria-label="Close">×</button></div><p class="hdev-message">Ready</p><div class="hdev-track"><i></i></div><pre hidden></pre><div class="hdev-panel-actions"><button class="hdev-hide" type="button">Hide badge</button></div></section>';document.documentElement.append(box);const pill=box.querySelector('.hdev-pill'),panel=box.querySelector('.hdev-panel'),close=box.querySelector('.hdev-close'),hideButton=box.querySelector('.hdev-hide'),value=box.querySelector('.hdev-value'),message=box.querySelector('.hdev-message'),bar=box.querySelector('i'),details=box.querySelector('pre'),phaseTag=box.querySelector('.hdev-phase-tag');let revision=-1,initialized=false,reloadOnReady=false,isReloading=false;const isRebuildingScreen=document.body.classList.contains('hydrogo-dev-body');const hiddenKey='hydrogo-dev-indicator-hidden';let userHidden=localStorage.getItem(hiddenKey)==='1';const show=()=>{if(!userHidden)box.classList.remove('hdev-hidden')};const togglePanel=(open)=>{panel.hidden=!open;pill.setAttribute('aria-expanded',String(open));box.classList.toggle('hdev-open',open)};pill.addEventListener('click',()=>togglePanel(panel.hidden));close.addEventListener('click',()=>togglePanel(false));hideButton.addEventListener('click',()=>{userHidden=true;localStorage.setItem(hiddenKey,'1');togglePanel(false);box.classList.add('hdev-hidden')});if(userHidden&&!isRebuildingScreen)box.classList.add('hdev-hidden');const doReload=()=>{if(isReloading)return;isReloading=true;try{window.location.replace(window.location.href);}catch(_){window.location.reload();}};let pollTimer=null;async function poll(){if(isReloading)return;try{const response=await fetch('` + statusPath + `?t='+Date.now(),{cache:'no-store'});if(!response.ok){show();value.textContent='Compiling…';message.textContent='Preparing runtime server…';phaseTag.textContent='STARTING';reloadOnReady=true;scheduleNext(250);return;}const status=await response.json();const phase=(status.phase||'ready').toLowerCase();const error=phase==='error'||Boolean(status.error);value.textContent=error?'Error':phase==='ready'?'Ready':phase.toUpperCase();message.textContent=status.error||status.message;bar.style.width=status.progress+'%';phaseTag.textContent=error?'ERROR':phase.toUpperCase();box.classList.toggle('hdev-error',error);box.classList.toggle('hdev-busy',!error&&phase!=='ready');details.hidden=!status.error;details.textContent=status.error||'';show();if(isRebuildingScreen){if(phase==='ready'&&!error){doReload();return;}}else{if(initialized&&status.revision!==revision){if(phase==='ready'&&!error){doReload();return;}reloadOnReady=true;}if(initialized&&reloadOnReady&&phase==='ready'&&!error){doReload();return;}}initialized=true;revision=status.revision;scheduleNext(phase==='ready'&&!reloadOnReady?1200:250);}catch(_){show();value.textContent='Offline';message.textContent='Reconnecting to dev server…';phaseTag.textContent='OFFLINE';box.classList.add('hdev-error');reloadOnReady=true;scheduleNext(350);}}function scheduleNext(ms){clearTimeout(pollTimer);pollTimer=setTimeout(poll,ms);}poll();})();`

const devStyles = `body.hydrogo-dev-body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#090d16;color:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased;padding:20px;box-sizing:border-box}
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
