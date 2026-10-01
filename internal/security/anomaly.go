package security

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

type AnomalyConfig struct {
	Enabled      bool
	MaxBodyBytes int64
	BypassPaths  []string
	OnDetect     func(*http.Request, string)
	Extractor    interface {
		Extract(*http.Request) string
	}
}

var anomalyPatterns = []struct {
	rule string
	text string
}{
	{rule: "sql_union", text: "union select"},
	{rule: "sql_boolean", text: "or 1=1"},
	{rule: "sql_delay", text: "sleep("},
	{rule: "xss_tag", text: "<script"},
	{rule: "xss_event", text: "onerror="},
	{rule: "xss_scheme", text: "javascript:"},
}

// AnomalyMiddleware rejects a short allowlist of obvious injection markers.
// It is defense in depth only; application validation and parameterized SQL
// remain the security boundary.
func AnomalyMiddleware(cfg AnomalyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled || bypassAnomalyPath(r.URL.Path, cfg.BypassPaths) {
				next.ServeHTTP(w, r)
				return
			}
			decodedQuery := make([]string, 0)
			for key, values := range r.URL.Query() {
				decodedQuery = append(decodedQuery, key)
				decodedQuery = append(decodedQuery, values...)
			}
			if rule := anomalyRule(strings.Join(decodedQuery, "&")); rule != "" {
				rejectAnomaly(w, r, cfg.OnDetect, rule)
				return
			}
			if rule := anomalyRule(r.URL.RawQuery); rule != "" {
				rejectAnomaly(w, r, cfg.OnDetect, rule)
				return
			}
			if bodyScannable(r) {
				limit := cfg.MaxBodyBytes
				if limit <= 0 {
					limit = 64 << 10
				}
				body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
				if err != nil {
					http.Error(w, "request could not be processed", http.StatusBadRequest)
					return
				}
				if int64(len(body)) > limit {
					http.Error(w, "request body is too large", http.StatusRequestEntityTooLarge)
					return
				}
				r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), r.Body))
				if rule := anomalyRule(string(body)); rule != "" {
					rejectAnomaly(w, r, cfg.OnDetect, rule)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func anomalyRule(value string) string {
	value = strings.ToLower(strings.ReplaceAll(value, "+", " "))
	for _, pattern := range anomalyPatterns {
		if strings.Contains(value, pattern.text) {
			return pattern.rule
		}
	}
	if strings.Contains(value, "union%20select") || strings.Contains(value, "%3cscript") {
		return "encoded_payload"
	}
	return ""
}

func bodyScannable(r *http.Request) bool {
	if r.Body == nil || (r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch) {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]))
	return contentType == "application/x-www-form-urlencoded" || contentType == "application/json" || strings.HasPrefix(contentType, "text/")
}

func bypassAnomalyPath(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" && (path == prefix || strings.HasSuffix(prefix, "/") && strings.HasPrefix(path, prefix)) {
			return true
		}
	}
	return false
}

func rejectAnomaly(w http.ResponseWriter, r *http.Request, onDetect func(*http.Request, string), rule string) {
	if onDetect != nil {
		onDetect(r, rule)
	}
	http.Error(w, "request could not be processed", http.StatusBadRequest)
}
