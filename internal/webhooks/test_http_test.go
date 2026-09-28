package webhooks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func TestTestHandlerDispatchesOneSignedRequestOutsideEventQueue(t *testing.T) {
	calls := 0
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Hydro-Signature") == "" {
			t.Error("signature missing")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer receiver.Close()
	handler := TestHandler{Endpoints: []Endpoint{{ID: "site", URL: receiver.URL, Secret: "test", Topics: map[string]struct{}{"page.published": {}}}}}
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true})
	request := httptest.NewRequest("POST", "/", strings.NewReader("endpoint_id=site&topic=page.published"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 || !strings.Contains(response.Body.String(), `"status":200`) || !strings.Contains(response.Body.String(), "sha256=") {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
	}
}

func TestTestHandlerRejectsNonAdmin(t *testing.T) {
	handler := TestHandler{Endpoints: []Endpoint{{ID: "site", URL: "https://example.test", Secret: "secret", Topics: map[string]struct{}{"page.published": {}}}}}
	req := httptest.NewRequest("POST", "/?endpoint_id=site&topic=page.published", nil).WithContext(auth.WithUser(context.Background(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d", rec.Code)
	}
}

func TestTestHandlerRendersAdminSandbox(t *testing.T) {
	handler := TestHandler{Endpoints: []Endpoint{{ID: "billing", URL: "https://receiver.test", Secret: "never-show-this", Topics: map[string]struct{}{"page.published": {}}}}}
	req := httptest.NewRequest("GET", "/admin/webhooks/test", nil).WithContext(auth.WithUser(context.Background(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Webhook test sandbox") || !strings.Contains(rec.Body.String(), `value="billing"`) || strings.Contains(rec.Body.String(), "never-show-this") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
