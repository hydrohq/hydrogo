package requestmeta

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPDropsEphemeralPortAndIgnoresForwardedHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "192.0.2.8:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	if got := ClientIP(req); got != "192.0.2.8" {
		t.Fatalf("client IP = %q", got)
	}
}

func TestClientIPAcceptsBareAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "2001:db8::1"
	if got := ClientIP(req); got != "2001:db8::1" {
		t.Fatalf("client IP = %q", got)
	}
}
