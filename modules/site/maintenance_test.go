package site

import "testing"

func TestMaintenanceSettingsArePrivateAndBounded(t *testing.T) {
	want := MaintenanceSettings{Enabled: true, Allowlist: []string{"192.0.2.1", "198.51.100.0/24"}}
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
}
