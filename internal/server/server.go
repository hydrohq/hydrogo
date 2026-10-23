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
	SecurityScriptHashes       []string
	SecurityStyleHashes        []string
	AdminSupport               adminstudio.SupportConfig
	AdminSupportResolver       func(*http.Request) adminstudio.SupportConfig
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
	PublicErrorPage            func(context.Context, int) ([]byte, bool)
	Maintenance                MaintenanceConfig
	// AdminRequestAudit, when set, wraps the router to record admin API writes
	// (see internal/audit.RequestLog).
	AdminRequestAudit  func(http.Handler) http.Handler
	CustomCodeProvider func(*http.Request) (head, body string)
	SafeMode           bool
	PublicHTMLCache    *PublicHTMLCache
	// MaxInFlightRequests caps concurrently handled requests for this process.
	// Requests beyond the cap fail fast with 503 instead of queuing behind
	// already-admitted work (ADR-0333). Non-positive disables the limiter.
	MaxInFlightRequests int
	// CSRFExemptPaths skips CSRF validation entirely for the listed routes:
	// an inbound third-party webhook (a payment provider's callback) has no
	// browser session to carry the double-submit cookie, and its own
	// request signature is the real security boundary. See
	// security.CSRFMiddleware's own doc comment.
	CSRFExemptPaths []string
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
	return serveListener(ctx, config, listener)
}

func serveListener(ctx context.Context, config Config, listener net.Listener) error {
	ctx = nonNilContext(ctx)
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
	if config.CustomCodeProvider != nil && !config.SafeMode {
		handler = CustomCodeMiddleware(config.CustomCodeProvider)(handler)
	}
	// Runs inside the auth middleware, so the signed-in user and request id are
	// already on the request context.
	if config.AdminRequestAudit != nil {
		handler = config.AdminRequestAudit(handler)
	}
	// Cache the uncompressed representation; compression runs per response.

	handler = CompressionMiddleware(handler)
	handler = ETagMiddleware(handler)
	handler = config.PublicHTMLCache.Middleware(handler)

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
		return r.URL.Path == "/api/v1/admin/media"
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
	handler = PublicErrorPageMiddleware(config.PublicErrorPage)(handler)
	handler = observability.LoggingMiddleware(config.Logger)(handler)
	// Trace context is optional/no-op until an operator configures an exporter.
	// It is wrapped before RequestID below so the request correlation ID is
	// available as a bounded trace attribute.
	handler = observability.TracingMiddleware(handler)

	// Generate a fresh correlation identifier before security and auth middleware.
	handler = security.RequestIDMiddleware(handler)

	// Apply CSRF protection. The news list route renders no CSRF-dependent widget
	// (verified by reading modules/news/transport/http.go's publicListHTML), so a
	// cookie-less GET there can skip minting hydro_csrf and stay eligible for the
	// shared/edge caches that any Set-Cookie would otherwise permanently defeat.
	// Individual pages and news articles always render a cookie-consent or
	// helpful-feedback widget bound to the CSRF cookie, so they keep today's
	// behavior unchanged.
	handler = security.CSRFMiddleware(config.SecureCookie, publicContentCacheEligible, config.CSRFExemptPaths...)(handler)

	if config.Maintenance.Enabled != nil {
		handler = MaintenanceMiddleware(config.Maintenance)(handler)
	}

	// Apply Auth session context injection
	if config.AuthSvc != nil {
		handler = auth.Middleware(config.AuthSvc, config.SecureCookie)(handler)
	}
	if config.IPBlocklist != nil {
		handler = config.IPBlocklist.Middleware(handler)
	}

	// Apply security headers (CSP allows inline code only by hash)
	handler = security.HeadersMiddleware(security.HeaderConfig{
		EnableHSTS:   config.EnableHSTS,
		Preset:       config.SecurityHeadersPreset,
		ScriptHashes: config.SecurityScriptHashes,
		StyleHashes:  config.SecurityStyleHashes,
	})(handler)

	// Keep metrics ahead of every other application middleware so rejected
	// requests from any layer below it (body limit, CSRF, auth, and routing)
	// are visible to operators. Only the concurrency limiter wraps outside
	// this, and it bypasses /metrics itself (see below).
	if config.Metrics != nil {
		handler = config.Metrics.Middleware(handler)
	}
	// Final response guard: no HTML comment is allowed into production output.
	handler = observability.HTMLCommentMiddleware(handler)
	handler = adminstudio.SupportConfigResolverMiddleware(config.AdminSupport, config.AdminSupportResolver)(handler)

	// Outermost: shed load before any other middleware spends CPU on a request
	// this process cannot serve promptly. Health/readiness probes and metrics
	// scraping are exempt: shedding them would make an orchestrator kill the
	// instance that is trying to protect itself, or blind an operator during
	// the exact overload they need visibility into.
	handler = ConcurrencyLimitMiddleware(config.MaxInFlightRequests, func(r *http.Request) bool {
		switch r.URL.Path {
		case "/healthz", "/readyz", "/metrics":
			return true
		default:
			return false
		}
	})(handler)

	return handler
}

// publicContentCacheEligible matches the same route patterns as
// staticCacheURLPath: "/public/news/{locale}/html" (the news list), and the
// individual "/public/pages/{locale}/{slug}/html" and
// "/public/news/{locale}/{slug}/html" routes. All of these now render their
// CSRF-dependent widgets (cookie consent, helpful-feedback) without requiring
// the cookie to already exist on this response — the widgets fetch a token
// client-side from /csrf-token when needed instead of relying on a
// server-baked value embedded in shared/cached HTML. See
// a browser client and internal/security/csrf_token_handler.go.
func publicContentCacheEligible(r *http.Request) bool {
	return staticCacheURLPath(r.URL.Path)
}
