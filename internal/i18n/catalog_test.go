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

func TestResolveCatalogLocaleAndValidation(t *testing.T) {
	if got := ResolveCatalogLocale("en-US"); got != "en" {
		t.Fatalf("resolved locale = %q", got)
	}
	if got := ResolveCatalogLocale("de"); got != "tr" {
		t.Fatalf("unsupported locale fallback = %q", got)
	}
	if !IsSupportedCatalogLocale("tr-TR") || IsSupportedCatalogLocale("de-DE") {
		t.Fatal("locale validation did not enforce catalog allowlist")
	}
}

func TestTenantOverridesAreBoundedAndFallBackToPrimaryCatalog(t *testing.T) {
	const tenantID = "catalog-test-tenant"
	defer ReplaceOverrides(tenantID, nil)
	if got := TForTenant(tenantID, "en", "admin.pages"); got != "Pages" {
		t.Fatalf("initial locale value = %q", got)
	}
	if !SetOverride(tenantID, "en", "admin.pages", "Editorial pages") {
		t.Fatal("valid override was rejected")
	}
	if got := TForTenant(tenantID, "en", "admin.pages"); got != "Editorial pages" {
		t.Fatalf("tenant override = %q", got)
	}
	if got := TForTenant("another-tenant", "en", "admin.pages"); got != "Pages" {
		t.Fatalf("override leaked across tenants: %q", got)
	}
	if got := TForTenant(tenantID, "xx", "admin.language.tr"); got != "Türkçe" {
		t.Fatalf("missing locale fallback = %q", got)
	}
	if SetOverride(tenantID, "en", "unknown.key", "value") || SetOverride(tenantID, "en", "admin.pages", "bad\nvalue") {
		t.Fatal("invalid override was accepted")
	}
	rows := CatalogRows(tenantID, "en")
	for _, row := range rows {
		if row.Key == "admin.pages" && (row.Missing || row.Translation != "Editorial pages") {
			t.Fatalf("override row = %#v", row)
		}
	}
}

func TestOverrideSettingsOnlyAcceptKnownCatalogEntries(t *testing.T) {
	got := OverrideSettings(map[string]string{
		OverrideSettingKey("en", "admin.pages"): "Pages (custom)",
		"i18n.override.xx.admin.pages":          "unsupported locale",
		"i18n.override.en.unknown":              "unknown key",
	})
	if got["en"]["admin.pages"] != "Pages (custom)" || len(got["en"]) != 1 {
		t.Fatalf("parsed overrides = %#v", got)
	}
}
