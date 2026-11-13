package site

import (
	"testing"
	"time"
)

func TestBusinessProfileSettingAndOpenHours(t *testing.T) {
	profile := BusinessProfile{Name: "Studio", Address: "1 Main St", Timezone: "Europe/Istanbul", Hours: map[time.Weekday]BusinessHours{time.Monday: {Opens: "09:00", Closes: "18:00"}}}
	setting, err := profile.ToSetting()
	if err != nil {
		t.Fatal(err)
	}
	decoded, ok := BusinessProfileFromSettings([]Setting{setting})
	if !ok || decoded.Name != "Studio" {
		t.Fatalf("profile decode=%+v/%v", decoded, ok)
	}
	open := time.Date(2026, 9, 28, 12, 0, 0, 0, time.FixedZone("Istanbul", 3*60*60))
	if !profile.IsOpenAt(open) || profile.IsOpenAt(open.Add(7*time.Hour)) {
		t.Fatal("business hours status is wrong")
	}
	profile.Website = "javascript:alert(1)"
	if profile.Validate() == nil {
		t.Fatal("unsafe business URL accepted")
	}
}

func TestBusinessProfileExtendedCorporateFieldsAndDefaults(t *testing.T) {
	profile := BusinessProfile{
		Name:       "Atelier",
		LegalName:  "Atelier Deri Ltd. Şti.",
		TaxID:      "1234567890",
		TaxOffice:  "Beyoğlu",
		MersisNo:   "0123456789000001",
		KepAddress: "atelier@hs01.kep.tr",
		City:       "İstanbul",
		Country:    "Türkiye",
		PostalCode: "34430",
	}
	setting, err := profile.ToSetting()
	if err != nil {
		t.Fatalf("extended profile rejected: %v", err)
	}
	decoded, ok := BusinessProfileFromSettings([]Setting{setting})
	if !ok || decoded.Timezone != "UTC" || decoded.LegalName != profile.LegalName || decoded.TaxOffice != profile.TaxOffice {
		t.Fatalf("decoded=%+v ok=%v", decoded, ok)
	}
	ld, err := BusinessJSONLD(decoded)
	if err != nil || ld == "" {
		t.Fatalf("BusinessJSONLD err=%v ld=%q", err, ld)
	}
	invalidKep := profile
	invalidKep.KepAddress = "not-an-email"
	if invalidKep.Validate() == nil {
		t.Fatal("invalid KEP address accepted")
	}
}
