package bffpolicy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBuildCoreRequestStripsBrowserBoundaryAndUsesServerCookie(t *testing.T) {
	in := httptest.NewRequest(http.MethodPost, "https://storefront.example/api/v1/commerce/cart/checkout?x=1", nil)
	in.Header.Set("Authorization", "Bearer browser-token")
	in.Header.Set("Cookie", "hydro_session=browser-cookie")
	in.Header.Set("Origin", "https://storefront.example")
	in.Header.Set("Referer", "https://storefront.example/cart")
	if _, err := BuildCoreRequest(in, mustURL(t, "http://core.internal"), "hydro_session=server-side-cookie"); !errors.Is(err, ErrBrowserCredential) {
		t.Fatalf("expected browser credential rejection, got %v", err)
	}

	in.Header.Del("Authorization")
	out, err := BuildCoreRequest(in, mustURL(t, "http://core.internal"), "hydro_session=server-side-cookie")
	if err != nil {
		t.Fatal(err)
	}
	if out.URL.String() != "http://core.internal/api/v1/commerce/cart/checkout?x=1" {
		t.Fatalf("unexpected core URL: %s", out.URL)
	}
	if got := out.Header.Get("Cookie"); got != "hydro_session=server-side-cookie" {
		t.Fatalf("unexpected core cookie: %q", got)
	}
	for _, name := range []string{"Origin", "Referer", "Authorization"} {
		if out.Header.Get(name) != "" {
			t.Fatalf("%s leaked across BFF boundary", name)
		}
	}
}

func TestBuildCoreRequestRejectsNonAPIAndMissingServerSession(t *testing.T) {
	for _, requestPath := range []string{"/admin", "/metrics", "/api/internal", "/api/v1/../admin"} {
		in := httptest.NewRequest(http.MethodGet, "https://storefront.example"+requestPath, nil)
		if _, err := BuildCoreRequest(in, mustURL(t, "https://core.example"), "server=session"); !errors.Is(err, ErrDisallowedPath) {
			t.Errorf("%s: expected disallowed path, got %v", requestPath, err)
		}
	}
	in := httptest.NewRequest(http.MethodGet, "https://storefront.example/api/v1/pages", nil)
	if _, err := BuildCoreRequest(in, mustURL(t, "https://core.example"), ""); !errors.Is(err, ErrMissingCoreCookie) {
		t.Fatalf("expected missing server session, got %v", err)
	}
}

func TestValidateCoreOrigin(t *testing.T) {
	if _, err := ValidateCoreOrigin("https://core.example/"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "//core.example", "https://user:pass@core.example", "https://core.example/api/v1", "ftp://core.example"} {
		if _, err := ValidateCoreOrigin(value); !errors.Is(err, ErrInvalidCoreOrigin) {
			t.Errorf("%q: expected invalid origin, got %v", value, err)
		}
	}
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
