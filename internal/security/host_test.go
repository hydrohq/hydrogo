package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHostMiddlewareBindsCanonicalOrigin(t *testing.T) {
	middleware, err := NewHostMiddleware("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, test := range []struct {
		name string
		host string
		want int
	}{
		{name: "canonical host", host: "example.test:443", want: http.StatusNoContent},
		{name: "canonical default port", host: "example.test", want: http.StatusNoContent},
		{name: "trailing dot", host: "example.test.:443", want: http.StatusNoContent},
		{name: "wrong host", host: "attacker.test:443", want: http.StatusMisdirectedRequest},
		{name: "wrong port", host: "example.test:8443", want: http.StatusMisdirectedRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
			req.Host = test.host
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != test.want {
				t.Fatalf("host %q status = %d, want %d", test.host, res.Code, test.want)
			}
		})
	}
}

func TestHostMiddlewareRedirectsWWWTwinToCanonicalOrigin(t *testing.T) {
	for _, test := range []struct {
		name, origin, host, want string
		code                     int
	}{
		{name: "www to apex", origin: "https://example.test", host: "www.example.test", want: "https://example.test/urun/x?a=1", code: http.StatusPermanentRedirect},
		{name: "www with default port", origin: "https://example.test", host: "www.example.test:443", want: "https://example.test/urun/x?a=1", code: http.StatusPermanentRedirect},
		{name: "apex to www origin", origin: "https://www.example.test", host: "example.test", want: "https://www.example.test/urun/x?a=1", code: http.StatusPermanentRedirect},
		{name: "unrelated subdomain is still refused", origin: "https://example.test", host: "evil.example.test", code: http.StatusMisdirectedRequest},
		{name: "look-alike www is still refused", origin: "https://example.test", host: "www.example.test.evil.test", code: http.StatusMisdirectedRequest},
		{name: "www on wrong port is still refused", origin: "https://example.test", host: "www.example.test:8443", code: http.StatusMisdirectedRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			middleware, err := NewHostMiddleware(test.origin)
			if err != nil {
				t.Fatal(err)
			}
			handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not run") }))
			req := httptest.NewRequest(http.MethodGet, "/urun/x?a=1", nil)
			req.Host = test.host
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != test.code {
				t.Fatalf("status = %d, want %d", res.Code, test.code)
			}
			if test.want != "" && res.Header().Get("Location") != test.want {
				t.Fatalf("Location = %q, want %q", res.Header().Get("Location"), test.want)
			}
		})
	}
}

func TestHostMiddlewareAllowsUnconfiguredLocalDevelopment(t *testing.T) {
	middleware, err := NewHostMiddleware("")
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "anything.test"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("unconfigured host status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestHostMiddlewareCanonicalizesIDNHostnames(t *testing.T) {
	middleware, err := NewHostMiddleware("https://bücher.example")
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "https://xn--bcher-kva.example/", nil)
	req.Host = "xn--bcher-kva.example:443"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("IDN host status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestHostMismatchSaysWhatWasExpected(t *testing.T) {
	middleware, err := NewHostMiddleware("https://crm.example.test")
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("must not run") }))
	req := httptest.NewRequest(http.MethodGet, "http://hydrogo:8080/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusMisdirectedRequest || !strings.Contains(body, "crm.example.test:443") || !strings.Contains(body, "hydrogo:8080") || !strings.Contains(body, "Host header") {
		t.Errorf("status %d body %q", rec.Code, body)
	}
}

func TestHostMiddlewareLoopbackSSRAllowance(t *testing.T) {
	middleware, err := NewHostMiddleware("https://shop.example.test")
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	cases := []struct {
		name       string
		remoteAddr string
		host       string
		want       int
	}{
		{name: "loopback ipv4 SSR", remoteAddr: "127.0.0.1:54321", host: "127.0.0.1:8080", want: http.StatusNoContent},
		{name: "loopback ipv6 SSR", remoteAddr: "[::1]:54321", host: "[::1]:8080", want: http.StatusNoContent},
		{name: "loopback peer with foreign host", remoteAddr: "127.0.0.1:54321", host: "evil.example", want: http.StatusMisdirectedRequest},
		{name: "non-loopback peer spoofing loopback host", remoteAddr: "203.0.113.10:54321", host: "127.0.0.1:8080", want: http.StatusMisdirectedRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/api/v1/healthz", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Host = tc.host
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("remote=%s host=%s got %d, want %d", tc.remoteAddr, tc.host, rec.Code, tc.want)
			}
		})
	}
}
