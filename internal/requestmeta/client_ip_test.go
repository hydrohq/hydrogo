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

func TestIPExtractor(t *testing.T) {
	tests := []struct {
		name        string
		behindProxy bool
		trustedCIDRs []string
		remoteAddr  string
		headers     map[string]string
		want        string
	}{
		{
			name:       "direct connection ignores headers if not trusted",
			remoteAddr: "192.0.2.8:1234",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			want:       "192.0.2.8",
		},
		{
			name:        "behind proxy trusts True-Client-IP",
			behindProxy: true,
			remoteAddr:  "192.0.2.8:1234",
			headers:     map[string]string{"True-Client-IP": "203.0.113.1"},
			want:        "203.0.113.1",
		},
		{
			name:        "behind proxy trusts CF-Connecting-IP",
			behindProxy: true,
			remoteAddr:  "192.0.2.8:1234",
			headers:     map[string]string{"CF-Connecting-IP": "203.0.113.2"},
			want:        "203.0.113.2",
		},
		{
			name:        "behind proxy trusts X-Forwarded-For rightmost untrusted",
			behindProxy: true,
			remoteAddr:  "192.0.2.8:1234",
			headers:     map[string]string{"X-Forwarded-For": "203.0.113.3, 10.0.0.1"},
			want:        "203.0.113.3",
		},
		{
			name:         "trusted CIDR trusts X-Forwarded-For and skips trusted proxies",
			behindProxy:  false,
			trustedCIDRs: []string{"192.0.2.8/32", "10.0.0.0/8"},
			remoteAddr:   "192.0.2.8:1234",
			headers:      map[string]string{"X-Forwarded-For": "203.0.113.4, 10.0.0.5, 10.0.0.6"},
			want:         "203.0.113.4",
		},
		{
			name:         "trusted CIDR but XFF only has trusted proxies",
			behindProxy:  false,
			trustedCIDRs: []string{"192.0.2.8/32", "10.0.0.0/8"},
			remoteAddr:   "192.0.2.8:1234",
			headers:      map[string]string{"X-Forwarded-For": "10.0.0.5, 10.0.0.6"},
			want:         "10.0.0.5",
		},
		{
			name:         "IP spoofing prevented: direct connection not in trusted CIDRs",
			behindProxy:  false,
			trustedCIDRs: []string{"10.0.0.0/8"},
			remoteAddr:   "192.0.2.8:1234", // not trusted
			headers:      map[string]string{"X-Forwarded-For": "203.0.113.5"},
			want:         "192.0.2.8",
		},
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
				t.Errorf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}
