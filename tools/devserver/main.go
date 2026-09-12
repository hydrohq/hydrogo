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
	root, binary, appAddress, databaseDSN string
	state                                 *state
	mu                                    sync.Mutex
	child                                 *exec.Cmd
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
	poll := flag.Duration("poll", 600*time.Millisecond, "file watch polling interval")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if *databaseDSN == "" {
		*databaseDSN = database.SQLiteDSN(filepath.Join(root, "data", "demo.sqlite"))
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
	supervisor := &supervisor{root: root, binary: binary, appAddress: *appAddress, databaseDSN: *databaseDSN, state: status}
	supervisor.rebuild(ctx, "ilk build tamamlandı", true)
	defer supervisor.stop()

	target, err := url.Parse("http://" + *appAddress)
	if err != nil {
		fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, proxyErr error) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Yeniden Başlatılıyor</title><link rel="stylesheet" href="`+stylePath+`"><script src="`+clientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card"><div class="hdev-spinner"></div><h2>Sunucu Yeniden Başlatılıyor</h2><p>Değişiklikler derleniyor ve servis hazırlanıyor…</p><div class="hdev-bar-wrap"><div class="hdev-bar-indeterminate"></div></div></main></body></html>`)
	}
	proxy.ModifyResponse = injectClient
	mux := http.NewServeMux()
	mux.HandleFunc(statusPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(status.snapshot())
	})
	mux.HandleFunc(clientPath, serveClient)
	mux.HandleFunc(stylePath, serveStyles)
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

	server := &http.Server{Addr: *address, Handler: noStore(mux), ReadHeaderTimeout: 5 * time.Second}
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

const devClient = `(()=>{if(window.__hydrogoDev)return;window.__hydrogoDev=true;const link=document.createElement('link');link.rel='stylesheet';link.href='` + stylePath + `';document.head.append(link);const box=document.createElement('aside');box.id='hydrogo-dev-status';box.innerHTML='<div class="hdev-head"><strong>HydroGo Dev</strong><span class="hdev-value">Başlatılıyor</span></div><div class="hdev-track"><i></i></div><pre hidden></pre>';document.documentElement.append(box);const value=box.querySelector('.hdev-value'),bar=box.querySelector('i'),details=box.querySelector('pre');let revision=-1,initialized=false,reloadOnReady=false,hideTimer;const show=()=>{clearTimeout(hideTimer);box.classList.remove('hdev-hidden')},hide=()=>{clearTimeout(hideTimer);hideTimer=setTimeout(()=>box.classList.add('hdev-hidden'),900)};async function poll(){try{const response=await fetch('` + statusPath + `',{cache:'no-store'});if(!response.ok){show();value.textContent='Sunucu hazırlanıyor…';if(initialized)reloadOnReady=true;return;}const status=await response.json();value.textContent=status.message+' · '+status.progress+'%';bar.style.width=status.progress+'%';box.classList.toggle('hdev-error',status.phase==='error');details.hidden=!status.error;details.textContent=status.error||'';if(status.phase==='ready'&&!status.error)hide();else show();if(initialized&&status.revision!==revision){if(status.phase==='ready'){setTimeout(()=>location.reload(),180);return}reloadOnReady=true}if(initialized&&reloadOnReady&&status.phase==='ready'){setTimeout(()=>location.reload(),180);return}initialized=true;revision=status.revision}catch(_){show();value.textContent='Yeniden bağlanılıyor…';if(initialized)reloadOnReady=true;}}poll();setInterval(poll,600)})();`

const devStyles = `body.hydrogo-dev-body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#090d16;color:#f8fafc;font-family:'Plus Jakarta Sans',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;-webkit-font-smoothing:antialiased;padding:24px;box-sizing:border-box}
.hydrogo-dev-card{display:flex;flex-direction:column;align-items:center;text-align:center;padding:36px 32px 32px;max-width:380px;width:100%;background:#131d31;border:1px solid #1e293b;border-radius:20px;box-shadow:0 24px 50px rgba(0,0,0,.45)}
.hdev-spinner{width:36px;height:36px;border:3px solid rgba(37,99,235,0.18);border-top-color:#3b82f6;border-radius:50%;animation:hdev-spin .85s cubic-bezier(.5,.1,.4,.9) infinite;margin-bottom:18px}
@keyframes hdev-spin{to{transform:rotate(360deg)}}
.hydrogo-dev-card h2{margin:0 0 8px;font-size:1.15rem;font-weight:700;letter-spacing:-.02em;color:#f8fafc}
.hydrogo-dev-card p{margin:0 0 20px;font-size:.86rem;color:#94a3b8;line-height:1.5}
.hdev-bar-wrap{width:100%;height:4px;border-radius:99px;background:#1e293b;overflow:hidden;position:relative}
.hdev-bar-indeterminate{position:absolute;height:100%;background:linear-gradient(90deg,#2563eb,#60a5fa);border-radius:99px;animation:hdev-progress 1.4s infinite ease-in-out}
@keyframes hdev-progress{0%{left:-40%;width:40%}50%{left:20%;width:60%}100%{left:100%;width:40%}}
#hydrogo-dev-status{position:fixed;right:16px;bottom:16px;z-index:2147483647;width:min(340px,calc(100vw - 32px));padding:10px 14px;border:1px solid #1e293b;border-radius:12px;background:#131d31;color:#f8fafc;box-shadow:0 12px 30px rgba(0,0,0,.35);font:12px/1.45 'Plus Jakarta Sans',sans-serif;transition:all .2s ease}
#hydrogo-dev-status.hdev-hidden{opacity:0;pointer-events:none;transform:translateY(8px)}
.hdev-head{display:flex;justify-content:space-between;align-items:center;gap:12px}
.hdev-head strong{color:#60a5fa;font-weight:700;font-size:.82rem;letter-spacing:-.01em}
.hdev-value{color:#94a3b8;font-size:.78rem;font-weight:500}
.hdev-track{height:3px;margin-top:8px;overflow:hidden;border-radius:99px;background:#1e293b}
.hdev-track i{display:block;width:5%;height:100%;border-radius:inherit;background:linear-gradient(90deg,#2563eb,#60a5fa);transition:width .25s ease}
#hydrogo-dev-status.hdev-error{border-color:rgba(239,68,68,0.4);background:#1a1114}
#hydrogo-dev-status.hdev-error .hdev-head strong,#hydrogo-dev-status.hdev-error .hdev-value{color:#f87171}
#hydrogo-dev-status pre{max-height:200px;margin:8px 0 0;padding:10px;overflow:auto;border-radius:6px;background:#090d16;color:#fca5a5;font:11px/1.45 'JetBrains Mono',monospace;white-space:pre-wrap;border:1px solid rgba(239,68,68,0.2)}`
