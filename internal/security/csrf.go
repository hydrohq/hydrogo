package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	CSRFCookieName         = "hydro_csrf"
	CSRFHeaderName         = "X-CSRF-Token"
	CSRFFormField          = "_csrf"
	maxCSRFFormBytes int64 = 1 << 20
)

type csrfContextKey struct{}

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// apiTokenPrefix mirrors auth.APITokenPrefix; security cannot import auth.
const apiTokenPrefix = "hyd_"

func carriesAPIToken(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-API-Key")) != "" {
		return true
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	return len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.HasPrefix(parts[1], apiTokenPrefix)
}

func isSafeCSRFMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func CSRFTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if tok, ok := ctx.Value(csrfContextKey{}).(string); ok {
		return tok
	}
	return ""
}

// CSRFMiddleware issues and validates the double-submit CSRF cookie. skipCookieFor,
// when non-nil, is consulted only for cookie-less safe-method (GET/HEAD/OPTIONS)
// requests; when it reports true, the request proceeds with a per-request token
// but no Set-Cookie header is written. This exists for routes that are otherwise
// eligible for shared/edge caching (see internal/server/static_cache.go): any
// Set-Cookie header permanently disqualifies a response from that cache (and from
// most CDN/browser shared caches), so unconditionally minting a cookie on every
// anonymous GET defeats caching entirely. Callers must only mark a route via
// skipCookieFor when they have verified the route's handler never renders a form
// or widget that depends on the CSRF cookie actually reaching the browser.
// exemptPaths, when given, are routes skipped entirely regardless of
// method -- an inbound third-party webhook (a payment provider's server-to-
// server callback) has no browser session to carry a CSRF cookie, and its
// own request signature is the real security boundary; CSRF double-submit
// protects browser-originated mutations, not these. Matched exactly, or as
// a prefix when the configured value ends in "/", the same semantics as
// internal/security/anomaly.go's BypassPaths.
func CSRFMiddleware(secure bool, skipCookieFor func(*http.Request) bool, exemptPaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if csrfPathExempt(r.URL.Path, exemptPaths) {
				next.ServeHTTP(w, r)
				return
			}
			// Mobile credentials are bearer tokens carried outside browser cookies.
			// This dedicated surface authenticates every mutation in its handlers;
			// applying double-submit CSRF here would make native clients unusable.
			if strings.HasPrefix(r.URL.Path, "/api/v1/mobile/") {
				next.ServeHTTP(w, r)
				return
			}
			// HydroGo API tokens (Authorization: Bearer hyd_… / X-API-Key) make
			// the auth middleware ignore cookies, so there is no ambient
			// credential for CSRF to protect; mint no cookie and skip checks.
			if carriesAPIToken(r) {
				next.ServeHTTP(w, r)
				return
			}
			// CSP violation reports are telemetry, not state-changing application
			// mutations; their own bounded parser handles validation.
			if r.URL.Path == "/csp-report" && r.Method == http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			var token string
			cookie, err := r.Cookie(CSRFCookieName)
			switch {
			case err == nil && cookie.Value != "":
				token = cookie.Value
			case skipCookieFor != nil && isSafeCSRFMethod(r.Method) && skipCookieFor(r):
				tok, genErr := GenerateCSRFToken()
				if genErr != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				token = tok
			default:
				tok, genErr := GenerateCSRFToken()
				if genErr != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				token = tok
				http.SetCookie(w, &http.Cookie{ // #nosec G124 -- double-submit CSRF token: the admin SPA reads it to send X-CSRF-Token, it grants no access by itself and is SameSite=Strict
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // Accessible by JS/HTMX for headers if needed
					Secure:   secure,
					SameSite: http.SameSiteStrictMode,
				})
			}

			ctx := context.WithValue(r.Context(), csrfContextKey{}, token)

			// Safe methods do not require CSRF validation
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Browsers label every request with Sec-Fetch-Site and a page cannot
			// forge it. A write that another site's page started is refused even
			// if it somehow carries a valid token (defence in depth on top of the
			// SameSite cookie and the double-submit token).
			if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
				writeCSRFRejection(w, r, "cross-site request refused")
				return
			}

			// Validate state-changing methods
			providedToken := r.Header.Get(CSRFHeaderName)
			if providedToken == "" {
				// Parse small forms only. Large/chunked multipart requests must send
				// the token in the header so CSRF validation cannot buffer an upload.
				contentType := r.Header.Get("Content-Type")
				if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") || (strings.HasPrefix(contentType, "multipart/form-data") && r.ContentLength >= 0 && r.ContentLength <= maxCSRFFormBytes) {
					r.Body = http.MaxBytesReader(w, r.Body, maxCSRFFormBytes)
					if strings.HasPrefix(contentType, "multipart/form-data") {
						_ = r.ParseMultipartForm(2 << 20) // #nosec G120 -- the body is capped by MaxBytesReader above
					} else {
						_ = r.ParseForm()
					}
					providedToken = r.FormValue(CSRFFormField)
				}
			}

			if providedToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(providedToken)) != 1 {
				writeCSRFRejection(w, r, "invalid or missing csrf token")
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func csrfPathExempt(path string, exemptPaths []string) bool {
	for _, exempt := range exemptPaths {
		exempt = strings.TrimSpace(exempt)
		if exempt == "" {
			continue
		}
		if path == exempt || (strings.HasSuffix(exempt, "/") && strings.HasPrefix(path, exempt)) {
			return true
		}
	}
	return false
}

// writeCSRFRejection answers 403 in the API error envelope, with the request
// id, so a client can show and report it like any other failure.
func writeCSRFRejection(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:      ErrorCode(http.StatusForbidden),
		Message:   message,
		RequestID: RequestIDFromContext(r.Context()),
	})
}
