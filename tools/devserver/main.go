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
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// devserver is meant to be left running for an entire workday, proxying a
// lot of HTTP/WebSocket traffic (admin Vite HMR, starter HMR, the dev
// status poll every 250ms-1.2s per open tab). Go's GC only returns memory
// to the OS lazily, so without a bound, RSS creeps upward over hours of
// that traffic instead of settling once warm. A soft memory limit (Go
// 1.19+) makes the GC work harder to stay under it instead of just
// growing; it's a ceiling, not a reservation, so it costs nothing when
// idle.
const softMemoryLimit = 768 << 20 // 768 MiB

const (
	statusPath = "/__hydrogo/dev/status"
	clientPath = "/__hydrogo/dev/client.js"
	stylePath  = "/__hydrogo/dev/client.css"
)

func main() {
	debug.SetMemoryLimit(softMemoryLimit)
	address := flag.String("address", "127.0.0.1:8090", "development proxy listen address (use 0.0.0.0:PORT to allow LAN access, e.g. from a phone on the same network)")
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
	status := &state{Phase: "starting", Progress: 5, Message: "Preparing development server…"}
	supervisor := &supervisor{root: root, binary: binary, appAddress: *appAddress, databaseDSN: *databaseDSN, redisURL: *redisURL, state: status}
	supervisor.rebuild(ctx, "initial build complete", true)
	if *seedDemo {
		status.set("seeding", 88, "Preparing demo data…", "")
		seed := exec.CommandContext(ctx, binary, "seed-demo", "--confirm-demo", "--database-dsn", *databaseDSN) // #nosec G204 -- developer tool; the commands are fixed and the arguments come from its own flags, no shell
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

	stopAdmin := startAdminFrontend(ctx, root)
	defer stopAdmin()

	stopStarter := startStarterFrontend(ctx, root)
	defer stopStarter()

	target, err := url.Parse("http://" + *appAddress)
	if err != nil {
		fatal(err)
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

	viteTarget, _ := url.Parse("http://127.0.0.1:5173")
	viteProxy := httputil.NewSingleHostReverseProxy(viteTarget)
	viteProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, _ error) {
		proxy.ServeHTTP(w, r)
	}

	starterTarget, _ := url.Parse("http://127.0.0.1:3000")
	starterProxy := httputil.NewSingleHostReverseProxy(starterTarget)
	starterProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, _ error) {
		proxy.ServeHTTP(w, r)
	}

	checkPort := func(port int) bool {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		return false
	}

	isAdminVitePath := func(path string) bool {
		return strings.HasPrefix(path, "/admin") ||
			strings.HasPrefix(path, "/admin/") ||
			strings.HasPrefix(path, "/@fs/") ||
			strings.HasPrefix(path, "/@id/")
	}

	isBackendPath := func(path string) bool {
		return strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/media/") ||
			strings.HasPrefix(path, "/auth/") ||
			strings.HasPrefix(path, "/member/") ||
			strings.HasPrefix(path, "/csrf-token") ||
			strings.HasPrefix(path, "/healthz") ||
			strings.HasPrefix(path, "/readyz") ||
			strings.HasPrefix(path, "/metrics") ||
			strings.HasPrefix(path, "/sitemap") ||
			strings.HasPrefix(path, "/llms")
	}

	holdWhileRebuilding := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin" {
			http.Redirect(w, r, "/admin/", http.StatusFound)
			return
		}
		if isAdminVitePath(r.URL.Path) && checkPort(5173) {
			viteProxy.ServeHTTP(w, r)
			return
		}
		if !isBackendPath(r.URL.Path) && checkPort(3000) {
			starterProxy.ServeHTTP(w, r)
			return
		}

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
	go func() {
		// Wait a moment for child Vite processes to finish initial logs, then display the unified gateway banner
		time.Sleep(1200 * time.Millisecond)
		fmt.Printf(`
┌──────────────────────────────────────────────────────────────────────────┐
│  🚀 HydroGo Tek Port Geliştirme Gateway'i Hazır!                        │
│  👉 http://%s                                                  │
│                                                                          │
│  • Ana Sayfa (Starter Vitrini):  http://%s/                     │
│  • Admin Studio (Yönetim):       http://%s/admin                │
│  • REST API:                     http://%s/api/v1/...           │
│                                                                          │
│  ℹ️ Vite :3000 ve :5173 portları iç Hot-Reload (HMR) servisleridir.      │
│     Doğrudan tek bir port üzerinden (8090) çalışabilirsiniz!             │
└──────────────────────────────────────────────────────────────────────────┘
`, *address, *address, *address, *address)
	}()
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
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

func startAdminFrontend(ctx context.Context, root string) func() {
	adminDir := filepath.Join(root, "admin")
	if info, err := os.Stat(filepath.Join(adminDir, "package.json")); err != nil || info.IsDir() {
		return func() {}
	}

	// Check if port 5173 is already in use
	conn, err := net.DialTimeout("tcp", "127.0.0.1:5173", 50*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		fmt.Println("hydro dev: existing Admin Vite dev server detected on 127.0.0.1:5173")
		return func() {}
	}

	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	if _, err := exec.LookPath(npm); err != nil {
		return func() {}
	}

	cmd := exec.CommandContext(ctx, npm, "run", "dev")
	cmd.Dir = adminDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "hydro dev: could not start admin Vite server: %v\n", err)
		return func() {}
	}
	fmt.Println("hydro dev: admin Vite dev server started (port 5173)")

	return func() {
		if cmd != nil && cmd.Process != nil {
			if runtime.GOOS == "windows" {
				_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run() // #nosec G204 -- developer tool; the commands are fixed and the arguments come from its own flags, no shell
			} else {
				_ = cmd.Process.Kill()
			}
			_, _ = cmd.Process.Wait()
		}
	}
}

func startStarterFrontend(ctx context.Context, root string) func() {
	starterDir := filepath.Join(root, "examples", "svelte-starter")
	if info, err := os.Stat(filepath.Join(starterDir, "package.json")); err != nil || info.IsDir() {
		return func() {}
	}

	// Check if port 3000 is already in use
	conn, err := net.DialTimeout("tcp", "127.0.0.1:3000", 50*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		fmt.Println("hydro dev: existing Starter dev server detected on 127.0.0.1:3000")
		return func() {}
	}

	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	if _, err := exec.LookPath(npm); err != nil {
		return func() {}
	}

	cmd := exec.CommandContext(ctx, npm, "run", "dev")
	cmd.Dir = starterDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "hydro dev: could not start starter Vite server: %v\n", err)
		return func() {}
	}
	fmt.Println("hydro dev: starter Vite dev server started (port 3000)")

	return func() {
		if cmd != nil && cmd.Process != nil {
			if runtime.GOOS == "windows" {
				_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run() // #nosec G204 -- developer tool; the commands are fixed and the arguments come from its own flags, no shell
			} else {
				_ = cmd.Process.Kill()
			}
			_, _ = cmd.Process.Wait()
		}
	}
}
