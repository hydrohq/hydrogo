package security

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var inlineCSPBlock = regexp.MustCompile(`(?s)<(script|style)(?:\s[^>]*)?>(.*?)</(?:script|style)>`)

// InlineCSPDirectives hashes exact inline script and style bodies in trusted,
// server-rendered HTML so the caller can add them to the existing strict CSP.
func InlineCSPDirectives(document []byte) string {
	scripts, styles := InlineCSPHashes(document)
	var directives []string
	if len(scripts) > 0 {
		quoted := make([]string, 0, len(scripts))
		for _, source := range scripts {
			quoted = append(quoted, "'sha256-"+source+"'")
		}
		directives = append(directives, "script-src "+strings.Join(quoted, " "))
	}
	if len(styles) > 0 {
		quoted := make([]string, 0, len(styles))
		for _, source := range styles {
			quoted = append(quoted, "'sha256-"+source+"'")
		}
		directives = append(directives, "style-src "+strings.Join(quoted, " "))
	}
	return strings.Join(directives, "; ")
}

// InlineCSPHashes returns SHA-256 source values for the trusted inline script
// and style blocks in a rendered document. Callers must only hash machine-owned
// output; user content must be escaped or kept outside executable blocks.
func InlineCSPHashes(document []byte) (scripts, styles []string) {
	for _, match := range inlineCSPBlock.FindAllSubmatch(document, -1) {
		digest := sha256.Sum256(match[2])
		source := base64.StdEncoding.EncodeToString(digest[:])
		if bytes.Equal(match[1], []byte("script")) {
			scripts = append(scripts, source)
		} else {
			styles = append(styles, source)
		}
	}
	return scripts, styles
}

func isDevEnvironment() bool {
	dev := strings.TrimSpace(os.Getenv("HYDRO_DEV_MODE"))
	env := strings.ToLower(strings.TrimSpace(os.Getenv("HYDRO_ENV")))
	return dev == "1" || dev == "true" || env == "development" || env == "dev" || env == "local"
}

type HeaderConfig struct {
	EnableHSTS   bool
	Preset       string
	ScriptHashes []string
	StyleHashes  []string
}

const (
	HeaderPresetStrict   = "strict"
	HeaderPresetStandard = "standard"
	HeaderPresetRelaxed  = "relaxed"
)

func HeadersMiddleware(cfg HeaderConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			preset := strings.ToLower(strings.TrimSpace(cfg.Preset))
			if preset == "" {
				preset = HeaderPresetStrict
			}
			frameAncestors, frameOptions := "'none'", "DENY"
			referrer := "strict-origin-when-cross-origin"
			stylePolicy := "'self'"
			scriptPolicy := "'self'"
			for _, hash := range cfg.StyleHashes {
				stylePolicy += " 'sha256-" + hash + "'"
			}
			for _, hash := range cfg.ScriptHashes {
				scriptPolicy += " 'sha256-" + hash + "'"
			}
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
			csp := fmt.Sprintf("default-src 'self'; script-src %s; style-src %s; style-src-attr 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob:; frame-src 'self' https://www.youtube-nocookie.com https://player.vimeo.com; object-src 'none'; base-uri 'self'; frame-ancestors %s; form-action 'self';", scriptPolicy, stylePolicy, frameAncestors)
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

			next.ServeHTTP(w, r)
			if extra := strings.TrimSpace(w.Header().Get("X-Hydro-Custom-CSP")); extra != "" {
				w.Header().Del("X-Hydro-Custom-CSP")
				w.Header().Set("Content-Security-Policy", MergeCSP(w.Header().Get("Content-Security-Policy"), extra))
			}
		})
	}
}

// MergeCSP combines additional directive sources with the existing policy.
func MergeCSP(policy, extra string) string {
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

// AllowSameOriginFrame permits the current response to be embedded by the
// same origin, replacing any stricter frame-ancestors directive.
func AllowSameOriginFrame(header http.Header) {
	directives := strings.Split(header.Get("Content-Security-Policy"), ";")
	found := false
	for i, directive := range directives {
		fields := strings.Fields(directive)
		if len(fields) > 0 && fields[0] == "frame-ancestors" {
			directives[i] = " frame-ancestors 'self'"
			found = true
		}
	}
	if !found {
		directives = append(directives, " frame-ancestors 'self'")
	}
	header.Set("Content-Security-Policy", strings.TrimSpace(strings.Join(directives, ";")))
	header.Set("X-Frame-Options", "SAMEORIGIN")
}
