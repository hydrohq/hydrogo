package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type linkcheckRoundTripper func(*http.Request) (*http.Response, error)

func (f linkcheckRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestScanLinksCommandOutputsActionableCSV(t *testing.T) {
	client := &http.Client{Transport: linkcheckRoundTripper(func(r *http.Request) (*http.Response, error) {
		status, contentType, body := http.StatusNotFound, "text/plain", "missing"
		if r.URL.Path == "" || r.URL.Path == "/" {
			status, contentType = http.StatusOK, "text/html"
			body = `<title>Home</title><a href="/missing">broken page</a><img src="/gone.png" alt="logo">`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	var out, stderr strings.Builder
	const startURL = "https://example.test"
	if err := scanLinksCommandWithClient([]string{"--url", startURL, "--delay", "1ms", "--concurrency", "2"}, &out, &stderr, client); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"source_page,source_title,broken_url,http_status,anchor_text,suggested_action", startURL + "/missing", startURL + "/gone.png", "Home", "broken page", "logo", "404", "Update the address"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("CSV missing %q: %s", expected, out.String())
		}
	}
}

func TestScanLinksCommandRequiresURL(t *testing.T) {
	var out, stderr strings.Builder
	if err := scanLinksCommand(nil, &out, &stderr); err == nil {
		t.Fatal("missing URL was accepted")
	}
}
