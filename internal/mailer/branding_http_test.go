package mailer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type brandingSettingsFake struct{ value EmailBranding }

func (s *brandingSettingsFake) ReadEmailBranding(context.Context) (EmailBranding, error) {
	return s.value, nil
}
func (s *brandingSettingsFake) WriteEmailBranding(_ context.Context, value EmailBranding) error {
	s.value = value
	return nil
}

func TestEmailBrandingHandlerRequiresAdminAndPersistsValidatedFields(t *testing.T) {
	settings := &brandingSettingsFake{}
	handler := EmailBrandingHandler{Settings: settings}
	route := protectedMailerMutation(t, "POST /admin/mailer/branding", handler)
	form := url.Values{"legal_name": {"Example Ltd"}, "physical_address": {"1 Main Street"}, "privacy_policy_url": {"https://example.test/privacy"}, "footer_notice": {"Terms apply"}, "footer_text": {"Manage preferences"}, "logo_asset_id": {"logo-1"}, "primary_color": {"#123456"}}
	request := httptest.NewRequest(http.MethodPost, "/admin/mailer/branding", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = withMailerCSRF(request, "valid-token")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}))
	response := httptest.NewRecorder()
	route.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || settings.value.LegalName != "Example Ltd" || settings.value.LogoAssetID != "logo-1" {
		t.Fatalf("status=%d saved=%+v", response.Code, settings.value)
	}
	request = httptest.NewRequest(http.MethodGet, "/admin/mailer/branding", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("editor access status=%d", response.Code)
	}
}

func TestEmailBrandingMutationRouteRejectsCSRFAndNonAdminRoles(t *testing.T) {
	settings := &brandingSettingsFake{}
	route := protectedMailerMutation(t, "POST /admin/mailer/branding", EmailBrandingHandler{Settings: settings})
	form := url.Values{"legal_name": {"Should not save"}, "primary_color": {"#123456"}}.Encode()
	for _, tc := range []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "admin with invalid csrf", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, token: "wrong-token"},
		{name: "editor with valid csrf", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "viewer with valid csrf", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, token: "valid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/admin/mailer/branding", strings.NewReader(form))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request = withMailerCSRF(request, tc.token)
			request = request.WithContext(auth.WithUser(request.Context(), tc.user))
			response := httptest.NewRecorder()
			route.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden || settings.value.LegalName != "" {
				t.Fatalf("status=%d saved=%+v", response.Code, settings.value)
			}
		})
	}
}

func protectedMailerMutation(t *testing.T, pattern string, handler http.Handler) http.Handler {
	t.Helper()
	mux, err := router.New([]router.Route{{Pattern: pattern, Access: router.Protected, Handler: handler}}, func(r *http.Request) bool {
		user := auth.UserFromContext(r.Context())
		return user != nil && user.Active
	})
	if err != nil {
		t.Fatal(err)
	}
	return security.CSRFMiddleware(false, nil)(mux)
}

func withMailerCSRF(request *http.Request, token string) *http.Request {
	request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
	request.Header.Set(security.CSRFHeaderName, token)
	return request
}
