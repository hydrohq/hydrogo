package security

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func TestIPBlocklistExactCIDRExpiryAndCapacity(t *testing.T) {
	list := NewIPBlocklist(2)
	if err := list.Ban("192.0.2.4", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := list.Ban("2001:db8::/32", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if !list.Blocked("192.0.2.4") || list.Blocked("192.0.2.5") || !list.Blocked("2001:db8::1") {
		t.Fatal("exact or CIDR ban lookup failed")
	}
	if err := list.Ban("198.51.100.1", time.Time{}); err != ErrBlocklistFull {
		t.Fatalf("full list error = %v", err)
	}
	if err := list.Unban("192.0.2.4"); err != nil || list.Blocked("192.0.2.4") {
		t.Fatalf("unban = %v, still blocked=%v", err, list.Blocked("192.0.2.4"))
	}
	if err := list.Replace([]IPBan{{Prefix: netip.MustParsePrefix("192.0.2.0/24")}}); err != nil || !list.Blocked("192.0.2.100") {
		t.Fatalf("replace = %v", err)
	}
	called := false
	handler := list.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.100:1234"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("blocked request status=%d downstream=%v", response.Code, called)
	}
}
