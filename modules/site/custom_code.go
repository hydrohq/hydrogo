package site

import (
	"encoding/json"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/customcode"
)

const CustomCodeSettingKey = "site.custom_code"

// CustomCode stores bounded, provider-validated site integration snippets.
type CustomCode struct {
	Head string `json:"custom_head_code"`
	Body string `json:"custom_body_end_code"`
}

func (c CustomCode) Setting() (Setting, error) {
	c.Head, c.Body = strings.TrimSpace(c.Head), strings.TrimSpace(c.Body)
	if _, err := customcode.Validate(c.Head, c.Body); err != nil {
		return Setting{}, ErrInvalidSetting
	}
	data, err := json.Marshal(c)
	if err != nil {
		return Setting{}, ErrInvalidSetting
	}
	return Setting{Key: CustomCodeSettingKey, Value: string(data), Public: false}, nil
}

func CustomCodeFromSetting(setting Setting) CustomCode {
	var code CustomCode
	if setting.Key != CustomCodeSettingKey || setting.Public || json.Unmarshal([]byte(setting.Value), &code) != nil {
		return CustomCode{}
	}
	if _, err := customcode.Validate(code.Head, code.Body); err != nil {
		return CustomCode{}
	}
	return code
}
