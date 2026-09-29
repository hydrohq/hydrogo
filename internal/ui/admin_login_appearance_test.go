package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func TestAdminLoginUsesValidatedBackgroundAndOverlay(t *testing.T) {
	var output bytes.Buffer
	appearance := site.Appearance{LoginBackgroundURL: "/media/hero.webp", LoginOverlayOpacity: .35}
	if err := RenderAdminLoginWithAppearance(&output, "csrf", "nonce", "/admin", appearance); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `linear-gradient(rgba(9,10,15,0.35),rgba(9,10,15,0.35)),url("/media/hero.webp")`) {
		t.Fatal("validated login appearance missing")
	}
	output.Reset()
	appearance.LoginBackgroundURL = "https://tracker.example/pixel"
	if err := RenderAdminLoginWithAppearance(&output, "csrf", "nonce", "/admin", appearance); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "tracker.example") {
		t.Fatal("untrusted background leaked into login CSS")
	}
}
