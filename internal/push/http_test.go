package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestWebSubscriptionEndpointBindsToSignedInUser(t *testing.T) {
	repo := &memorySubscriptions{}
	handler := Handler{Dispatcher: Dispatcher{Repository: repo}}
	body := `{"endpoint":"https://push.example.test/sub/1","keys":{"auth":"abcdefgh1234","p256dh":"abcdefghijklmnopqrstuvwx123456"}}`
	req := httptest.NewRequest("POST", "/api/v1/push/web/subscription", strings.NewReader(body))
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true}))
	recorder := httptest.NewRecorder()
	handler.subscribeWeb(recorder, req)
	if recorder.Code != 204 || len(repo.items) != 1 || repo.items[0].UserID != "admin-1" || repo.items[0].Platform != PlatformWeb {
		t.Fatalf("status=%d subscriptions=%+v", recorder.Code, repo.items)
	}
	if err := json.Unmarshal(repo.items[0].Keys, &struct {
		Endpoint string `json:"endpoint"`
	}{}); err != nil {
		t.Fatalf("stored browser subscription is invalid json: %v", err)
	}
}

func TestAdminBroadcastHonorsAudienceAndValidatesMessage(t *testing.T) {
	repo := &memorySubscriptions{items: []Subscription{{ID: "m1", UserID: "u1", Platform: PlatformMobile, Endpoint: strings.Repeat("m", 24)}, {ID: "m2", UserID: "u2", Platform: PlatformMobile, Endpoint: strings.Repeat("n", 24)}}}
	mobile := &fakeSender{}
	audited := ""
	handler := Handler{Dispatcher: Dispatcher{Repository: repo, Mobile: mobile}, Audit: auth.AuditRecorderFunc(func(_ context.Context, action, target string) error { audited = action + ":" + target; return nil })}
	request := httptest.NewRequest("POST", "/admin/push/broadcast", strings.NewReader(url.Values{"title": {"Campaign"}, "body": {"A notice"}, "audience": {"mobile"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set(security.CSRFHeaderName, "csrf-token")
	request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf-token"})
	request = request.WithContext(auth.WithUser(context.Background(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	routeHandler := protectedPushRoutes(t, handler)
	recorder := httptest.NewRecorder()
	routeHandler.ServeHTTP(recorder, request)
	if recorder.Code != 303 || mobile.sent != 2 || !strings.Contains(recorder.Header().Get("Location"), "sent=2%2C2%2C0") {
		t.Fatalf("status=%d sent=%d location=%q", recorder.Code, mobile.sent, recorder.Header().Get("Location"))
	}
	if audited != "push.broadcast:mobile:2:2:0" {
		t.Fatalf("audit record=%q", audited)
	}
}

func TestAdminBroadcastRouteRequiresCSRFAndAdminRole(t *testing.T) {
	repo := &memorySubscriptions{items: []Subscription{{ID: "m1", UserID: "u1", Platform: PlatformMobile, Endpoint: strings.Repeat("m", 24)}}}
	mobile := &fakeSender{}
	audited := false
	handler := Handler{
		Dispatcher: Dispatcher{Repository: repo, Mobile: mobile},
		Audit:      auth.AuditRecorderFunc(func(context.Context, string, string) error { audited = true; return nil }),
	}
	routes := protectedPushRoutes(t, handler)
	for _, tc := range []struct {
		name   string
		role   auth.Role
		token  string
		active bool
		status int
	}{
		{name: "admin without token", role: auth.RoleAdmin, active: true},
		{name: "admin with invalid token", role: auth.RoleAdmin, token: "wrong-token", active: true},
		{name: "inactive admin with valid token", role: auth.RoleAdmin, token: "csrf-token", status: http.StatusUnauthorized},
		{name: "editor with valid token", role: auth.RoleEditor, token: "csrf-token", active: true},
		{name: "viewer with valid token", role: auth.RoleViewer, token: "csrf-token", active: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/admin/push/broadcast", strings.NewReader(url.Values{
				"title": {"Campaign"}, "body": {"A notice"}, "audience": {"mobile"},
			}.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tc.token != "" {
				request.Header.Set(security.CSRFHeaderName, tc.token)
			}
			request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf-token"})
			request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: string(tc.role), Role: tc.role, Active: tc.active}))
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			wantStatus := tc.status
			if wantStatus == 0 {
				wantStatus = http.StatusForbidden
			}
			if response.Code != wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), wantStatus)
			}
		})
	}
	if mobile.sent != 0 || audited {
		t.Fatalf("denied broadcasts had side effects: sent=%d audited=%t", mobile.sent, audited)
	}
}

func protectedPushRoutes(t *testing.T, handler Handler) http.Handler {
	t.Helper()
	mux, err := router.New(handler.Routes(), func(r *http.Request) bool {
		user := auth.UserFromContext(r.Context())
		return user != nil && user.Active
	})
	if err != nil {
		t.Fatal(err)
	}
	return security.CSRFMiddleware(false, nil)(mux)
}

func TestPushUIAndWorkerAreModuleRouteResources(t *testing.T) {
	handler := Handler{Dispatcher: Dispatcher{}}
	worker := httptest.NewRecorder()
	handler.serviceWorker(worker, httptest.NewRequest("GET", "/push/service-worker.js", nil))
	if worker.Code != 200 || worker.Header().Get("Service-Worker-Allowed") != "/" || !strings.Contains(worker.Body.String(), "notificationclick") {
		t.Fatalf("worker status=%d headers=%v", worker.Code, worker.Header())
	}
}
