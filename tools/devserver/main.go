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
	"io"
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
		if _, err := os.Stat(s.binary); err == nil {
			if err := copyFile(s.binary, previousBinary); err != nil {
				s.state.set("error", 100, "Çalışan sürüm yedeklenemedi; mevcut sürüm korunuyor.", err.Error())
				return
			}
		}
		s.state.set("building", 20, "Değişiklik algılandı: derleniyor…", "")
		command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", s.binary, "./cmd/hydro")
		command.Dir = s.root
		var output bytes.Buffer
		command.Stdout, command.Stderr = &output, &output
		if err := command.Run(); err != nil {
			s.state.set("error", 100, "Derleme başarısız; çalışan sürüm korunuyor.", shorten(output.String(), 12000))
			return
		}
	}

	// Keep the local database schema aligned with the binary before restarting.
	// A failed migration must not take down the currently running child.
	s.state.set("migrating", 48, "Veritabanı migration'ları uygulanıyor…", "")
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
			s.state.set("error", 100, "Migration başarısız; çalışan sürüm korunuyor.", shorten(migrationOutput.String(), 12000))
			return
		}
	}

	s.state.set("restarting", 75, "Uygulama yeniden başlatılıyor…", "")
	s.stopChild()
	if err := s.startChild(s.binary); err != nil {
		s.state.set("error", 100, "Uygulama başlatılamadı.", err.Error())
		return
	}

	// Ping the internal app address before declaring the restart complete. A
	// child can start successfully and exit immediately (for example after a
	// startup validation error), so a successful Process.Start is not enough.
	ready := s.waitForReady(4 * time.Second)
	if !ready {
		// Restore the last known-good binary when the newly built binary cannot
		// serve. This keeps the dev environment usable while exposing the real
		// failure in the overlay instead of falsely reporting "Hazır".
		s.stopChild()
		if _, restoreErr := os.Stat(previousBinary); restoreErr == nil {
			if copyErr := copyFile(previousBinary, s.binary); copyErr == nil {
				if startErr := s.startChild(s.binary); startErr == nil && s.waitForReady(4*time.Second) {
					s.state.set("ready", 100, "Önceki çalışan sürüm geri yüklendi · "+reason, "Yeni sürüm başlatılamadı; önceki sürüm çalışıyor.")
					return
				}
			}
		}
		s.state.set("error", 100, "Uygulama hazır değil; çalışan sürüm korunuyor.", "Yeni binary /healthz veya / yanıtı vermedi.")
		return
	}

	s.state.set("ready", 100, "Hazır · "+reason, "")
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
}

