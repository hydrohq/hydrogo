package requestmeta

import (
	"net/http"
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

func TestIPExtractor(t *testing.T) {
	tests := []struct {
		name         string
		behindProxy  bool
		trustedCIDRs []string
		remoteAddr   string
		headers      map[string]string
		want         string
	}{
		{name: "no proxy configured ignores headers", remoteAddr: "192.0.2.8:1234", headers: map[string]string{"X-Forwarded-For": "10.0.0.1"}, want: "192.0.2.8"},
		{name: "behind_proxy without a list trusts only a loopback proxy", behindProxy: true, remoteAddr: "127.0.0.1:1234", headers: map[string]string{"X-Forwarded-For": "203.0.113.3"}, want: "203.0.113.3"},
		{name: "behind_proxy without a list ignores headers from a public peer", behindProxy: true, remoteAddr: "192.0.2.8:1234", headers: map[string]string{"X-Forwarded-For": "203.0.113.3"}, want: "192.0.2.8"},
		{name: "a client cannot pick its address through a trusted proxy", behindProxy: true, remoteAddr: "127.0.0.1:1234", headers: map[string]string{"X-Forwarded-For": "6.6.6.6, 203.0.113.3"}, want: "203.0.113.3"},
		{name: "vendor headers are not trusted", behindProxy: true, remoteAddr: "127.0.0.1:1234", headers: map[string]string{"True-Client-IP": "6.6.6.6", "CF-Connecting-IP": "6.6.6.6"}, want: "127.0.0.1"},
		{name: "trusted CIDRs skip every trusted hop", trustedCIDRs: []string{"192.0.2.8/32", "10.0.0.0/8"}, remoteAddr: "192.0.2.8:1234", headers: map[string]string{"X-Forwarded-For": "203.0.113.4, 10.0.0.5, 10.0.0.6"}, want: "203.0.113.4"},
		{name: "all hops trusted resolves to the originating host", trustedCIDRs: []string{"192.0.2.8/32", "10.0.0.0/8"}, remoteAddr: "192.0.2.8:1234", headers: map[string]string{"X-Forwarded-For": "10.0.0.5, 10.0.0.6"}, want: "10.0.0.5"},
		{name: "untrusted peer is taken at face value", trustedCIDRs: []string{"10.0.0.0/8"}, remoteAddr: "192.0.2.8:1234", headers: map[string]string{"X-Forwarded-For": "203.0.113.5"}, want: "192.0.2.8"},
		{name: "malformed hop stops at the proxy", trustedCIDRs: []string{"10.0.0.0/8"}, remoteAddr: "10.0.0.2:1234", headers: map[string]string{"X-Forwarded-For": "not-an-ip"}, want: "10.0.0.2"},
		{name: "X-Real-IP from a trusted proxy", trustedCIDRs: []string{"10.0.0.0/8"}, remoteAddr: "10.0.0.2:1234", headers: map[string]string{"X-Real-IP": "203.0.113.9"}, want: "203.0.113.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, err := NewIPExtractor(tt.behindProxy, tt.trustedCIDRs)
			if err != nil {
				t.Fatalf("NewIPExtractor failed: %v", err)
			}
			req := httptest.NewRequest("GET", "http://example.test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if got := ext.Extract(req); got != tt.want {
				t.Fatalf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMiddlewareMakesClientIPProxyAware(t *testing.T) {
	ext, err := NewIPExtractor(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	var seen string
	handler := Middleware(ext)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = ClientIP(r) }))
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.77")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "203.0.113.77" {
		t.Fatalf("ClientIP behind the middleware = %q, want the forwarded client", seen)
	}
}
