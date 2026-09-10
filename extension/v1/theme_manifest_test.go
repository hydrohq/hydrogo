package v1

import "testing"

func TestThemeManifestValidation(t *testing.T) {
	manifest := ThemeManifest{Name: "corporate", Version: "1.0.0", Tokens: map[string]string{"primary": "#123456"}, Slots: []string{"header", "content"}, Assets: []string{"theme.css"}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	manifest.Name = "../unsafe"
	if err := manifest.Validate(); err == nil {
		t.Fatal("unsafe manifest accepted")
	}
}
