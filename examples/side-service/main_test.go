package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const secret = "0123456789abcdef"

func signed(body string, at time.Time, key string) *http.Request {
	stamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(stamp + "." + body))
	r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(body))
	r.Header.Set("X-Hydro-Timestamp", stamp)
	r.Header.Set("X-Hydro-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return r
}

func TestHandler(t *testing.T) {
	now := time.Now()
	h := handler(secret, func() time.Time { return now })
	post := `{"id":"1","topic":"hook.invoices.transition","data":{"type":"invoices","on":"transition","to":"posted","fields":{"total":0}}}`
	ok := `{"id":"2","topic":"hook.invoices.transition","data":{"type":"invoices","on":"transition","to":"posted","fields":{"total":5}}}`
	for name, tc := range map[string]struct {
		req  *http.Request
		code int
	}{
		"rejects by rule": {signed(post, now, secret), http.StatusUnprocessableEntity},
		"allows":          {signed(ok, now, secret), http.StatusNoContent},
		"wrong secret":    {signed(ok, now, "another-secret-value"), http.StatusUnauthorized},
		"stale request":   {signed(ok, now.Add(-time.Hour), secret), http.StatusUnauthorized},
		"unsigned":        {httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(ok)), http.StatusUnauthorized},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, tc.req)
		if w.Code != tc.code {
			t.Errorf("%s: status %d, want %d (%s)", name, w.Code, tc.code, w.Body)
		}
	}
}
