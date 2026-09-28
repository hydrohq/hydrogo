package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const shutdownTimeout = 10 * time.Second

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

type Config struct {
	Address                    string
	AutoTLS                    bool
	TLSHost                    string
	TLSCacheDir                string
	ShutdownTimeout            time.Duration
	Ready                      func(context.Context) error
	AuthSvc                    *auth.AuthService
	Routes                     []router.Route
	Authorizer                 router.Authorizer
	Metrics                    *observability.Metrics
	Logger                     *slog.Logger
	EnableHSTS                 bool
	SecurityHeadersPreset      string
	AdminSupport               adminstudio.SupportConfig
	IPBlocklist                *security.IPBlocklist
	AnomalyFilterEnabled       bool
	AnomalyBypassPaths         []string
	OnSecurityAnomaly          func(*http.Request, string)
	OnPanic                    func(context.Context, *http.Request, string, []byte)
	OnHTTPError                func(*http.Request, int)
	SecureCookie               bool
	HTTPBodyLimit              int64
	PublicOrigin               string
	PublicAPIOrigins           []string
	TenantResolver             *tenant.HostResolver
	TenantAdmission            *security.TenantAdmission
	DistributedTenantAdmission *security.RedisTenantAdmission
	RealtimeHub                *realtime.Hub
}

func Serve(ctx context.Context, config Config) error {
	ctx = nonNilContext(ctx)
	if config.AutoTLS {
		return serveAutoTLS(ctx, config)
	}
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
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		_, _ = w.Write([]byte("ok\n"))
	})
	readiness := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		if config.Ready != nil {
			checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := config.Ready(checkCtx); err != nil {
				if config.Metrics != nil {
					config.Metrics.RecordReadinessFailure()
					config.Metrics.SetReadinessState(false)
				}
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
		}
		if config.Metrics != nil {
			config.Metrics.SetReadinessState(true)
		}
		_, _ = w.Write([]byte("ready\n"))
	})

	routes := []router.Route{
		{Pattern: "GET /healthz", Access: router.Public, Handler: health},
		{Pattern: "GET /readyz", Access: router.Public, Handler: readiness},
		{Pattern: "POST /csp-report", Access: router.Public, Handler: http.HandlerFunc(security.CSPReportHandler)},
	}
	routes = append(routes, config.Routes...)
	if config.RealtimeHub != nil {
		routes = append(routes, router.Route{Pattern: "GET /api/v1/realtime/subscriptions", Access: router.Protected, Handler: config.RealtimeHub})
	}

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
	handler = CompressionMiddleware(handler)
	handler = ETagMiddleware(handler)
	hostMiddleware, err := security.NewHostMiddleware(config.PublicOrigin)
	if err != nil {
		panic(err)
	}
	handler = hostMiddleware(handler)
	if config.TenantResolver != nil && config.TenantResolver.Enabled() {
		handler = config.TenantResolver.Middleware(handler)
	}
	if config.DistributedTenantAdmission != nil {
		handler = config.DistributedTenantAdmission.Middleware(handler)
	} else if config.TenantAdmission != nil {
		handler = config.TenantAdmission.Middleware(handler)
	}
	publicCORS, err := security.NewPublicReadCORSMiddleware(config.PublicAPIOrigins)
	if err != nil {
		panic(err)
	}
	handler = publicCORS(handler)

	// General requests use the selected capacity profile budget. Media uploads
	// enforce their larger, uploader-specific limit in the media handler.
	handler = security.RequestBodyLimitMiddleware(config.HTTPBodyLimit, func(r *http.Request) bool {
		return r.URL.Path == "/api/media" || r.URL.Path == "/api/v1/media"
	})(handler)
	handler = security.AnomalyMiddleware(security.AnomalyConfig{
		Enabled: config.AnomalyFilterEnabled, MaxBodyBytes: config.HTTPBodyLimit,
		BypassPaths: config.AnomalyBypassPaths, OnDetect: config.OnSecurityAnomaly,
	})(handler)
	handler = security.RecoveryMiddlewareWithObserver(config.Logger, config.OnPanic)(handler)
	handler = observability.HTTPErrorMiddleware(config.OnHTTPError)(handler)

	// Normalize handler errors after the body limit and before the request ID
	// middleware so even middleware-generated errors include request_id.
	handler = security.ErrorMiddleware(handler)
	handler = observability.LoggingMiddleware(config.Logger)(handler)
	// Trace context is optional/no-op until an operator configures an exporter.
	// It is wrapped before RequestID below so the request correlation ID is
	// available as a bounded trace attribute.
	handler = observability.TracingMiddleware(handler)

	// Generate a fresh correlation identifier before security and auth middleware.
	handler = security.RequestIDMiddleware(handler)

	// Apply CSRF protection
	handler = security.CSRFMiddleware(config.SecureCookie)(handler)

	// Apply Auth session context injection
	if config.AuthSvc != nil {
		handler = auth.Middleware(config.AuthSvc, config.SecureCookie)(handler)
	}
	if config.IPBlocklist != nil {
		handler = config.IPBlocklist.Middleware(handler)
	}

	// Apply Security Headers with CSP nonces
	handler = security.HeadersMiddleware(security.HeaderConfig{
		EnableHSTS: config.EnableHSTS,
		Preset:     config.SecurityHeadersPreset,
	})(handler)

	// Keep metrics outermost so rejected requests from every middleware layer
	// (body limit, CSRF, auth, and routing) are visible to operators.
	if config.Metrics != nil {
		handler = config.Metrics.Middleware(handler)
	}
	// Final response guard: no HTML comment is allowed into production output.
	handler = observability.HTMLCommentMiddleware(handler)
	handler = adminstudio.SupportConfigMiddleware(config.AdminSupport)(handler)

	return handler
}
