package site

import (
	"encoding/json"
	"net"
	"strings"
)

const MaintenanceSettingKey = "site.maintenance"

type MaintenanceSettings struct {
	Enabled   bool     `json:"enabled"`
	Allowlist []string `json:"allowlist,omitempty"`
}

func (v MaintenanceSettings) Validate() bool {
	if len(v.Allowlist) > 32 {
		return false
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
