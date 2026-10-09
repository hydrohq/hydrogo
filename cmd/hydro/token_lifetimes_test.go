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
	value auth.TokenLifetimes
	set   bool
	writes int
}

func (s *tokenLifetimeSettingsFake) Read(context.Context) (auth.TokenLifetimes, bool, error) {
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
	routes, err := router.New([]router.Route{{Pattern: "POST /admin/security/token-lifetimes", Access: router.Protected, Handler: handler}}, func(*http.Request) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	protected := security.CSRFMiddleware(false, nil)(routes)
	request := func(reset, invite, csrf string, user *auth.User) *httptest.ResponseRecorder {
		form := url.Values{"password_reset_ttl": {reset}, "invitation_ttl": {invite}, "_csrf": {csrf}}
		req := httptest.NewRequest(http.MethodPost, "/admin/security/token-lifetimes", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = req.WithContext(auth.WithUser(req.Context(), user))
		req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		recorder := httptest.NewRecorder()
		protected.ServeHTTP(recorder, req)
		return recorder
	}
	admin := &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}
	if got := request("24h", "168h", "invalid-token", admin); got.Code != http.StatusForbidden || settings.writes != 0 {
		t.Fatalf("invalid CSRF status=%d writes=%d", got.Code, settings.writes)
	}
	if got := request("24h", "168h", "valid-token", admin); got.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", got.Code, got.Body.String())
	}
	if settings.value.PasswordReset != 24*time.Hour || settings.value.Invitation != 7*24*time.Hour || settings.writes != 1 {
		t.Fatalf("saved policy=%+v", settings.value)
	}
	if got := request("2h", "24h", "valid-token", admin); got.Code != http.StatusBadRequest || settings.writes != 1 {
		t.Fatalf("unsupported lifetime status=%d", got.Code)
	}
	if got := request("15m", "24h", "valid-token", &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}); got.Code != http.StatusForbidden || settings.writes != 1 {
		t.Fatalf("editor save status=%d", got.Code)
	}
}
