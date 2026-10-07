package security

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadersMiddlewareSetsSecurityContract(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{EnableHSTS: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusNoContent {
		t.Errorf("status = %d", response.Code)
	}
	if response.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS header missing")
	}
}

func TestInlineCSPDirectivesHashEachExactBlock(t *testing.T) {
	script, style := []byte("window.ready=true;"), []byte("body{display:grid}")
	scriptHash, styleHash := sha256.Sum256(script), sha256.Sum256(style)
	document := []byte("<script>" + string(script) + "</script><style>" + string(style) + "</style>")
	got := InlineCSPDirectives(document)
	for _, want := range []string{
		"script-src 'sha256-" + base64.StdEncoding.EncodeToString(scriptHash[:]) + "'",
		"style-src 'sha256-" + base64.StdEncoding.EncodeToString(styleHash[:]) + "'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("inline CSP directives missing %q: %s", want, got)
		}
	}
}

func TestHeadersMiddlewareAllowsConfiguredInlineHashes(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{ScriptHashes: []string{"script-hash"}, StyleHashes: []string{"style-hash"}})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := response.Header().Get("Content-Security-Policy")
	for _, want := range []string{"script-src 'self' 'sha256-script-hash'", "style-src 'self' 'sha256-style-hash'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP missing %q: %s", want, csp)
		}
	}
}
