package main

import (
	"bytes"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func TestLoginSuccessMessageFillsTheLoginStatusRegion(t *testing.T) {
	var page bytes.Buffer
	if err := ui.RenderAdminLoginWithLocaleAndAppearance(&page, "csrf", "/admin", "en", site.Appearance{LoginOverlayOpacity: .4}); err != nil {
		t.Fatal(err)
	}
	message := loginSuccessMessage(url.Values{"invite_accepted": {"1"}}, "en")
	got := string(withLoginSuccess(page.Bytes(), message))
	if !strings.Contains(got, `class="form-alert alert-success" role="status">Your account is ready.`) {
		t.Fatalf("login page did not show the invitation confirmation: %s", got)
	}
	if tr := loginSuccessMessage(url.Values{"password_reset": {"1"}}, "tr"); !strings.HasPrefix(tr, "Parolan güncellendi") {
		t.Fatalf("turkish reset message = %q", tr)
	}
	if plain := withLoginSuccess(page.Bytes(), loginSuccessMessage(url.Values{}, "en")); !bytes.Equal(plain, page.Bytes()) {
		t.Fatal("a plain sign-in changed the login page")
	}
	if escaped := string(withLoginSuccess(page.Bytes(), "<b>x</b>")); strings.Contains(escaped, "<b>x</b>") {
		t.Fatal("status message was not escaped")
	}
}
