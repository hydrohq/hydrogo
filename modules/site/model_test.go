package site

import "testing"

func TestResolvePublicHomePath(t *testing.T) {
	fallback := "/public/pages/en/demo/html"
	cases := []struct {
		name     string
		settings []Setting
		want     string
	}{
		{name: "public internal path", settings: []Setting{{Key: "site.home_path", Value: "/public/pages/en/home/html", Public: true}}, want: "/public/pages/en/home/html"},
		{name: "private ignored", settings: []Setting{{Key: "site.home_path", Value: "/private", Public: false}}, want: fallback},
		{name: "external ignored", settings: []Setting{{Key: "site.home_path", Value: "https://evil.example", Public: true}}, want: fallback},
		{name: "network path ignored", settings: []Setting{{Key: "site.home_path", Value: "//evil.example", Public: true}}, want: fallback},
		{name: "newline ignored", settings: []Setting{{Key: "site.home_path", Value: "/home\nnext", Public: true}}, want: fallback},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolvePublicHomePath(tc.settings, fallback); got != tc.want {
				t.Fatalf("ResolvePublicHomePath() = %q, want %q", got, tc.want)
			}
		})
	}
}
