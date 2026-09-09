package security

import (
	"net/http"
	"net/http/httptest"
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
