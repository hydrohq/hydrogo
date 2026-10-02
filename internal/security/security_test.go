package security

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type countingReader struct {
	reader io.Reader
	reads  atomic.Int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads.Add(1)
	return r.reader.Read(p)
}

func TestSecurityHeadersAndCSPNonce(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{EnableHSTS: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := NonceFromContext(r.Context())
		if nonce == "" {
			t.Errorf("expected CSP nonce in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' 'nonce-") {
		t.Errorf("expected CSP header with script nonce, got %q", csp)
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options DENY, got %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Errorf("expected HSTS header")
	}
}

func TestCSRFProtection(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	csrf := CSRFMiddleware(false, nil)(nextHandler)

	// 1. GET request generates a CSRF cookie and token in context
	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/form", nil)
	csrf.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("GET /form status %d", recGet.Code)
	}
	cookies := recGet.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == CSRFCookieName {
			csrfCookie = c
			break
		}
	}
	if csrfCookie == nil || csrfCookie.Value == "" {
		t.Fatalf("expected CSRF cookie on GET")
	}

	token := csrfCookie.Value

	// 2. POST request with invalid / missing token -> 403 Forbidden
	recPostBad := httptest.NewRecorder()
	reqPostBad := httptest.NewRequest(http.MethodPost, "/form", nil)
	reqPostBad.AddCookie(csrfCookie)
	csrf.ServeHTTP(recPostBad, reqPostBad)

	if recPostBad.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing CSRF header, got %d", recPostBad.Code)
	}

	// 3. POST request with valid X-CSRF-Token header -> 200 OK
	recPostGood := httptest.NewRecorder()
	reqPostGood := httptest.NewRequest(http.MethodPost, "/form", nil)
	reqPostGood.AddCookie(csrfCookie)
	reqPostGood.Header.Set(CSRFHeaderName, token)
	csrf.ServeHTTP(recPostGood, reqPostGood)

	if recPostGood.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid CSRF header, got %d", recPostGood.Code)
	}
}

func TestCSRFDoesNotReadLargeMultipartWithoutHeaderToken(t *testing.T) {
	var called atomic.Bool
	csrf := CSRFMiddleware(false, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called.Store(true)
	}))
	cookie := &http.Cookie{Name: CSRFCookieName, Value: "csrf-token"}
	body := &countingReader{reader: strings.NewReader(strings.Repeat("x", int(maxCSRFFormBytes)+1))}
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.ContentLength = maxCSRFFormBytes + 1
	req.Header.Set("Content-Type", "multipart/form-data; boundary=hydro")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	csrf.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden || called.Load() {
		t.Fatalf("large multipart CSRF status=%d called=%v", res.Code, called.Load())
	}
	if got := body.reads.Load(); got != 0 {
		t.Fatalf("large multipart body was read %d times before header validation", got)
	}
}

func TestCSRFAcceptsSmallMultipartFormToken(t *testing.T) {
	csrf := CSRFMiddleware(false, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	cookie := &http.Cookie{Name: CSRFCookieName, Value: "csrf-token"}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField(CSRFFormField, cookie.Value); err != nil {
		t.Fatalf("write CSRF form field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	csrf.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("small multipart CSRF status=%d, want 200", res.Code)
	}
}
