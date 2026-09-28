package observability

import (
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"go.opentelemetry.io/otel/trace"
)

const maxLoggedPathBytes = 512

// LoggingMiddleware emits one bounded JSON event per request. It deliberately
// excludes query strings, request bodies, cookies, and response bodies.
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if logger == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			attrs := []slog.Attr{
				slog.String("request_id", security.RequestIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", boundedPath(r.URL.Path)),
				slog.String("route", NormalizeRoute(r.URL.Path)),
				slog.String("module", RouteModule(r.URL.Path)),
				slog.Int("status", recorder.status),
				slog.Int64("duration_ms", time.Since(started).Milliseconds()),
				slog.Int64("request_bytes", r.ContentLength),
				slog.Int64("response_bytes", int64(recorder.bytes)),
			}
			if span := trace.SpanContextFromContext(r.Context()); span.IsValid() {
				attrs = append(attrs, slog.String("trace_id", span.TraceID().String()))
			}
			if scope, err := tenant.FromContext(r.Context()); err == nil {
				attrs = append(attrs, slog.String("tenant_id", scope.TenantID))
			}
			if user := auth.UserFromContext(r.Context()); user != nil && user.ID != "" {
				attrs = append(attrs, slog.String("actor_id", user.ID), slog.String("user_id", user.ID))
			}
			if session := auth.SessionFromContext(r.Context()); session != nil && session.ID != "" {
				attrs = append(attrs, slog.String("session_id", session.ID))
			}
			if jobID := security.JobIDFromContext(r.Context()); jobID != "" {
				attrs = append(attrs, slog.String("job_id", jobID))
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http_request", attrs...)
		})
	}
}

func boundedPath(path string) string {
	if len(path) <= maxLoggedPathBytes {
		return path
	}
	cut := maxLoggedPathBytes
	for cut > 0 && !utf8.ValidString(path[:cut]) {
		cut--
	}
	return path[:cut] + "…"
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	bytes       int
	onStatus    func(int)
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	if r.onStatus != nil {
		r.onStatus(status)
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(body)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// HTTPErrorMiddleware reports generic 5xx responses without buffering bodies.
// The private recovery marker deduplicates panics, which are reported with a
// stack by the recovery observer instead.
func HTTPErrorMiddleware(observe func(*http.Request, int)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if observe == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recoveredPanic := false
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			recorder.onStatus = func(status int) {
				if recorder.Header().Get("X-Hydro-Recovered-Panic") == "true" {
					recorder.Header().Del("X-Hydro-Recovered-Panic")
					recoveredPanic = true
					return
				}
				if status >= http.StatusInternalServerError {
					observe(r, status)
				}
			}
			next.ServeHTTP(recorder, r)
			if recorder.status >= http.StatusInternalServerError && !recoveredPanic && !recorder.wroteHeader {
				observe(r, recorder.status)
			}
		})
	}
}
