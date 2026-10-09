package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestTestHandlerDispatchesOneSignedRequestOutsideEventQueue(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read webhook body: %v", err)
		}
		mac := hmac.New(sha256.New, []byte("test"))
		_, _ = io.WriteString(mac, r.Header.Get("X-Hydro-Timestamp")+".")
		_, _ = mac.Write(body)
		if got, want := r.Header.Get("X-Hydro-Signature"), "sha256="+hex.EncodeToString(mac.Sum(nil)); got != want {
			t.Errorf("signature=%q want=%q", got, want)
		}
		if r.Header.Get("X-Hydro-Event") != "page.published" || !strings.Contains(string(body), `"initiated_by":"admin"`) {
			t.Errorf("event header/body=%q / %s", r.Header.Get("X-Hydro-Event"), body)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok")), Request: r}, nil
	})}
	handler := TestHandler{Dispatcher: Dispatcher{Client: client}, Endpoints: []Endpoint{{ID: "site", URL: "https://receiver.example.test/hook", Secret: "test", Topics: map[string]struct{}{"page.published": {}}}}}
	route := protectedWebhookRoutes(t, handler)
	ctx := auth.WithUser(context.Background(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true})
	request := httptest.NewRequest(http.MethodPost, "/admin/api/webhooks/test", strings.NewReader(url.Values{"endpoint_id": {"site"}, "topic": {"page.published"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
	request.Header.Set(security.CSRFHeaderName, "valid-token")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	route.ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 || !strings.Contains(response.Body.String(), `"status":200`) || !strings.Contains(response.Body.String(), "sha256=") {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
	}
}

func TestTestHandlerRouteRequiresCSRFAndAdminRole(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok")), Request: r}, nil
	})}
	handler := TestHandler{Dispatcher: Dispatcher{Client: client}, Endpoints: []Endpoint{{ID: "site", URL: "https://example.test/hook", Secret: "secret", Topics: map[string]struct{}{"page.published": {}}}}}
	route := protectedWebhookRoutes(t, handler)
	for _, tc := range []struct {
		name   string
		user   *auth.User
		token  string
		status int
	}{
		{name: "admin without csrf", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}},
		{name: "admin with invalid csrf", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, token: "wrong-token"},
		{name: "inactive admin with valid csrf", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: false}, token: "valid-token", status: http.StatusUnauthorized},
		{name: "editor with valid csrf", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "viewer with valid csrf", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, token: "valid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/admin/api/webhooks/test", strings.NewReader("endpoint_id=site&topic=page.published"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
			if tc.token != "" {
				request.Header.Set(security.CSRFHeaderName, tc.token)
			}
			request = request.WithContext(auth.WithUser(request.Context(), tc.user))
			response := httptest.NewRecorder()
			route.ServeHTTP(response, request)
			wantStatus := tc.status
			if wantStatus == 0 {
				wantStatus = http.StatusForbidden
			}
			if response.Code != wantStatus {
				t.Fatalf("status=%d want=%d body=%q", response.Code, wantStatus, response.Body.String())
			}
		})
	}
	if calls != 0 {
		t.Fatalf("denied requests dispatched %d webhook(s)", calls)
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

func protectedWebhookRoutes(t *testing.T, handler TestHandler) http.Handler {
	t.Helper()
	routes, err := router.New([]router.Route{
		{Pattern: "GET /admin/webhooks/test", Access: router.Protected, Handler: handler},
		{Pattern: "POST /admin/api/webhooks/test", Access: router.Protected, Handler: handler},
	}, func(r *http.Request) bool {
		user := auth.UserFromContext(r.Context())
		return user != nil && user.Active
	})
	if err != nil {
		t.Fatal(err)
	}
	return security.CSRFMiddleware(false, nil)(routes)
}

type webhookRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f webhookRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
