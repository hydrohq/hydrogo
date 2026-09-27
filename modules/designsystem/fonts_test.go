package designsystem

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fontRoundTripper func(*http.Request) (*http.Response, error)

func (f fontRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestLocalizeGoogleFontDownloadsBoundedWOFF2AndGeneratesCSS(t *testing.T) {
	destination := t.TempDir()
	client := &http.Client{Transport: fontRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := "@font-face{font-family:'Inter';font-style:normal;font-weight:400 700;src:url(https://fonts.gstatic.com/s/inter/v13/test.woff2) format('woff2');unicode-range:U+0000-00FF;}"
		if request.URL.Host == "fonts.googleapis.com" {
			if request.Header.Get("User-Agent") == "" {
				t.Error("browser user agent is required for WOFF2 CSS")
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
		}
		if request.URL.Host != "fonts.gstatic.com" {
			t.Fatalf("unexpected font host: %s", request.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("woff2-bytes")), Header: make(http.Header), Request: request}, nil
	})}
	cssPath, err := LocalizeGoogleFont(context.Background(), "Inter", destination, client)
	if err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(cssPath)
	if err != nil || !strings.Contains(string(css), "font-display:swap") || !strings.Contains(string(css), "format('woff2')") || !strings.Contains(string(css), "body{font-family:'Inter'") {
		t.Fatalf("generated CSS=%s err=%v", css, err)
	}
	fonts, err := filepath.Glob(filepath.Join(destination, "*.woff2"))
	if err != nil || len(fonts) != 1 {
		t.Fatalf("font files=%v err=%v", fonts, err)
	}
	data, err := os.ReadFile(fonts[0])
	if err != nil || string(data) != "woff2-bytes" {
		t.Fatalf("font file=%q err=%v", data, err)
	}
}

func TestLocalizeGoogleFontRejectsUnsafeFamilyAndUntrustedAssetURL(t *testing.T) {
	if _, err := LocalizeGoogleFont(context.Background(), "Inter;url(evil)", t.TempDir(), nil); err == nil {
		t.Fatal("unsafe family accepted")
	}
	client := &http.Client{Transport: fontRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := "@font-face{font-family:'Inter';font-weight:400;src:url(https://evil.example/font.woff2);}"
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})}
	if _, err := LocalizeGoogleFont(context.Background(), "Inter", t.TempDir(), client); err == nil {
		t.Fatal("untrusted font URL was accepted")
	}
}

func TestFontAssetsOnlyServesGeneratedFontNames(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "font-test.woff2"), []byte("font"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := FontAssets(directory)
	request := httptest.NewRequest(http.MethodGet, "/fonts/font-test.woff2", nil)
	request.SetPathValue("path", "font-test.woff2")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "font/woff2" || response.Body.String() != "font" {
		t.Fatalf("font response=%d %v %q", response.Code, response.Header(), response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/fonts/secret.txt", nil)
	request.SetPathValue("path", "secret.txt")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected private file response %d", response.Code)
	}
}
