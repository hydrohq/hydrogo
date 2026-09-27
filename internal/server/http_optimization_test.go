package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestETagMiddlewareReturnsNotModifiedForMatchingTag(t *testing.T) {
	body := strings.Repeat("public page content ", 100)
	handler := ETagMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, body)
	}))
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/page", nil))
	if first.Code != http.StatusOK || first.Header().Get("ETag") == "" || first.Body.String() != body {
		t.Fatalf("first response: status=%d headers=%v body length=%d", first.Code, first.Header(), first.Body.Len())
	}
	request := httptest.NewRequest(http.MethodGet, "/page", nil)
	request.Header.Set("If-None-Match", first.Header().Get("ETag"))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)
	if second.Code != http.StatusNotModified || second.Body.Len() != 0 {
		t.Fatalf("conditional response: status=%d body=%q", second.Code, second.Body.String())
	}
}

func TestETagMiddlewareSkipsPersonalizedAndUnsafeResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		cookie bool
		cache  string
	}{{name: "cookie", cookie: true}, {name: "private", cache: "private, max-age=60"}, {name: "no store", cache: "no-store"}} {
		t.Run(test.name, func(t *testing.T) {
			handler := ETagMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.Header().Set("Cache-Control", test.cache)
				_, _ = io.WriteString(w, "page")
			}))
			request := httptest.NewRequest(http.MethodGet, "/page", nil)
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: "session", Value: "secret"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Header().Get("ETag") != "" {
				t.Fatalf("unexpected ETag on unsafe response: %v", response.Header())
			}
		})
	}
}

func TestCompressionMiddlewareNegotiatesBrotliAndGzip(t *testing.T) {
	body := strings.Repeat("hydro-go-compressible-response ", 120)
	for _, encoding := range []string{"br", "gzip"} {
		t.Run(encoding, func(t *testing.T) {
			handler := CompressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = io.WriteString(w, body)
			}))
			request := httptest.NewRequest(http.MethodGet, "/page", nil)
			request.Header.Set("Accept-Encoding", encoding)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if got := response.Header().Get("Content-Encoding"); got != encoding || !strings.Contains(response.Header().Get("Vary"), "Accept-Encoding") {
				t.Fatalf("compression headers: %v", response.Header())
			}
			var reader io.Reader
			if encoding == "br" {
				reader = brotli.NewReader(response.Body)
			} else {
				gzipReader, err := gzip.NewReader(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				defer gzipReader.Close()
				reader = gzipReader
			}
			plain, err := io.ReadAll(reader)
			if err != nil || string(plain) != body {
				t.Fatalf("decompressed body differs: error=%v length=%d", err, len(plain))
			}
		})
	}
}

func TestCompressionMiddlewareSkipsSmallAndAlreadyCompressedBodies(t *testing.T) {
	for _, test := range []struct{ typ, body string }{{"text/plain", "small"}, {"image/png", strings.Repeat("x", 1500)}} {
		handler := CompressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", test.typ)
			_, _ = io.WriteString(w, test.body)
		}))
		request := httptest.NewRequest(http.MethodGet, "/asset", nil)
		request.Header.Set("Accept-Encoding", "br, gzip")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Header().Get("Content-Encoding") != "" || response.Body.String() != test.body {
			t.Fatalf("unexpected encoding for %s: headers=%v", test.typ, response.Header())
		}
	}
}
