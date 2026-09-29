package site

import (
	"testing"
	"time"
)

func TestMaintenanceSettingsArePrivateAndBounded(t *testing.T) {
	want := MaintenanceSettings{Enabled: true, Allowlist: []string{"192.0.2.1", "198.51.100.0/24"}, StartsAt: "2026-09-29T02:00:00Z", EndsAt: "2026-09-29T04:00:00Z", Headline: "We’ll be back", Message: "Please visit later."}
	setting, err := want.Setting()
	if err != nil || setting.Public {
		t.Fatalf("setting=%#v err=%v", setting, err)
	}
	if got := MaintenanceFromSetting(setting); !got.Enabled || len(got.Allowlist) != 2 {
		t.Fatalf("round trip=%#v", got)
	}
	if _, err := (MaintenanceSettings{Allowlist: []string{"not an address"}}).Setting(); err == nil {
		t.Fatal("invalid address accepted")
	}
	if _, err := (MaintenanceSettings{Enabled: true, StartsAt: "2026-09-29T04:00:00Z", EndsAt: "2026-09-29T02:00:00Z"}).Setting(); err == nil {
		t.Fatal("reversed maintenance window accepted")
	}
}

func TestMaintenanceWindowAndPreviewKey(t *testing.T) {
	settings := MaintenanceSettings{Enabled: true, StartsAt: "2026-09-29T02:00:00Z", EndsAt: "2026-09-29T04:00:00Z"}
	if settings.ActiveAt(time.Date(2026, 9, 29, 1, 59, 0, 0, time.UTC)) || !settings.ActiveAt(time.Date(2026, 9, 29, 2, 0, 0, 0, time.UTC)) || settings.ActiveAt(time.Date(2026, 9, 29, 4, 0, 0, 0, time.UTC)) {
		t.Fatal("maintenance window boundaries were not respected")
	}
	key := "long-enough-preview-key"
	settings.PreviewKeyHash = MaintenancePreviewKeyHash(key)
	if !settings.AllowsPreviewKey(key) || settings.AllowsPreviewKey(key+"wrong") || MaintenancePreviewKeyHash("short") != "" {
		t.Fatal("preview key validation failed")
	}
}