func (s *supervisor) startChild(binary string) error {
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
	go func(process *exec.Cmd) { _ = process.Wait() }(child)
	return nil
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
	status := &state{Phase: "starting", Progress: 5, Message: "Geliştirme sunucusu hazırlanıyor…"}
	supervisor := &supervisor{root: root, binary: binary, appAddress: *appAddress, databaseDSN: *databaseDSN, redisURL: *redisURL, state: status}
	supervisor.rebuild(ctx, "ilk build tamamlandı", true)
	if *seedDemo {
		status.set("seeding", 88, "Demo verileri hazırlanıyor…", "")
		seed := exec.CommandContext(ctx, binary, "seed-demo", "--confirm-demo", "--database-dsn", *databaseDSN)
		seed.Dir = root
		var seedOutput bytes.Buffer
		seed.Stdout, seed.Stderr = &seedOutput, &seedOutput
		if err := seed.Run(); err != nil {
			status.set("error", 100, "Demo verileri hazırlanamadı; servis yine de çalışıyor.", shorten(seedOutput.String(), 12000))
		} else {
			status.set("ready", 100, "Hazır · demo verileri yüklendi", "")
		}
	}
	defer supervisor.stop()

	target, err := url.Parse("http://" + *appAddress)
	if err != nil {
		fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, proxyErr error) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Yeniden Başlatılıyor</title><link rel="stylesheet" href="`+stylePath+`"><script src="`+clientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card"><div class="hdev-brand"><span class="hdev-brand-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg></span>HYDRO<b>GO</b></div><div class="hdev-spinner"></div><div class="hdev-badge"><span class="hdev-pulse-dot"></span> GELİŞTİRME DERLEMESİ</div><h2>Sunucu Güncelleniyor</h2><p>Kaynak kod veya şablon değişiklikleri derleniyor, servis hazırlanıyor…</p><div class="hdev-bar-wrap"><div class="hdev-bar-indeterminate"></div></div></main></body></html>`)
	}
	indicatorEnabled := *devIndicator && !envBoolDisabled("HYDRO_DEV_INDICATOR")
	if indicatorEnabled {
		proxy.ModifyResponse = injectClient
	}
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
	mux.Handle("/", proxy)

	go watch(ctx, root, *poll, func(change fileChange) {
		if change.rebuild {
			supervisor.rebuild(ctx, "güncellendi", true)
			return
		}
		rel, err := filepath.Rel(root, change.path)
		if err != nil {
			rel = filepath.Base(change.path)
		}
		status.set("ready", 100, "Asset güncellendi: "+filepath.ToSlash(rel), "")
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
	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelShutdown()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Printf("HydroGo dev proxy hazır: http://%s (uygulama: %s)\n", *address, *appAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fatal(err)
	}
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

const devClient = `(()=>{if(window.__hydrogoDev)return;window.__hydrogoDev=true;const link=document.createElement('link');link.rel='stylesheet';link.href='` + stylePath + `';document.head.append(link);const box=document.createElement('aside');box.id='hydrogo-dev-status';box.innerHTML='<button class="hdev-pill" type="button" aria-expanded="false" aria-label="HydroGo geliştirme durumunu aç"><span class="hdev-dot"></span><span class="hdev-title">HYDROGO DEV</span><span class="hdev-phase-tag">LIVE</span><span class="hdev-value">Hazırlanıyor…</span><span class="hdev-chevron" aria-hidden="true">⌄</span></button><section class="hdev-panel" hidden><div class="hdev-panel-head"><strong>Geliştirme durumu</strong><button class="hdev-close" type="button" aria-label="Paneli kapat">×</button></div><p class="hdev-message">Hazırlanıyor…</p><div class="hdev-track"><i></i></div><pre hidden></pre><div class="hdev-panel-actions"><button class="hdev-hide" type="button">Göstergeyi gizle</button></div></section>';document.documentElement.append(box);const pill=box.querySelector('.hdev-pill'),panel=box.querySelector('.hdev-panel'),close=box.querySelector('.hdev-close'),hideButton=box.querySelector('.hdev-hide'),value=box.querySelector('.hdev-value'),message=box.querySelector('.hdev-message'),bar=box.querySelector('i'),details=box.querySelector('pre'),phaseTag=box.querySelector('.hdev-phase-tag');let revision=-1,initialized=false,reloadOnReady=false;const hiddenKey='hydrogo-dev-indicator-hidden';let userHidden=localStorage.getItem(hiddenKey)==='1';const show=()=>{if(!userHidden)box.classList.remove('hdev-hidden')};const togglePanel=(open)=>{panel.hidden=!open;pill.setAttribute('aria-expanded',String(open));box.classList.toggle('hdev-open',open)};pill.addEventListener('click',()=>togglePanel(panel.hidden));close.addEventListener('click',()=>togglePanel(false));hideButton.addEventListener('click',()=>{userHidden=true;localStorage.setItem(hiddenKey,'1');togglePanel(false);box.classList.add('hdev-hidden')});if(userHidden)box.classList.add('hdev-hidden');async function poll(){try{const response=await fetch('` + statusPath + `',{cache:'no-store'});if(!response.ok){show();value.textContent='Sunucu hazırlanıyor…';message.textContent='Sunucu hazırlanıyor…';phaseTag.textContent='STARTING';if(initialized)reloadOnReady=true;return;}const status=await response.json();const phase=(status.phase||'ready').toLowerCase();const error=phase==='error'||Boolean(status.error);value.textContent=error?'Hata':status.message+' · '+status.progress+'%';message.textContent=status.error||status.message;bar.style.width=status.progress+'%';phaseTag.textContent=error?'ERROR':phase.toUpperCase();box.classList.toggle('hdev-error',error);box.classList.toggle('hdev-busy',!error&&phase!=='ready');details.hidden=!status.error;details.textContent=status.error||'';show();if(initialized&&status.revision!==revision){if(phase==='ready'){setTimeout(()=>location.reload(),180);return}reloadOnReady=true}if(initialized&&reloadOnReady&&phase==='ready'){setTimeout(()=>location.reload(),180);return}initialized=true;revision=status.revision}catch(_){show();value.textContent='Bağlantı yok';message.textContent='Geliştirme sunucusuna yeniden bağlanılıyor…';phaseTag.textContent='OFFLINE';box.classList.add('hdev-error');if(initialized)reloadOnReady=true;}}poll();setInterval(poll,600)})();`

const devStyles = `body.hydrogo-dev-body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#0b0f19;background-image:radial-gradient(circle at 50% 0%,rgba(59,130,246,0.32),transparent 50%),radial-gradient(circle at 85% 20%,rgba(6,182,212,0.22),transparent 45%),radial-gradient(circle at 15% 55%,rgba(139,92,246,0.2),transparent 45%),radial-gradient(circle at 75% 85%,rgba(59,130,246,0.18),transparent 50%);background-attachment:fixed;color:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased;padding:24px;box-sizing:border-box}
.hydrogo-dev-card{display:flex;flex-direction:column;align-items:center;text-align:center;padding:44px 36px 38px;max-width:420px;width:100%;background:linear-gradient(165deg,rgba(23,34,56,0.88) 0%,rgba(13,20,36,0.96) 100%);backdrop-filter:blur(24px);-webkit-backdrop-filter:blur(24px);border:1px solid rgba(96,165,250,0.28);border-top:1px solid rgba(255,255,255,0.25);border-radius:28px;box-shadow:0 25px 60px -12px rgba(0,0,0,0.75),0 0 45px rgba(6,182,212,0.12);position:relative;overflow:hidden}
.hydrogo-dev-card::before{content:'';position:absolute;top:0;left:0;right:0;height:2px;background:linear-gradient(90deg,transparent 0%,rgba(6,182,212,0.8) 35%,rgba(59,130,246,0.8) 65%,transparent 100%)}
.hdev-brand{display:inline-flex;align-items:center;gap:10px;font-size:1.35rem;font-weight:800;letter-spacing:-.03em;color:#fff;margin-bottom:20px}
.hdev-brand-icon{width:34px;height:34px;border-radius:10px;background:linear-gradient(135deg,rgba(6,182,212,0.2) 0%,rgba(59,130,246,0.3) 100%);border:1px solid rgba(56,189,248,0.4);display:flex;align-items:center;justify-content:center;color:#38bdf8;box-shadow:0 4px 14px rgba(6,182,212,0.25)}
.hdev-brand b{color:#fff}
.hdev-spinner{width:42px;height:42px;border:3px solid rgba(6,182,212,0.18);border-top-color:#38bdf8;border-right-color:#818cf8;border-radius:50%;animation:hdev-spin .85s cubic-bezier(.5,.1,.4,.9) infinite;margin-bottom:18px;box-shadow:0 0 20px rgba(56,189,248,0.25)}
@keyframes hdev-spin{to{transform:rotate(360deg)}}
.hdev-badge{display:inline-flex;align-items:center;gap:6px;font-size:.72rem;font-weight:700;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;color:#38bdf8;background:rgba(6,182,212,0.12);border:1px solid rgba(6,182,212,0.3);padding:4px 10px;border-radius:9999px;margin-bottom:14px;letter-spacing:.04em}
.hdev-pulse-dot{width:6px;height:6px;border-radius:50%;background:#38bdf8;box-shadow:0 0 8px #38bdf8;animation:hdev-pulse 1.8s infinite ease-in-out}
@keyframes hdev-pulse{0%,100%{opacity:1;transform:scale(1)}50%{opacity:.4;transform:scale(.85)}}
.hydrogo-dev-card h2{margin:0 0 8px;font-size:1.45rem;font-weight:800;letter-spacing:-.025em;color:#fff}
.hydrogo-dev-card p{margin:0 0 24px;font-size:.92rem;color:#cbd5e1;line-height:1.55}
.hdev-bar-wrap{width:100%;height:5px;border-radius:99px;background:rgba(15,23,42,0.8);border:1px solid rgba(255,255,255,0.08);overflow:hidden;position:relative}
.hdev-bar-indeterminate{position:absolute;height:100%;background:linear-gradient(90deg,#06b6d4,#3b82f6,#8b5cf6);border-radius:99px;animation:hdev-progress 1.4s infinite ease-in-out;box-shadow:0 0 12px rgba(56,189,248,0.5)}
@keyframes hdev-progress{0%{left:-40%;width:40%}50%{left:20%;width:60%}100%{left:100%;width:40%}}

#hydrogo-dev-status{position:fixed;right:20px;bottom:20px;z-index:2147483647;width:min(330px,calc(100vw - 40px));color:#f8fafc;font:12px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;transition:opacity .2s ease,transform .2s ease}
#hydrogo-dev-status.hdev-hidden{opacity:0;pointer-events:none;transform:translateY(8px)}
.hdev-pill{display:flex;align-items:center;gap:8px;width:100%;padding:8px 10px;border:1px solid #30343b;border-radius:7px;background:#111318;color:#f5f5f5;box-shadow:0 5px 18px rgba(0,0,0,.22);cursor:pointer;text-align:left}
.hdev-pill:hover{background:#191c21;border-color:#4b5059}.hdev-pill:focus-visible,.hdev-close:focus-visible,.hdev-hide:focus-visible{outline:2px solid #fff;outline-offset:2px}
.hdev-dot{width:7px;height:7px;flex:0 0 auto;border-radius:50%;background:#a7f3d0}.hdev-busy .hdev-dot{background:#d4d4d8;animation:hdev-pulse 1.4s infinite ease-in-out}.hdev-error .hdev-dot{background:#ef4444}
.hdev-title{font-weight:700;font-size:11px;letter-spacing:.04em}.hdev-phase-tag{color:#a1a1aa;font-size:10px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.hdev-error .hdev-phase-tag{color:#f87171}.hdev-value{min-width:0;flex:1;color:#a1a1aa;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;font-size:11px}.hdev-chevron{color:#a1a1aa;font-size:16px;line-height:10px}.hdev-panel{margin-top:6px;padding:12px;border:1px solid #30343b;border-radius:7px;background:#111318;box-shadow:0 10px 28px rgba(0,0,0,.28)}.hdev-panel[hidden]{display:none}.hdev-panel-head{display:flex;align-items:center;justify-content:space-between;color:#f5f5f5}.hdev-close{border:0;background:transparent;color:#a1a1aa;font-size:20px;line-height:1;cursor:pointer}.hdev-message{margin:10px 0;color:#d4d4d8}.hdev-track{height:3px;overflow:hidden;border-radius:3px;background:#272a30}.hdev-track i{display:block;width:5%;height:100%;border-radius:inherit;background:#d4d4d8;transition:width .25s ease}.hdev-error .hdev-track i{background:#ef4444}.hdev-panel pre{max-height:220px;margin:10px 0 0;padding:10px;overflow:auto;border-radius:5px;background:#090a0c;color:#fca5a5;font:11px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre-wrap}.hdev-panel-actions{margin-top:10px}.hdev-hide{padding:0;border:0;background:transparent;color:#a1a1aa;font-size:11px;text-decoration:underline;text-underline-offset:3px;cursor:pointer}.hdev-hide:hover{color:#fff}`
