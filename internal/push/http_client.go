package push

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

func webPushHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConns: 32, MaxIdleConnsPerHost: 2, MaxConnsPerHost: 4, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second, ExpectContinueTimeout: time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid push endpoint port")
		}
		ips := []netip.Addr{}
		if literal, err := netip.ParseAddr(host); err == nil {
			ips = append(ips, literal)
		} else {
			resolved, resolveErr := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if resolveErr != nil {
				return nil, resolveErr
			}
			ips = resolved
		}
		var lastErr error
		for _, ip := range ips {
			ip = ip.Unmap()
			if !publicPushIP(ip) {
				return nil, errors.New("push endpoint resolved to a non-public address")
			}
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		if lastErr == nil {
			lastErr = errors.New("push endpoint has no public address")
		}
		return nil, lastErr
	}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func publicPushIP(ip netip.Addr) bool {
	return ip.IsValid() && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
func safePushEndpointHost(host string) bool {
	if host == "" || host != strings.TrimSpace(host) || strings.HasSuffix(strings.ToLower(host), ".localhost") || strings.HasSuffix(strings.ToLower(host), ".local") || strings.EqualFold(host, "localhost") {
		return false
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return publicPushIP(ip)
	}
	return true
}
