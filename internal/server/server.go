package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

const shutdownTimeout = 10 * time.Second

type Config struct {
	Address         string
	ShutdownTimeout time.Duration
	Ready           func(context.Context) error
	AuthSvc         *auth.AuthService
	Routes          []router.Route
	Authorizer      router.Authorizer
	Metrics         *observability.Metrics
	EnableHSTS      bool
	SecureCookie    bool
}

func Serve(ctx context.Context, config Config) error {
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{
		Handler:           NewHandler(config),
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
		timeout := config.ShutdownTimeout
		if timeout <= 0 {
			timeout = shutdownTimeout
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
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

func NewHandler(config Config) http.Handler {
	health := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	readiness := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if config.Ready != nil {
			checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := config.Ready(checkCtx); err != nil {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
		}
		_, _ = w.Write([]byte("ready\n"))
	})

	routes := []router.Route{
		{Pattern: "GET /healthz", Access: router.Public, Handler: health},
		{Pattern: "GET /readyz", Access: router.Public, Handler: readiness},
	}
	routes = append(routes, config.Routes...)

	authorizer := config.Authorizer
	if authorizer == nil && config.AuthSvc != nil {
		authorizer = auth.Authorizer()
	}
	if config.Metrics != nil && authorizer != nil {
		routes = append(routes, router.Route{Pattern: "GET /metrics", Access: router.Protected, Handler: http.HandlerFunc(config.Metrics.Handler)})
	}

	routerHandler, err := router.New(routes, authorizer)
	if err != nil {
		panic(err)
	}

	var handler http.Handler = routerHandler
	if config.Metrics != nil {
		handler = config.Metrics.Middleware(handler)
	}

	// Generate a fresh correlation identifier before security and auth middleware.
	handler = security.RequestIDMiddleware(handler)

	// Apply CSRF protection
	handler = security.CSRFMiddleware(config.SecureCookie)(handler)

	// Apply Auth session context injection
	if config.AuthSvc != nil {
		handler = auth.Middleware(config.AuthSvc)(handler)
	}

	// Apply Security Headers with CSP nonces
	handler = security.HeadersMiddleware(security.HeaderConfig{
		EnableHSTS: config.EnableHSTS,
	})(handler)

	return handler
}
