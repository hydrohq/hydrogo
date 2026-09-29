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
)

type tokenLifetimeSettingsFake struct {
	value auth.TokenLifetimes
	set   bool
}

func (s *tokenLifetimeSettingsFake) Read(context.Context) (auth.TokenLifetimes, bool, error) {
	return s.value, s.set, nil
}
func (s *tokenLifetimeSettingsFake) Write(_ context.Context, value auth.TokenLifetimes) error {
	s.value, s.set = value, true
	return nil
}

func TestAdminTokenLifetimeSettingsApplyAndRejectUnsupportedChoices(t *testing.T) {
	settings := &tokenLifetimeSettingsFake{}
	handler := adminTokenLifetimeSettingsHandler{Settings: settings, Defaults: auth.TokenLifetimes{PasswordReset: time.Hour, Invitation: 24 * time.Hour}}
	request := func(reset, invite string, user *auth.User) *httptest.ResponseRecorder {
		form := url.Values{"password_reset_ttl": {reset}, "invitation_ttl": {invite}}
		req := httptest.NewRequest(http.MethodPost, "/admin/security/token-lifetimes", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = req.WithContext(auth.WithUser(req.Context(), user))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}
	admin := &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}
	if got := request("24h", "168h", admin); got.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", got.Code, got.Body.String())
	}
	if settings.value.PasswordReset != 24*time.Hour || settings.value.Invitation != 7*24*time.Hour {
		t.Fatalf("saved policy=%+v", settings.value)
	}
	if got := request("2h", "24h", admin); got.Code != http.StatusBadRequest {
		t.Fatalf("unsupported lifetime status=%d", got.Code)
	}
	if got := request("15m", "24h", &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}); got.Code != http.StatusForbidden {
		t.Fatalf("editor save status=%d", got.Code)
	}
}
