package linkcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

func TestCrawlFindsInternalExternalAndMissingAssets(t *testing.T) {
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<a href="/again">again</a>`))
	}))
	defer external.Close()
	var site *httptest.Server
	site = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<title>Home</title><a href="/missing">Read <b>this</b></a><img src="/gone.png" alt="hero"><a href="` + external.URL + `/missing">Outside</a>`))
		case "/missing", "/gone.png":
			http.NotFound(w, r)
		default:
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<a href="/">home</a>`))
		}
	}))
	defer site.Close()

	client := &http.Client{Timeout: time.Second}
	findings, err := Crawl(context.Background(), Options{StartURL: site.URL, MaxPages: 10, Concurrency: 3, RequestDelay: time.Millisecond, Client: client})
	if err != nil {
		t.Fatal(err)
	}
	joined := make(map[string]Finding, len(findings))
	for _, finding := range findings {
		joined[finding.URL] = finding
	}
	if got := joined[site.URL+"/missing"]; got.Status != 404 || got.AnchorText != "Read this" || got.SourceTitle != "Home" {
		t.Fatalf("internal finding=%#v; all=%#v", got, findings)
	}
	if got := joined[site.URL+"/gone.png"]; got.Status != 404 || got.AnchorText != "hero" {
		t.Fatalf("asset finding=%#v", got)
	}
	if got := joined[external.URL+"/missing"]; got.Status != 404 || got.AnchorText != "Outside" {
		t.Fatalf("external finding=%#v", got)
	}
}

func TestCrawlValidatesURLAndBoundsInputs(t *testing.T) {
	for _, raw := range []string{"", "javascript:alert(1)", "//example.test/path", "https://user:pass@example.test"} {
		if _, err := Crawl(context.Background(), Options{StartURL: raw}); err == nil {
			t.Errorf("accepted invalid URL %q", raw)
		}
	}
}

func TestParseLinksIncludesResponsiveImagesAndStylesheets(t *testing.T) {
	links := parseLinks([]byte(`<html><head><link rel="stylesheet" href="/site.css"></head><body><img srcset="/small.webp 480w, /large.webp 1200w" alt="hero"></body></html>`))
	got := make(map[string]bool)
	for _, link := range links {
		got[link.href] = true
	}
	for _, want := range []string{"/site.css", "/small.webp", "/large.webp"} {
		if !got[want] {
			t.Errorf("missing discovered asset %s: %#v", want, links)
		}
	}
}

func TestPublicAddressBlocksNonPublicNetworks(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "198.18.0.1", "::1", "fc00::1"} {
		address, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatal(err)
		}
		if publicAddress(address) {
			t.Errorf("private network address %s allowed", raw)
		}
	}
	for _, raw := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		address, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !publicAddress(address) {
			t.Errorf("public address %s blocked", raw)
		}
	}
}

func TestCrawlRetriesTransientResponsesBeforeReporting(t *testing.T) {
	var temporaryCalls atomic.Int32
	var stableCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<a href="/temporary">temporary</a><a href="/stable">stable</a>`))
		case "/temporary":
			if temporaryCalls.Add(1) == 1 {
				http.Error(w, "later", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/stable":
			stableCalls.Add(1)
			http.Error(w, "later", http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	findings, err := Crawl(context.Background(), Options{StartURL: server.URL, MaxPages: 5, MaxTargets: 10, RequestDelay: time.Millisecond, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if temporaryCalls.Load() != 2 || stableCalls.Load() != 2 {
		t.Fatalf("retry calls temporary=%d stable=%d", temporaryCalls.Load(), stableCalls.Load())
	}
	if len(findings) != 1 || findings[0].URL != server.URL+"/stable" || findings[0].Status != http.StatusServiceUnavailable {
		t.Fatalf("findings=%#v", findings)
	}
}
