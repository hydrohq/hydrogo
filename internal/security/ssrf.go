package security

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// IsBlockedSSRFIP checks whether an IP address belongs to Private, Loopback,
// Link-Local (169.254.0.0/16, RFC 1918, RFC 3927) or other restricted ranges.
func IsBlockedSSRFIP(ip net.IP, allowLoopback bool) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() {
		return !allowLoopback
	}
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// 169.254.0.0/16 - Link-Local / AWS/GCP/Azure Metadata
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 127.0.0.0/8 - Loopback
		if ip4[0] == 127 {
			return !allowLoopback
		}
		// 10.0.0.0/8 - Private RFC 1918
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12 - Private RFC 1918
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16 - Private RFC 1918
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 100.64.0.0/10 - Carrier-Grade NAT
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// 0.0.0.0/8
		if ip4[0] == 0 {
			return true
		}
	}
	return false
}

// NewSSRFDialer returns a *net.Dialer configured with a Control hook that
// validates the resolved destination IP address against internal/restricted ranges.
func NewSSRFDialer(allowLoopback bool) *net.Dialer {
	return &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return errors.New("ssrf: invalid target IP address")
			}
			if IsBlockedSSRFIP(ip, allowLoopback) {
				return fmt.Errorf("ssrf: connection to restricted address %s blocked", ip.String())
			}
			return nil
		},
	}
}

// NewSSRFProtectedTransport returns an http.Transport configured with socket-level
// IP validation to block DNS-rebinding and SSRF attacks against internal networks.
func NewSSRFProtectedTransport(allowLoopback bool) *http.Transport {
	dialer := NewSSRFDialer(allowLoopback)
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// NewSSRFProtectedClient returns an http.Client equipped with SSRF protection
// at both the transport and redirect layers.
func NewSSRFProtectedClient(timeout time.Duration, allowLoopback bool) *http.Client {
	if timeout <= 0 {
		timeout = 25 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: NewSSRFProtectedTransport(allowLoopback),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
