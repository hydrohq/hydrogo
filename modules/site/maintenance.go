package site

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net"
	"strings"
	"time"
)

const MaintenanceSettingKey = "site.maintenance"

type MaintenanceSettings struct {
	Enabled        bool     `json:"enabled"`
	Allowlist      []string `json:"allowlist,omitempty"`
	StartsAt       string   `json:"starts_at,omitempty"`
	EndsAt         string   `json:"ends_at,omitempty"`
	PreviewKeyHash string   `json:"preview_key_hash,omitempty"`
	Headline       string   `json:"headline,omitempty"`
	Message        string   `json:"message,omitempty"`
}

func (v MaintenanceSettings) Validate() bool {
	if len(v.Allowlist) > 32 {
		return false
	}
	if len(v.Headline) > 160 || len(v.Message) > 1000 {
		return false
	}
	start, startErr := parseMaintenanceTime(v.StartsAt)
	end, endErr := parseMaintenanceTime(v.EndsAt)
	if startErr != nil || endErr != nil || (start != nil && end != nil && !start.Before(*end)) {
		return false
	}
	if v.PreviewKeyHash != "" {
		decoded, err := hex.DecodeString(v.PreviewKeyHash)
		if err != nil || len(decoded) != sha256.Size {
			return false
		}
	}
	for _, entry := range v.Allowlist {
		entry = strings.TrimSpace(entry)
		if net.ParseIP(entry) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err != nil {
			return false
		}
	}
	return true
}

func parseMaintenanceTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func (v MaintenanceSettings) ActiveAt(now time.Time) bool {
	if !v.Enabled {
		return false
	}
	start, _ := parseMaintenanceTime(v.StartsAt)
	end, _ := parseMaintenanceTime(v.EndsAt)
	now = now.UTC()
	return (start == nil || !now.Before(*start)) && (end == nil || now.Before(*end))
}

func MaintenancePreviewKeyHash(key string) string {
	key = strings.TrimSpace(key)
	if len(key) < 16 || len(key) > 128 {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (v MaintenanceSettings) AllowsPreviewKey(key string) bool {
	got := MaintenancePreviewKeyHash(key)
	return got != "" && len(v.PreviewKeyHash) == len(got) && subtle.ConstantTimeCompare([]byte(got), []byte(v.PreviewKeyHash)) == 1
}

func (v MaintenanceSettings) Setting() (Setting, error) {
	if !v.Validate() {
		return Setting{}, ErrInvalidSetting
	}
	data, err := json.Marshal(v)
	if err != nil {
		return Setting{}, ErrInvalidSetting
	}
	return Setting{Key: MaintenanceSettingKey, Value: string(data), Public: false}, nil
}

func MaintenanceFromSetting(setting Setting) MaintenanceSettings {
	var value MaintenanceSettings
	if setting.Key != MaintenanceSettingKey || setting.Public || json.Unmarshal([]byte(setting.Value), &value) != nil || !value.Validate() {
		return MaintenanceSettings{}
	}
	return value
}
