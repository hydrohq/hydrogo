package site

import (
	"testing"
	"time"
)

func TestStaffBroadcastNoticePrivateValidationAndExpiry(t *testing.T) {
	notice := StaffBroadcastNotice{Active: true, Severity: "warning", Message: "Deployment begins soon", ExpiresAt: "2026-09-29T12:00:00Z"}
	setting, err := notice.Setting()
	if err != nil || setting.Public {
		t.Fatalf("setting=%+v err=%v", setting, err)
	}
	loaded := StaffBroadcastNoticeFromSetting(setting)
	if !loaded.ActiveAt(time.Date(2026, 9, 29, 11, 59, 0, 0, time.UTC)) || loaded.ActiveAt(time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("notice activation window is incorrect")
	}
	if loaded.DismissKey() != notice.DismissKey() {
		t.Fatal("notice dismissal key is not stable")
	}
	for _, invalid := range []StaffBroadcastNotice{
		{Severity: "warning", Message: "", ExpiresAt: notice.ExpiresAt},
		{Severity: "urgent", Message: "x", ExpiresAt: notice.ExpiresAt},
		{Severity: "info", Message: "x", ExpiresAt: "invalid"},
	} {
		if invalid.Validate() {
			t.Errorf("invalid notice accepted: %+v", invalid)
		}
	}
}
