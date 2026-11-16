package security

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsBlockedSSRFIP(t *testing.T) {
	cases := []struct {
		ip            string
		allowLoopback bool
		blocked       bool
	}{
		// Loopback
		{"127.0.0.1", false, true},
		{"127.0.0.1", true, false},
		{"127.0.1.1", false, true},
		{"::1", false, true},
		{"::1", true, false},

		// Link-Local (Cloud metadata)
		{"169.254.169.254", false, true},
		{"169.254.1.1", true, true},

		// RFC 1918 Private
		{"10.0.0.1", false, true},
		{"10.255.255.255", true, true},
		{"172.16.0.1", false, true},
		{"172.31.255.255", true, true},
		{"192.168.1.1", false, true},
		{"192.168.0.100", true, true},

		// CGNAT
		{"100.64.0.1", false, true},
		{"100.127.255.254", true, true},

		// 0.0.0.0
		{"0.0.0.0", false, true},

		// Public IPs (should be allowed)
		{"8.8.8.8", false, false},
		{"1.1.1.1", true, false},
		{"93.184.216.34", false, false},
	}

	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP %s", c.ip)
		}
		got := IsBlockedSSRFIP(ip, c.allowLoopback)
		if got != c.blocked {
			t.Errorf("IsBlockedSSRFIP(%s, allowLoopback=%v) = %v; want %v", c.ip, c.allowLoopback, got, c.blocked)
		}
	}
}

func TestSSRFProtectedTransportBlocksLoopback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Default: allowLoopback = false -> should block
	clientBlocked := &http.Client{
		Transport: NewSSRFProtectedTransport(false),
		Timeout:   2 * time.Second,
	}
	_, err := clientBlocked.Get(ts.URL)
	if err == nil {
		t.Fatal("expected request to loopback server to be blocked, but succeeded")
	}
	if !strings.Contains(err.Error(), "restricted address") && !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected SSRF error, got: %v", err)
	}

	// Allowed: allowLoopback = true -> should succeed
	clientAllowed := &http.Client{
		Transport: NewSSRFProtectedTransport(true),
		Timeout:   2 * time.Second,
	}
	resp, err := clientAllowed.Get(ts.URL)
	if err != nil {
		t.Fatalf("expected loopback request to succeed with allowLoopback=true, got: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}
}

func TestSSRFDialerBlocksDirectDial(t *testing.T) {
	dialer := NewSSRFDialer(false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Dialing metadata IP
	conn, err := dialer.DialContext(ctx, "tcp", "169.254.169.254:80")
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("expected dial to 169.254.169.254 to fail, but succeeded")
	}
	if !strings.Contains(err.Error(), "restricted address") {
		t.Fatalf("expected restricted address error, got: %v", err)
	}
}
