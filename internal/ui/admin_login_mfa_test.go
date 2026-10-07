package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func TestAdminLoginContainsAccessibleMFAChallenge(t *testing.T) {
	var output bytes.Buffer
	if err := RenderAdminLogin(&output, "csrf", "/admin"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`id="mfa-challenge"`, `name="mfa_code"`, `data-mfa-countdown`, `data-mfa-next`, "Authenticator uygulamanızdaki güncel kodu"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("login page missing %q", want)
		}
	}
}

func TestAdminLoginLocalesAndReturnPath(t *testing.T) {
	for _, test := range []struct {
		locale string
		want   []string
		absent []string
	}{
		{locale: "en", want: []string{`<html lang="en"`, `Welcome back`, `Sign in`, `Two-step verification`, `Code expires in `, `href="/login?lang=tr&amp;return_to=%2Fadmin%2Fusers"`}, absent: []string{`Hoş Geldiniz`, `İki adımlı doğrulama`, `Giriş yapılıyor`}},
		{locale: "tr", want: []string{`<html lang="tr"`, `Hoş Geldiniz`, `Giriş Yap`, `İki adımlı doğrulama`, `Kodun süresi `, `href="/login?lang=en&amp;return_to=%2Fadmin%2Fusers"`}, absent: []string{`Welcome back`, `Two-step verification`, `The workspace for modern publishing`}},
	} {
		var output bytes.Buffer
		if err := RenderAdminLoginWithLocaleAndAppearance(&output, "csrf", "/admin/users", test.locale, site.Appearance{}); err != nil {
			t.Fatal(err)
		}
		for _, want := range test.want {
			if !strings.Contains(output.String(), want) {
				t.Errorf("%s login is missing %q", test.locale, want)
			}
		}
		for _, absent := range test.absent {
			if strings.Contains(output.String(), absent) {
				t.Errorf("%s login contains untranslated %q", test.locale, absent)
			}
		}
	}
}

func TestAdminLoginCSPHashesRenderedInlineBlocks(t *testing.T) {
	var output bytes.Buffer
	if err := RenderAdminLoginWithLocaleAndAppearance(&output, "csrf", "/admin", "en", site.Appearance{}); err != nil {
		t.Fatal(err)
	}
	policy := AdminLoginCSP(output.Bytes())
	if strings.Count(policy, "'sha256-") != 3 || !strings.Contains(policy, "script-src") || !strings.Contains(policy, "style-src") {
		t.Fatalf("unexpected login CSP hashes: %s", policy)
	}
	if AdminLoginCSP([]byte(`<html><script>alert(1)</script></html>`)) == "" {
		t.Fatal("script hash should be generated for an inline block")
	}
}
