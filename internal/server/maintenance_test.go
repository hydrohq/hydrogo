package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func TestMaintenanceMiddleware(t *testing.T) {
	mw := MaintenanceMiddleware(MaintenanceConfig{Enabled: func(*http.Request) bool { return true }, Render: func(*http.Request) ([]byte, error) { return []byte("maintenance page"), nil }, Allowlist: func(*http.Request) []string { return []string{"192.0.2.0/24"} }})
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := mw(next)
	cases := []struct {
		path, remote string
		want         int
	}{
		{"/", "198.51.100.2:99", 503}, {"/", "192.0.2.8:99", 200}, {"/login", "198.51.100.2:99", 200}, {"/designsystem/static/site.css", "198.51.100.2:99", 200},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.RemoteAddr = tc.remote
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("%s status=%d want %d", tc.path, w.Code, tc.want)
		}
		if tc.want == 503 && (w.Header().Get("Retry-After") != "300" || w.Header().Get("Cache-Control") != "no-store" || w.Body.String() != "maintenance page") {
			t.Errorf("maintenance response headers=%v body=%q", w.Header(), w.Body.String())
		}
	}
}

func TestMaintenanceMiddlewareAllowsActiveStaffOnly(t *testing.T) {
	handler := MaintenanceMiddleware(MaintenanceConfig{Enabled: func(*http.Request) bool { return true }})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	for _, tc := range []struct {
		role   auth.Role
		active bool
		want   int
	}{{auth.RoleAdmin, true, 200}, {auth.RoleEditor, true, 200}, {auth.RoleViewer, true, 503}, {auth.RoleAdmin, false, 503}} {
		r := httptest.NewRequest("GET", "/", nil)
		r = r.WithContext(auth.WithUser(r.Context(), &auth.User{Role: tc.role, Active: tc.active}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("role=%s active=%t status=%d want %d", tc.role, tc.active, w.Code, tc.want)
		}
	}
}

func TestMaintenancePreviewKeySetsShortLivedCookieAndRemovesQuery(t *testing.T) {
	const key = "long-enough-preview-key"
	config := MaintenanceConfig{
		Enabled:      func(*http.Request) bool { return true },
		SecureCookie: true,
		PreviewKey:   func(_ *http.Request, value string) bool { return value == key },
	}
	handler := MaintenanceMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	request := httptest.NewRequest(http.MethodGet, "/catalog?preview_key="+key+"&page=2", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/catalog?page=2" {
		t.Fatalf("preview response status=%d location=%q", w.Code, w.Header().Get("Location"))
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].MaxAge != 4*60*60 {
		t.Fatalf("preview cookie=%+v", cookies)
	}
	request = httptest.NewRequest(http.MethodGet, "/catalog", nil)
	request.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("preview cookie status=%d body=%s", w.Code, w.Body.String())
	}
}
