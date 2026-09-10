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
	if rebuild {
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
		s.state.set("error", 100, "Migration başarısız; çalışan sürüm korunuyor.", shorten(migrationOutput.String(), 12000))
		return
	}

	s.state.set("restarting", 75, "Uygulama yeniden başlatılıyor…", "")
	s.mu.Lock()
	if s.child != nil && s.child.Process != nil {
		_ = s.child.Process.Kill()
		_, _ = s.child.Process.Wait()
	}
	child := exec.Command(s.binary, "serve", "--address", s.appAddress, "--database-dsn", s.databaseDSN)
	child.Dir = s.root
	child.Env = append(os.Environ(), "HYDRO_DEV_MODE=1")
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		s.mu.Unlock()
		s.state.set("error", 100, "Uygulama başlatılamadı.", err.Error())
		return
	}
	s.child = child
	s.mu.Unlock()

	go func(process *exec.Cmd) {
		_ = process.Wait()
	}(child)

	// Ping internal app address until it responds or context times out
	ready := false
	pingClient := &http.Client{Timeout: 300 * time.Millisecond}
	for i := 0; i < 40; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, pingErr := pingClient.Get("http://" + s.appAddress + "/healthz")
		if pingErr == nil {
			_ = resp.Body.Close()
			ready = true
			break
		}
	}
	if !ready {
		// Even if /healthz isn't standard, try root
		resp, pingErr := pingClient.Get("http://" + s.appAddress + "/")
		if pingErr == nil {
			_ = resp.Body.Close()
			ready = true
		}
	}

	s.state.set("ready", 100, "Hazır · "+reason, "")
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
		_, _ = io.WriteString(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo · Yeniden Başlatılıyor</title><link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin><link rel="preload" as="style" href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&amp;family=JetBrains+Mono:wght@400;500;600&amp;display=block"><link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&amp;family=JetBrains+Mono:wght@400;500;600&amp;display=block"><link rel="stylesheet" href="`+stylePath+`"><script src="`+clientPath+`"></script></head><body class="hydrogo-dev-body"><main class="hydrogo-dev-card"><div class="hdev-logo-mark"><svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><div class="hdev-badge"><span class="hdev-live-dot"></span><span>Geliştirme Sunucusu</span></div><div class="hdev-wait-text"><h2>Sunucu Yeniden Başlatılıyor</h2><p>Değişiklikler derleniyor, veritabanı migration'ları senkronize ediliyor ve servis ayağa kaldırılıyor…</p></div><div class="hdev-status-box"><div class="hdev-steps"><div class="hdev-step active"><span class="step-num">1</span><span>Derleme (Build)</span></div><div class="hdev-step-sep"></div><div class="hdev-step active"><span class="step-num">2</span><span>Migration</span></div><div class="hdev-step-sep"></div><div class="hdev-step active"><span class="step-num">3</span><span>Başlatma</span></div></div><div class="hdev-bar-wrap"><div class="hdev-bar-indeterminate"></div></div><div class="hdev-footer-info"><span class="hdev-reloading-msg">Servis yanıt verdiğinde sayfa otomatik yenilenecek</span><span class="hdev-port-badge">127.0.0.1:8090</span></div></div></main></body></html>`)
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
		status.set("ready", 100, "Asset güncellendi: "+change.path, "")
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

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "hydro dev:", err)
	os.Exit(1)
}

const devClient = `(()=>{if(window.__hydrogoDev)return;window.__hydrogoDev=true;const link=document.createElement('link');link.rel='stylesheet';link.href='` + stylePath + `';document.head.append(link);const box=document.createElement('aside');box.id='hydrogo-dev-status';box.innerHTML='<div class="hdev-head"><strong>HydroGo Dev</strong><span class="hdev-value">Başlatılıyor</span></div><div class="hdev-track"><i></i></div><pre hidden></pre>';document.documentElement.append(box);const value=box.querySelector('.hdev-value'),bar=box.querySelector('i'),details=box.querySelector('pre');let revision=-1,initialized=false,reloadOnReady=false,hideTimer;const show=()=>{clearTimeout(hideTimer);box.classList.remove('hdev-hidden')},hide=()=>{clearTimeout(hideTimer);hideTimer=setTimeout(()=>box.classList.add('hdev-hidden'),900)};async function poll(){try{const response=await fetch('` + statusPath + `',{cache:'no-store'});if(!response.ok){show();value.textContent='Sunucu hazırlanıyor…';if(initialized)reloadOnReady=true;return;}const status=await response.json();value.textContent=status.message+' · '+status.progress+'%';bar.style.width=status.progress+'%';box.classList.toggle('hdev-error',status.phase==='error');details.hidden=!status.error;details.textContent=status.error||'';if(status.phase==='ready'&&!status.error)hide();else show();if(initialized&&status.revision!==revision){if(status.phase==='ready'){setTimeout(()=>location.reload(),180);return}reloadOnReady=true}if(initialized&&reloadOnReady&&status.phase==='ready'){setTimeout(()=>location.reload(),180);return}initialized=true;revision=status.revision}catch(_){show();value.textContent='Yeniden bağlanılıyor…';if(initialized)reloadOnReady=true;}}poll();setInterval(poll,600)})();`

