package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func isDevEnvironment() bool {
	dev := strings.TrimSpace(os.Getenv("HYDRO_DEV_MODE"))
	env := strings.ToLower(strings.TrimSpace(os.Getenv("HYDRO_ENV")))
	return dev == "1" || dev == "true" || env == "development" || env == "dev" || env == "local"
}

type cspNonceContextKey struct{}

func GenerateCSPNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func NonceFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if nonce, ok := ctx.Value(cspNonceContextKey{}).(string); ok {
		return nonce
	}
	return ""
}

type HeaderConfig struct {
	EnableHSTS bool
}

func HeadersMiddleware(cfg HeaderConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := GenerateCSPNonce()
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), cspNonceContextKey{}, nonce)

			csp := fmt.Sprintf(
				"default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' 'nonce-%s'; style-src-attr 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self';",
				nonce, nonce,
			)

			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			if cfg.EnableHSTS {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}

			// In development mode or local runs, enforce no-store/no-cache on HTML and studio pages
			// to avoid stale UI renders and ensure instant updates.
			if isDevEnvironment() {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
