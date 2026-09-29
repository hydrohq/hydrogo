package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func TestMaintenanceMiddleware(t *testing.T) {
	mw := MaintenanceMiddleware(MaintenanceConfig{Enabled: func(*http.Request) bool { return true }, Render: func(*http.Request) ([]byte, error) { return []byte("maintenance page"), nil }, Allowlist: []string{"192.0.2.0/24"}})
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
