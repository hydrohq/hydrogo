package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func serveAutoTLS(ctx context.Context, config Config) error {
	host := strings.TrimSpace(config.TLSHost)
	if host == "" || strings.ContainsAny(host, "/:* ") || config.TLSCacheDir == "" {
		return errors.New("automatic TLS requires a hostname and persistent certificate cache")
	}
	if err := os.MkdirAll(filepath.Clean(config.TLSCacheDir), 0o700); err != nil {
		return err
	}
	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(config.TLSCacheDir),
		HostPolicy: func(_ context.Context, candidate string) error {
			if strings.EqualFold(candidate, host) {
				return nil
			}
			return errors.New("hostname is not configured for automatic TLS")
		},
	}
	app := &http.Server{
		Handler:           NewHandler(config),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	httpHandler := manager.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := url.URL{Scheme: "https", Host: host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
		http.Redirect(w, r, target.String(), http.StatusPermanentRedirect)
	}))
	challenge := &http.Server{Addr: ":80", Handler: httpHandler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	errCh := make(chan error, 2)
	go func() { errCh <- challenge.ListenAndServe() }()
	go func() { errCh <- app.ListenAndServeTLS("", "") }()
	select {
	case err := <-errCh:
		_ = challenge.Close()
		_ = app.Close()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = challenge.Shutdown(shutdownCtx)
		_ = app.Shutdown(shutdownCtx)
		for range 2 {
			if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
		return nil
	}
}
