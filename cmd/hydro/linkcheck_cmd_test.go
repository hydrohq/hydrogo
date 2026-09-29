package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScanLinksCommandOutputsActionableCSV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<title>Home</title><a href="/missing">broken page</a><img src="/gone.png" alt="logo">`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	var out, stderr strings.Builder
	if err := scanLinksCommand([]string{"--url", server.URL, "--delay", "1ms", "--concurrency", "2", "--allow-private-targets"}, &out, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"source_page,source_title,broken_url,http_status,anchor_text,suggested_action", server.URL + "/missing", server.URL + "/gone.png", "Home", "broken page", "logo", "404", "Update the address"} {
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
