package storefront

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hydrohq/hydrogo/internal/config"
)

// ExternalStorefront bridges HydroGo with an external headless/SSR storefront
// (such as SvelteKit, Next.js, Nuxt, Astro).
// It directly serves static client assets with precompression and immutable caching,
// supervises the Node/SSR child process (if Command is configured),
// and reverse-proxies dynamic SSR requests to the upstream server or UNIX domain socket.
type ExternalStorefront struct {
	cfg          config.ExternalStorefrontConfig
	logger       *slog.Logger
	proxy        *httputil.ReverseProxy
	upstreamURL  *url.URL
	isUnixSocket bool
	socketPath   string
	staticDir    string
}

// NewExternalStorefront creates an ExternalStorefront handler.
func NewExternalStorefront(cfg config.ExternalStorefrontConfig, logger *slog.Logger) (*ExternalStorefront, error) {
	if logger == nil {
		logger = slog.Default()
	}

	upstreamStr := strings.TrimSpace(cfg.Upstream)
	if upstreamStr == "" {
		if port, ok := cfg.Env["PORT"]; ok && port != "" {
			upstreamStr = "http://127.0.0.1:" + port
		} else {
			upstreamStr = "http://127.0.0.1:3000"
		}
	}

	es := &ExternalStorefront{
		cfg:       cfg,
		logger:    logger,
		staticDir: cfg.StaticDir,
	}

	var transport *http.Transport
	var targetURL *url.URL

	if strings.HasPrefix(upstreamStr, "unix:") {
		es.isUnixSocket = true
		es.socketPath = strings.TrimPrefix(upstreamStr, "unix:")
		targetURL = &url.URL{Scheme: "http", Host: "unix"}
		transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", es.socketPath)
			},
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
		}
	} else {
		u, err := url.Parse(upstreamStr)
		if err != nil {
			return nil, fmt.Errorf("invalid storefront upstream URL %q: %w", upstreamStr, err)
		}
		targetURL = u
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	es.upstreamURL = targetURL
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = transport
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		if h := req.Header.Get("Host"); h != "" {
			req.Host = h
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.WarnContext(r.Context(), "storefront upstream unavailable", "error", err, "path", r.URL.Path)
		if strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"storefront starting up","retry_after":3}`))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(getStartingHTML())
	}

	es.proxy = proxy
	return es, nil
}

// ServeHTTP handles storefront requests: static files are served directly with
// ultra-low latency and caching; dynamic page requests are proxied to Upstream.
func (s *ExternalStorefront) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.staticDir != "" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		if s.tryServeStatic(w, r) {
			return
		}
	}
	// Missing files under /_app/ are missing static bundles. Return 404 immediately.
	if strings.HasPrefix(r.URL.Path, "/_app/") {
		http.NotFound(w, r)
		return
	}
	s.proxy.ServeHTTP(w, r)
}

func (s *ExternalStorefront) tryServeStatic(w http.ResponseWriter, r *http.Request) bool {
	urlPath := r.URL.Path
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	cleaned := path.Clean(urlPath)
	if cleaned == "/" || cleaned == "." {
		return false
	}

	rel := strings.TrimPrefix(cleaned, "/")
	diskPath := filepath.Join(s.staticDir, filepath.FromSlash(rel))

	cleanStaticDir := filepath.Clean(s.staticDir)
	if !strings.HasPrefix(diskPath, cleanStaticDir) {
		return false
	}

	stat, err := os.Stat(diskPath)
	if err != nil || stat.IsDir() {
		return false
	}

	if strings.HasPrefix(cleaned, "/_app/immutable/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	acceptEncoding := r.Header.Get("Accept-Encoding")
	// 1. Try Brotli precompressed asset
	if strings.Contains(acceptEncoding, "br") {
		brPath := diskPath + ".br"
		if brStat, err := os.Stat(brPath); err == nil && !brStat.IsDir() {
			brFile, err := os.Open(brPath)
			if err == nil {
				defer brFile.Close()
				s.setMimeType(w, diskPath)
				w.Header().Set("Content-Encoding", "br")
				w.Header().Set("Vary", "Accept-Encoding")
				http.ServeContent(w, r, filepath.Base(diskPath), brStat.ModTime(), brFile)
				return true
			}
		}
	}

	// 2. Try Gzip precompressed asset
	if strings.Contains(acceptEncoding, "gzip") {
		gzPath := diskPath + ".gz"
		if gzStat, err := os.Stat(gzPath); err == nil && !gzStat.IsDir() {
			gzFile, err := os.Open(gzPath)
			if err == nil {
				defer gzFile.Close()
				s.setMimeType(w, diskPath)
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Vary", "Accept-Encoding")
				http.ServeContent(w, r, filepath.Base(diskPath), gzStat.ModTime(), gzFile)
				return true
			}
		}
	}

	http.ServeFile(w, r, diskPath)
	return true
}

func (s *ExternalStorefront) setMimeType(w http.ResponseWriter, filePath string) {
	ext := filepath.Ext(filePath)
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
		return
	}
	switch ext {
	case ".js", ".mjs":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	}
}

// RunSupervisor runs the child process supervisor in a loop, auto-restarting
// on crash and terminating cleanly when ctx is cancelled.
func (s *ExternalStorefront) RunSupervisor(ctx context.Context) {
	if strings.TrimSpace(s.cfg.Command) == "" {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := s.runChildProcess(ctx)
		if ctx.Err() != nil {
			return
		}

		s.logger.WarnContext(ctx, "storefront child process exited, restarting in 2 seconds", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *ExternalStorefront) runChildProcess(ctx context.Context) error {
	parts := strings.Fields(s.cfg.Command)
	if len(parts) == 0 {
		return errors.New("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	if s.cfg.Dir != "" {
		cmd.Dir = s.cfg.Dir
	}

	cmd.Env = os.Environ()

	if s.cfg.EnvFile != "" {
		if fileEnv, err := parseEnvFile(s.cfg.EnvFile); err == nil {
			for k, v := range fileEnv {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		} else {
			s.logger.WarnContext(ctx, "failed to read storefront env file", "path", s.cfg.EnvFile, "error", err)
		}
	}

	for k, v := range s.cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if s.isUnixSocket && s.socketPath != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SOCKET_PATH=%s", s.socketPath))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start storefront process: %w", err)
	}

	s.logger.InfoContext(ctx, "storefront child process started", "pid", cmd.Process.Pid, "command", s.cfg.Command)

	go s.streamLogs(stdout, "stdout")
	go s.streamLogs(stderr, "stderr")

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("stopping storefront child process...")
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-waitCh:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-waitCh
		}
		return nil
	case err := <-waitCh:
		return err
	}
}

func (s *ExternalStorefront) streamLogs(r io.Reader, stream string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		s.logger.Info(line, "source", "storefront", "stream", stream)
	}
}

func parseEnvFile(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			result[key] = val
		}
	}
	return result, scanner.Err()
}

var (
	startingHTML     []byte
	startingHTMLOnce sync.Once
)

func getStartingHTML() []byte {
	startingHTMLOnce.Do(func() {
		data, err := templateFS.ReadFile("storefront_starting.html")
		if err == nil {
			startingHTML = data
		} else {
			startingHTML = []byte("Storefront starting up...")
		}
	})
	return startingHTML
}
