package audit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type fakeRecorder struct {
	action, target string
	after          []byte
	calls          int
}

func (f *fakeRecorder) RecordDetails(_ context.Context, action, target string, _, after []byte) error {
	f.calls++
	f.action, f.target, f.after = action, target, after
	return nil
}

func TestRedactJSONHidesSecretsAtAnyDepth(t *testing.T) {
	out, err := json.Marshal(RedactJSON([]byte(`{"title":"Hello","password":"hunter2","nested":{"api_key":"abc","list":[{"refresh_token":"x","ok":1}]},"mfa_code":"123456"}`)))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, secret := range []string{"hunter2", `"abc"`, `"x"`, "123456"} {
		if strings.Contains(text, secret) {
			t.Errorf("%s leaked into %s", secret, text)
		}
	}
	if !strings.Contains(text, "Hello") || !strings.Contains(text, `"ok":1`) {
		t.Errorf("non-sensitive content was dropped: %s", text)
	}
}

func TestRequestDetailsStaysUnderTheAuditLimit(t *testing.T) {
	body := `{"body":"` + strings.Repeat("long text ", 2000) + `","items":[` + strings.Repeat(`{"a":"`+strings.Repeat("z", 150)+`"},`, 40) + `{}]}`
	out := RequestDetails("PUT", "/api/v1/admin/pages/1", "token=abc&x=1", 200, "1.2.3.4", "agent", []byte(body), false)
	if len(out) > maxDetailBytes || !json.Valid(out) {
		t.Fatalf("details invalid or too large: %d bytes", len(out))
	}
	if strings.Contains(string(out), "token=abc") {
		t.Fatal("query secret leaked")
	}
}

func TestMiddlewareRecordsAuthenticatedAdminWritesAndPassesBodyOn(t *testing.T) {
	rec := &fakeRecorder{}
	handler := RequestLog{Recorder: rec}.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) > 2 && string(body) != `{"title":"T","password":"p"}` {
			t.Errorf("handler saw a modified body: %q", body)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	signedIn := func(r *http.Request) *http.Request {
		return r.WithContext(auth.WithUser(r.Context(), &auth.User{ID: "u1", Active: true}))
	}

	request := signedIn(httptest.NewRequest(http.MethodPost, "/api/v1/admin/pages", strings.NewReader(`{"title":"T","password":"p"}`)))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if rec.calls != 1 || rec.action != "admin.request" || rec.target != "POST /api/v1/admin/pages" {
		t.Fatalf("unexpected event: %+v", rec)
	}
	if strings.Contains(string(rec.after), `"p"`) || !strings.Contains(string(rec.after), `"status":201`) {
		t.Fatalf("details wrong: %s", rec.after)
	}

	for name, r := range map[string]*http.Request{
		"GET":            signedIn(httptest.NewRequest(http.MethodGet, "/api/v1/admin/pages", nil)),
		"public path":    signedIn(httptest.NewRequest(http.MethodPost, "/api/v1/pages", strings.NewReader("{}"))),
		"anonymous POST": httptest.NewRequest(http.MethodPost, "/api/v1/admin/pages", strings.NewReader("{}")),
	} {
		before := rec.calls
		r.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(httptest.NewRecorder(), r)
		if rec.calls != before {
			t.Errorf("%s must not be recorded", name)
		}
	}
}
