package mailer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
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
	form := url.Values{"legal_name": {"Example Ltd"}, "physical_address": {"1 Main Street"}, "privacy_policy_url": {"https://example.test/privacy"}, "footer_notice": {"Terms apply"}, "footer_text": {"Manage preferences"}, "logo_asset_id": {"logo-1"}, "primary_color": {"#123456"}}
	request := httptest.NewRequest(http.MethodPost, "/admin/mailer/branding", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
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
