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
	CSRFCookieName = "hydro_csrf"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFFormField  = "_csrf"
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
	if tok, ok := ctx.Value(csrfContextKey{}).(string); ok {
		return tok
	}
	return ""
}

func CSRFMiddleware(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				// Parse form if multipart or urlencoded
				contentType := r.Header.Get("Content-Type")
				if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") || strings.HasPrefix(contentType, "multipart/form-data") {
					_ = r.ParseMultipartForm(32 << 20) // max 32MB in memory for form parsing
					if r.MultipartForm == nil {
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
