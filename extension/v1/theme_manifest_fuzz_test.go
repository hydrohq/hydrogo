package v1

import "testing"

func FuzzSafeThemePath(f *testing.F) {
	for _, seed := range []string{"static/theme.css", "../secret", "", "//absolute", "blocks/../static/theme.js", "static\\theme.css"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_ = safeThemePath(value)
	})
}

func FuzzThemeManifestValidate(f *testing.F) {
	f.Add("corporate", "1.0.0", "static/theme.css")
	f.Add("", "", "../secret")
	f.Fuzz(func(t *testing.T, name, version, asset string) {
		manifest := ThemeManifest{Name: name, Version: version, Assets: []string{asset}}
		_ = manifest.Validate()
	})
}
