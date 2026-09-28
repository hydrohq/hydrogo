package config

import "testing"

func TestLoadMediaWatermarkAndHotlinkOptions(t *testing.T) {
	path := writeConfig(t, "media:\n  watermark_text: '© HydroGo'\n  watermark_position: bottom-left\n  watermark_opacity: 0.4\n  watermark_margin: 18\n")
	cfg, err := Load(path, lookup(map[string]string{
		"HYDRO_DATABASE_DSN":                "file:test.sqlite",
		"HYDRO_MEDIA_HOTLINK_PROTECTION":    "true",
		"HYDRO_MEDIA_ALLOWED_REFERER_HOSTS": "news.example, partner.example",
	}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Media.HotlinkProtection || len(cfg.Media.AllowedRefererHosts) != 2 || cfg.Media.WatermarkPosition != "bottom-left" || cfg.Media.WatermarkOpacity != .4 || cfg.Media.WatermarkMargin != 18 {
		t.Fatalf("media options = %+v", cfg.Media)
	}
	path = writeConfig(t, "media:\n  watermark_text: mark\n  watermark_opacity: 0.05\n")
	if _, err = Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{}); err == nil {
		t.Fatal("invalid watermark opacity accepted")
	}
}
