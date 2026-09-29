package designsystem

import (
	"strings"
	"testing"
)

func TestEmailBrandingFormEscapesSettingsAndIncludesCSRF(t *testing.T) {
	got := render(t, EmailBrandingForm(EmailBrandingFormProps{CSRF: "token", LegalName: `<script>alert(1)</script>`, PrimaryColor: "#abcdef"}))
	for _, marker := range []string{`name="_csrf" value="token"`, `name="legal_name"`, `name="logo_asset_id"`, `data-media-target="email-brand-logo-id"`, `name="primary_color"`, `href="/admin/media"`, "&lt;script&gt;"} {
		if !strings.Contains(got, marker) {
			t.Errorf("email settings form missing %q", marker)
		}
	}
	if strings.Contains(got, `<script>alert(1)</script>`) {
		t.Fatal("legal name was rendered without escaping")
	}
}
