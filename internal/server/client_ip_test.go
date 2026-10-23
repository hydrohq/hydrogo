package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// Behind a reverse proxy every connection comes from the proxy. Banning a
// client must ban that client, not the proxy (which would take the whole
// site down), and a direct caller must not be able to forge its address.
func TestBehindAProxyTheBlocklistSeesTheForwardedClient(t *testing.T) {
	extractor, err := requestmeta.NewIPExtractor(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	blocklist := security.NewIPBlocklist(16)
	if err := blocklist.Ban("203.0.113.66", time.Time{}); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(Config{ClientIP: extractor, IPBlocklist: blocklist, Routes: []router.Route{{
		Pattern: "GET /probe", Access: router.Public,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}}})

	request := func(remote, forwarded string) int {
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		req.RemoteAddr = remote
		if forwarded != "" {
			req.Header.Set("X-Forwarded-For", forwarded)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := request("127.0.0.1:4000", "203.0.113.66"); got != http.StatusForbidden {
		t.Fatalf("banned client through the proxy: status %d, want 403", got)
	}
	if got := request("127.0.0.1:4001", "198.51.100.1"); got != http.StatusNoContent {
		t.Fatalf("another client through the same proxy: status %d, want 204 (the proxy itself must not be banned)", got)
	}
	if got := request("203.0.113.66:5000", "198.51.100.1"); got != http.StatusForbidden {
		t.Fatalf("banned client connecting directly with a forged header: status %d, want 403", got)
	}
}
