package i18n

import "testing"

func TestCatalogTranslatesEnglishAndFallsBack(t *testing.T) {
	if got := T("en-US", "admin.pages"); got != "Pages" {
		t.Fatalf("English translation = %q", got)
	}
	if got := T("tr-TR", "admin.pages"); got != "Sayfalar" {
		t.Fatalf("Turkish translation = %q", got)
	}
	if got := T("en", "missing.key"); got != "missing.key" {
		t.Fatalf("missing translation = %q", got)
	}
}
