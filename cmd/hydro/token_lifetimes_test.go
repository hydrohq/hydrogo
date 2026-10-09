package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type tokenLifetimeSettingsFake struct {
	value  auth.TokenLifetimes
	set    bool
	writes int
	reads  int
}

func (s *tokenLifetimeSettingsFake) Read(context.Context) (auth.TokenLifetimes, bool, error) {
	s.reads++
	return s.value, s.set, nil
}
func (s *tokenLifetimeSettingsFake) Write(_ context.Context, value auth.TokenLifetimes) error {
	s.value, s.set = value, true
	s.writes++
	return nil
}

func TestAdminTokenLifetimeSettingsApplyAndRejectUnsupportedChoices(t *testing.T) {
	settings := &tokenLifetimeSettingsFake{}
	handler := adminTokenLifetimeSettingsHandler{Settings: settings, Defaults: auth.TokenLifetimes{PasswordReset: time.Hour, Invitation: 24 * time.Hour}}
	routes, err := router.New([]router.Route{
		{Pattern: "GET /admin/security/token-lifetimes", Access: router.Protected, Handler: handler},
		{Pattern: "POST /admin/security/token-lifetimes", Access: router.Protected, Handler: handler},
	}, func(*http.Request) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	protected := security.CSRFMiddleware(false, nil)(routes)
	request := func(method, reset, invite, csrf string, user *auth.User) *httptest.ResponseRecorder {
		form := url.Values{"password_reset_ttl": {reset}, "invitation_ttl": {invite}, "_csrf": {csrf}}
		var body *strings.Reader
		body = strings.NewReader(form.Encode())
		if method == http.MethodGet {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, "/admin/security/token-lifetimes", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if user != nil {
			req = req.WithContext(auth.WithUser(req.Context(), user))
		}
		req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		recorder := httptest.NewRecorder()
		protected.ServeHTTP(recorder, req)
		return recorder
	}
	admin := &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}
	if got := request(http.MethodPost, "24h", "168h", "invalid-token", admin); got.Code != http.StatusForbidden || settings.writes != 0 {
		t.Fatalf("invalid CSRF status=%d writes=%d", got.Code, settings.writes)
	}
	if got := request(http.MethodPost, "24h", "168h", "valid-token", admin); got.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", got.Code, got.Body.String())
	}
	if settings.value.PasswordReset != 24*time.Hour || settings.value.Invitation != 7*24*time.Hour || settings.writes != 1 {
		t.Fatalf("saved policy=%+v", settings.value)
	}
	baseline := settings.value
	reads := settings.reads
	for _, tc := range []struct {
		name string
		user *auth.User
	}{
		{name: "editor", user: &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}},
		{name: "viewer", user: &auth.User{ID: "viewer", Active: true, Role: auth.RoleViewer}},
		{name: "inactive admin", user: &auth.User{ID: "inactive", Active: false, Role: auth.RoleAdmin}},
		{name: "editor operations grant", user: &auth.User{ID: "editor-ops", Active: true, Role: auth.RoleEditor, PermissionOverrides: map[string]bool{auth.PermissionOperations: true}}},
		{name: "viewer operations grant", user: &auth.User{ID: "viewer-ops", Active: true, Role: auth.RoleViewer, PermissionOverrides: map[string]bool{auth.PermissionOperations: true}}},
		{name: "anonymous", user: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := request(http.MethodPost, "48h", "120h", "valid-token", tc.user); got.Code != http.StatusForbidden {
				t.Fatalf("valid-CSRF POST status=%d body=%s", got.Code, got.Body.String())
			}
			if got := request(http.MethodGet, "", "", "", tc.user); got.Code != http.StatusForbidden {
				t.Fatalf("GET status=%d body=%s", got.Code, got.Body.String())
			}
			if settings.value != baseline || settings.writes != 1 || settings.reads != reads {
				t.Fatalf("denied requests touched persisted state: value=%+v writes=%d reads=%d", settings.value, settings.writes, settings.reads)
			}
		})
	}
	if got := request(http.MethodPost, "2h", "24h", "valid-token", admin); got.Code != http.StatusBadRequest || settings.writes != 1 || settings.value != baseline {
		t.Fatalf("unsupported lifetime status=%d", got.Code)
	}
	if got := request(http.MethodPost, "48h", "120h", "invalid-token", admin); got.Code != http.StatusForbidden || settings.writes != 1 || settings.value != baseline {
		t.Fatalf("invalid-CSRF lifetime update status=%d", got.Code)
	}
}
