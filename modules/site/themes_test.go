package site

import "testing"

func TestValidateThemeID(t *testing.T) {
	if !ValidateThemeID("") {
		t.Fatal("empty theme id should be allowed")
	}
	for _, id := range []string{"minimal", "boutique", "editorial", "b2b-catalog"} {
		if !ValidateThemeID(id) {
			t.Fatalf("expected valid theme id %q", id)
		}
	}
	if ValidateThemeID("unknown-theme") || ValidateThemeID("../etc/passwd") {
		t.Fatal("expected unknown or unsafe theme id to be rejected")
	}
}

func TestValidateThemeTokensJSON(t *testing.T) {
	if !ValidateThemeTokensJSON("") {
		t.Fatal("empty tokens should be allowed")
	}
	if !ValidateThemeTokensJSON(`{"accent":"#00add8","radius":"md","header_variant":"centered"}`) {
		t.Fatal("valid partial tokens should be accepted")
	}
	cases := []string{
		`{"accent":"red;background:url(javascript:alert(1))"}`,
		`{"unknown":"value"}`,
		`{"radius":"999px"}`,
		`{"font_display":"Comic Sans MS"}`,
		`{"header_variant":"evil"}`,
		`not-json`,
	}
	for _, tc := range cases {
		if ValidateThemeTokensJSON(tc) {
			t.Fatalf("expected invalid tokens to be rejected: %s", tc)
		}
	}
}

func TestResolveTheme(t *testing.T) {
	theme, tokens := ResolveTheme("boutique", `{"accent":"#112233","radius":"lg"}`, "")
	if theme.ID != "boutique" {
		t.Fatalf("expected boutique, got %q", theme.ID)
	}
	if tokens.Accent != "#112233" || tokens.Radius != "lg" || tokens.Bg != theme.DefaultTokens.Bg {
		t.Fatalf("unexpected merged tokens: %+v", tokens)
	}

	previewTheme, previewTokens := ResolveTheme("boutique", `{"accent":"#112233"}`, "editorial")
	if previewTheme.ID != "editorial" {
		t.Fatalf("expected preview theme editorial, got %q", previewTheme.ID)
	}
	if previewTokens.Accent != previewTheme.DefaultTokens.Accent {
		t.Fatalf("preview should use target theme default tokens, got %+v", previewTokens)
	}
}
