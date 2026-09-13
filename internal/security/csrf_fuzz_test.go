package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzCSRFMiddlewareTokens(f *testing.F) {
	for _, seed := range []string{"", "token", "00000000000000000000000000000000", "\x00\xff"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, provided string) {
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		handler := CSRFMiddleware(false)(next)
		req := httptest.NewRequest(http.MethodPost, "/mutate", nil)
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "fixed-token"})
		req.Header.Set(CSRFHeaderName, provided)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if provided == "fixed-token" && recorder.Code != http.StatusOK {
			t.Fatalf("valid token status = %d", recorder.Code)
		}
	})
}
