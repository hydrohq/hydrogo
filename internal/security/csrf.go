package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
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

func CSRFTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if tok, ok := ctx.Value(csrfContextKey{}).(string); ok {
		return tok
	}
	return ""
}

func CSRFMiddleware(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mobile credentials are bearer tokens carried outside browser cookies.
			// This dedicated surface authenticates every mutation in its handlers;
			// applying double-submit CSRF here would make native clients unusable.
			if strings.HasPrefix(r.URL.Path, "/api/v1/mobile/") {
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
			if err == nil && cookie.Value != "" {
				token = cookie.Value
			} else {
				tok, err := GenerateCSRFToken()
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				token = tok
				http.SetCookie(w, &http.Cookie{
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

			// Validate state-changing methods
			providedToken := r.Header.Get(CSRFHeaderName)
			if providedToken == "" {
				// Parse small forms only. Large/chunked multipart requests must send
				// the token in the header so CSRF validation cannot buffer an upload.
				contentType := r.Header.Get("Content-Type")
				if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") || (strings.HasPrefix(contentType, "multipart/form-data") && r.ContentLength >= 0 && r.ContentLength <= maxCSRFFormBytes) {
					r.Body = http.MaxBytesReader(w, r.Body, maxCSRFFormBytes)
					if strings.HasPrefix(contentType, "multipart/form-data") {
						_ = r.ParseMultipartForm(2 << 20)
					} else {
						_ = r.ParseForm()
					}
					providedToken = r.FormValue(CSRFFormField)
				}
			}

			if providedToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(providedToken)) != 1 {
				http.Error(w, "invalid or missing csrf token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
