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
		_, _ = io.WriteString(w, `<!doctype html><title>HydroGo yeniden başlıyor</title><script src="`+clientPath+`"></script><main style="font:16px system-ui;padding:3rem">HydroGo geliştirici sunucusu yeniden başlıyor…</main>`)
	}
	proxy.ModifyResponse = injectClient
	mux := http.NewServeMux()
	mux.HandleFunc(statusPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(status.snapshot())
	})
	mux.HandleFunc(clientPath, serveClient)
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

const devClient = `(()=>{if(window.__hydrogoDev)return;window.__hydrogoDev=true;const box=document.createElement('aside');box.id='hydrogo-dev-status';box.innerHTML='<div class="hdev-head"><strong>HydroGo Dev</strong><span class="hdev-value">Başlatılıyor</span></div><div class="hdev-track"><i></i></div><pre hidden></pre>';document.documentElement.append(box);const style=document.createElement('style');style.textContent='#hydrogo-dev-status{position:fixed;right:16px;bottom:16px;z-index:2147483647;width:min(440px,calc(100vw - 32px));padding:12px 14px;border:1px solid #cbd5e1;border-radius:12px;background:#fff;color:#0f172a;box-shadow:0 12px 32px rgba(15,23,42,.18);font:12px/1.45 ui-sans-serif,system-ui,sans-serif}.hdev-head{display:flex;justify-content:space-between;gap:12px}.hdev-head strong{color:#4f46e5}.hdev-value{color:#475569}.hdev-track{height:4px;margin-top:9px;overflow:hidden;border-radius:99px;background:#e2e8f0}.hdev-track i{display:block;width:5%;height:100%;border-radius:inherit;background:#4f46e5;transition:width .25s ease}#hydrogo-dev-status.hdev-error{border-color:#fecaca;background:#fff7f7}#hydrogo-dev-status.hdev-error .hdev-head strong,#hydrogo-dev-status.hdev-error .hdev-value{color:#b91c1c}#hydrogo-dev-status pre{max-height:220px;margin:10px 0 0;padding:10px;overflow:auto;border-radius:8px;background:#1e293b;color:#f8fafc;font:11px/1.45 ui-monospace,monospace;white-space:pre-wrap}';document.documentElement.append(style);const value=box.querySelector('.hdev-value'),bar=box.querySelector('i'),details=box.querySelector('pre');let revision=-1,initialized=false,reloadOnReady=false;async function poll(){try{const response=await fetch('` + statusPath + `',{cache:'no-store'});const status=await response.json();value.textContent=status.message+' · '+status.progress+'%';bar.style.width=status.progress+'%';box.classList.toggle('hdev-error',status.phase==='error');details.hidden=!status.error;details.textContent=status.error||'';if(initialized&&status.revision!==revision){if(status.phase==='ready'){setTimeout(()=>location.reload(),180);return}reloadOnReady=true}if(initialized&&reloadOnReady&&status.phase==='ready'){setTimeout(()=>location.reload(),180);return}initialized=true;revision=status.revision}catch(_){value.textContent='Proxy bağlantısı bekleniyor…'}}poll();setInterval(poll,700)})();`
