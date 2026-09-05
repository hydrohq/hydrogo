package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

const shutdownTimeout = 10 * time.Second

// Serve starts the development HTTP skeleton. Only explicitly registered
// operational routes are reachable; feature routes are added by modules later.
type Config struct {
	Address string
	Ready   func(context.Context) error
}

func Serve(ctx context.Context, config Config) error {
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{
		Handler:           newHandler(config.Ready),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func newHandler(ready func(context.Context) error) http.Handler {
	health := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	readiness := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if ready != nil {
			checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ready(checkCtx); err != nil {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
		}
		_, _ = w.Write([]byte("ready\n"))
	})
	handler, err := router.New([]router.Route{
		{Pattern: "GET /healthz", Access: router.Public, Handler: health},
		{Pattern: "GET /readyz", Access: router.Public, Handler: readiness},
	}, nil)
	if err != nil {
		panic(err)
	}
	return handler
}
