package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const StaffNoticeSettingKey = "site.staff_notice"

type StaffBroadcastNotice struct {
	Active    bool   `json:"active"`
	Message   string `json:"message"`
	Severity  string `json:"severity"`
	ExpiresAt string `json:"expires_at"`
}

func (n StaffBroadcastNotice) Validate() bool {
	if len(strings.TrimSpace(n.Message)) == 0 || len(n.Message) > 1000 {
		return false
	}
	if n.Severity != "info" && n.Severity != "warning" && n.Severity != "critical" {
		return false
	}
	_, err := time.Parse(time.RFC3339, n.ExpiresAt)
	return err == nil
}

func (n StaffBroadcastNotice) ActiveAt(now time.Time) bool {
	expires, err := time.Parse(time.RFC3339, n.ExpiresAt)
	return n.Active && err == nil && now.Before(expires)
}

func (n StaffBroadcastNotice) DismissKey() string {
	data, _ := json.Marshal(n)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}

func (n StaffBroadcastNotice) Setting() (Setting, error) {
	if !n.Validate() {
		return Setting{}, ErrInvalidSetting
	}
	data, err := json.Marshal(n)
	if err != nil {
		return Setting{}, ErrInvalidSetting
	}
	return Setting{Key: StaffNoticeSettingKey, Value: string(data), Public: false}, nil
}

func StaffBroadcastNoticeFromSetting(setting Setting) StaffBroadcastNotice {
	var notice StaffBroadcastNotice
	if setting.Key != StaffNoticeSettingKey || setting.Public || json.Unmarshal([]byte(setting.Value), &notice) != nil || !notice.Validate() {
		return StaffBroadcastNotice{}
	}
	return notice
}
