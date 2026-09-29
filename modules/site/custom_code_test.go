package site

import "testing"

func TestCustomCodeSettingIsPrivateAndValidated(t *testing.T) {
	setting, err := (CustomCode{Head: `<script src="https://www.googletagmanager.com/gtm.js"></script>`}).Setting()
	if err != nil {
		t.Fatal(err)
	}
	if setting.Public || setting.Key != CustomCodeSettingKey {
		t.Fatalf("setting is not private: %#v", setting)
	}
	if got := CustomCodeFromSetting(setting); got.Head == "" {
		t.Fatal("valid code did not round trip")
	}
	setting.Public = true
	if got := CustomCodeFromSetting(setting); got.Head != "" {
		t.Fatal("public custom-code setting was exposed")
	}
	if _, err := (CustomCode{Body: `<script src="https://evil.example/x.js"></script>`}).Setting(); err == nil {
		t.Fatal("unknown provider was accepted")
	}
}