const devStyles = `body.hydrogo-dev-body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:radial-gradient(circle at 50% 20%,#131d31 0%,#090d16 100%);color:#f8fafc;font-family:'Plus Jakarta Sans',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;-webkit-font-smoothing:antialiased;padding:20px;box-sizing:border-box}
.hydrogo-dev-card{display:flex;flex-direction:column;align-items:center;text-align:center;padding:42px 36px 36px;max-width:480px;width:100%;background:rgba(19,29,49,0.85);backdrop-filter:blur(16px);border:1px solid rgba(255,255,255,0.08);border-radius:24px;box-shadow:0 30px 70px rgba(0,0,0,.6),0 0 0 1px rgba(99,102,241,0.15);position:relative;overflow:hidden}
.hydrogo-dev-card::before{content:"";position:absolute;top:0;left:0;right:0;height:3px;background:linear-gradient(90deg,#6366f1,#818cf8,#a855f7)}
.hdev-logo-mark{display:grid;place-items:center;width:54px;height:54px;border-radius:16px;background:linear-gradient(135deg,#6366f1,#4f46e5);color:#fff;box-shadow:0 8px 24px rgba(79,70,229,0.4);margin-bottom:18px}
.hdev-badge{display:inline-flex;align-items:center;gap:7px;padding:4px 12px;background:rgba(99,102,241,0.12);border:1px solid rgba(99,102,241,0.25);border-radius:999px;color:#818cf8;font-size:.76rem;font-weight:700;letter-spacing:.05em;text-transform:uppercase;margin-bottom:14px}
.hdev-live-dot{width:7px;height:7px;border-radius:50%;background:#818cf8;box-shadow:0 0 0 0 rgba(129,140,248,0.6);animation:hdev-pulse 1.8s infinite}
.hdev-wait-text h2{margin:0 0 10px;font-size:1.35rem;font-weight:800;letter-spacing:-.03em;color:#f8fafc;line-height:1.25}
.hdev-wait-text p{margin:0 0 28px;font-size:.9rem;color:#94a3b8;line-height:1.6}
.hdev-status-box{width:100%;display:grid;gap:16px;background:rgba(9,13,22,0.6);border:1px solid rgba(255,255,255,0.06);border-radius:16px;padding:18px}
.hdev-steps{display:flex;align-items:center;justify-content:space-between;gap:8px}
.hdev-step{display:flex;align-items:center;gap:6px;font-size:.76rem;font-weight:700;color:#94a3b8}
.hdev-step.active{color:#818cf8}
.hdev-step .step-num{display:grid;place-items:center;width:18px;height:18px;border-radius:50%;background:rgba(99,102,241,0.2);color:#818cf8;font-size:.7rem;font-weight:800}
.hdev-step-sep{flex:1;height:1px;background:rgba(255,255,255,0.08)}
.hdev-bar-wrap{width:100%;height:6px;border-radius:99px;background:rgba(255,255,255,0.06);overflow:hidden;position:relative}
.hdev-bar-indeterminate{position:absolute;height:100%;background:linear-gradient(90deg,#6366f1,#818cf8,#a855f7);border-radius:99px;animation:hdev-progress 1.5s infinite ease-in-out}
.hdev-footer-info{display:flex;justify-content:space-between;align-items:center;gap:12px;font-size:.78rem;color:#64748b}
.hdev-reloading-msg{color:#94a3b8;font-weight:500;text-align:left}
.hdev-port-badge{font-family:'JetBrains Mono',monospace;font-size:.72rem;background:rgba(255,255,255,0.05);padding:3px 7px;border-radius:6px;border:1px solid rgba(255,255,255,0.08);color:#818cf8}
@keyframes hdev-pulse{0%{box-shadow:0 0 0 0 rgba(129,140,248,0.6)}70%{box-shadow:0 0 0 7px rgba(129,140,248,0)}100%{box-shadow:0 0 0 0 rgba(129,140,248,0)}}
@keyframes hdev-progress{0%{left:-40%;width:40%}50%{left:20%;width:60%}100%{left:100%;width:40%}}
#hydrogo-dev-status{position:fixed;right:18px;bottom:18px;z-index:2147483647;width:min(400px,calc(100vw - 36px));padding:14px 16px;border:1px solid rgba(99,102,241,0.25);border-radius:14px;background:#131d31;color:#f8fafc;box-shadow:0 16px 40px rgba(0,0,0,.5);font:13px/1.45 'Plus Jakarta Sans',sans-serif;transition:all .2s ease;backdrop-filter:blur(10px)}
#hydrogo-dev-status.hdev-hidden{opacity:0;pointer-events:none;transform:translateY(10px)}
.hdev-head{display:flex;justify-content:space-between;align-items:center;gap:12px}
.hdev-head strong{color:#818cf8;font-weight:800;font-size:.85rem;letter-spacing:-.01em}
.hdev-value{color:#94a3b8;font-size:.8rem;font-weight:600}
.hdev-track{height:5px;margin-top:10px;overflow:hidden;border-radius:99px;background:rgba(255,255,255,0.08)}
.hdev-track i{display:block;width:5%;height:100%;border-radius:inherit;background:linear-gradient(90deg,#6366f1,#818cf8);transition:width .25s ease}
#hydrogo-dev-status.hdev-error{border-color:rgba(239,68,68,0.5);background:#1a1114}
#hydrogo-dev-status.hdev-error .hdev-head strong,#hydrogo-dev-status.hdev-error .hdev-value{color:#f87171}
#hydrogo-dev-status pre{max-height:220px;margin:10px 0 0;padding:12px;overflow:auto;border-radius:8px;background:#090d16;color:#fca5a5;font:11px/1.45 'JetBrains Mono',monospace;white-space:pre-wrap;border:1px solid rgba(239,68,68,0.2)}`
