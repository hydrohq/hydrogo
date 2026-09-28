package push

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
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
	request = request.WithContext(auth.WithUser(context.Background(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	recorder := httptest.NewRecorder()
	handler.broadcast(recorder, request)
	if recorder.Code != 303 || mobile.sent != 2 || !strings.Contains(recorder.Header().Get("Location"), "sent=2%2C2%2C0") {
		t.Fatalf("status=%d sent=%d location=%q", recorder.Code, mobile.sent, recorder.Header().Get("Location"))
	}
	if audited != "push.broadcast:mobile:2:2:0" {
		t.Fatalf("audit record=%q", audited)
	}
}

func TestPushUIAndWorkerAreModuleRouteResources(t *testing.T) {
	handler := Handler{Dispatcher: Dispatcher{}}
	worker := httptest.NewRecorder()
	handler.serviceWorker(worker, httptest.NewRequest("GET", "/push/service-worker.js", nil))
	if worker.Code != 200 || worker.Header().Get("Service-Worker-Allowed") != "/" || !strings.Contains(worker.Body.String(), "notificationclick") {
		t.Fatalf("worker status=%d headers=%v", worker.Code, worker.Header())
	}
	page := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/push", nil)
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}))
	handler.adminPage(page, req)
	if page.Code != 403 {
		t.Fatalf("non-admin push access status=%d", page.Code)
	}
}
