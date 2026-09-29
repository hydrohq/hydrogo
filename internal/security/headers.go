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
	Preset     string
}

const (
	HeaderPresetStrict   = "strict"
	HeaderPresetStandard = "standard"
	HeaderPresetRelaxed  = "relaxed"
)

func HeadersMiddleware(cfg HeaderConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := GenerateCSPNonce()
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), cspNonceContextKey{}, nonce)

			preset := strings.ToLower(strings.TrimSpace(cfg.Preset))
			if preset == "" {
				preset = HeaderPresetStrict
			}
			frameAncestors, frameOptions := "'none'", "DENY"
			referrer := "strict-origin-when-cross-origin"
			stylePolicy := "'nonce-" + nonce + "'"
			switch preset {
			case HeaderPresetStandard:
				frameAncestors, frameOptions = "'self'", "SAMEORIGIN"
				referrer = "no-referrer-when-downgrade"
			case HeaderPresetRelaxed:
				frameAncestors, frameOptions = "'self'", "SAMEORIGIN"
				referrer = "no-referrer-when-downgrade"
				stylePolicy += " 'unsafe-inline'"
			default:
				preset = HeaderPresetStrict
			}
			csp := fmt.Sprintf("default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' %s; style-src-attr 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob:; frame-src 'self' https://www.youtube-nocookie.com https://player.vimeo.com; object-src 'none'; base-uri 'self'; frame-ancestors %s; form-action 'self';", nonce, stylePolicy, frameAncestors)
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", frameOptions)
			w.Header().Set("Referrer-Policy", referrer)
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
			if extra := strings.TrimSpace(w.Header().Get("X-Hydro-Custom-CSP")); extra != "" {
				w.Header().Del("X-Hydro-Custom-CSP")
				w.Header().Set("Content-Security-Policy", mergeCSP(w.Header().Get("Content-Security-Policy"), extra))
			}
		})
	}
}

func mergeCSP(policy, extra string) string {
	directives := strings.Split(policy, ";")
	for _, addition := range strings.Split(extra, ";") {
		addition = strings.TrimSpace(addition)
		if addition == "" {
			continue
		}
		parts := strings.Fields(addition)
		if len(parts) < 2 {
			continue
		}
		merged := false
		for i, current := range directives {
			fields := strings.Fields(current)
			if len(fields) == 0 || fields[0] != parts[0] {
				continue
			}
			seen := make(map[string]struct{}, len(fields))
			for _, field := range fields[1:] {
				seen[field] = struct{}{}
			}
			for _, source := range parts[1:] {
				if _, ok := seen[source]; !ok {
					fields = append(fields, source)
				}
			}
			directives[i] = " " + strings.Join(fields, " ")
			merged = true
			break
		}
		if !merged {
			directives = append(directives, " "+addition)
		}
	}
	return strings.TrimSpace(strings.Join(directives, ";")) + ";"
}
