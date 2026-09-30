package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnomalyMiddlewareRejectsObviousPayloadBeforeHandler(t *testing.T) {
	called, detected := false, ""
	middleware := AnomalyMiddleware(AnomalyConfig{Enabled: true, OnDetect: func(_ *http.Request, rule string) { detected = rule }})
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`email=a%40example.test&note=UNION+SELECT+password`))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || called || detected != "sql_union" {
		t.Fatalf("anomaly response=%d called=%v rule=%q", response.Code, called, detected)
	}
}

func TestAnomalyMiddlewareRestoresBenignBodyAndHonorsBypass(t *testing.T) {
	for _, test := range []struct {
		path   string
		body   string
		bypass []string
	}{
		{path: "/contact", body: "message=hello"},
		{path: "/admin/pages/save", body: "body=%3Cscript%3E", bypass: []string{"/admin/pages/"}},
	} {
		called := false
		handler := AnomalyMiddleware(AnomalyConfig{Enabled: true, BypassPaths: test.bypass})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			if test.path == "/contact" {
				body := make([]byte, len(test.body))
				_, _ = r.Body.Read(body)
				if string(body) != test.body {
					t.Errorf("request body = %q", body)
				}
			}
		}))
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !called {
			t.Fatalf("path=%s response=%d called=%v", test.path, response.Code, called)
		}
	}
}

func TestAnomalyMiddlewareBoundsScannedBody(t *testing.T) {
	handler := AnomalyMiddleware(AnomalyConfig{Enabled: true, MaxBodyBytes: 4})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("oversized body reached handler") }))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345"))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status=%d", response.Code)
	}
}
