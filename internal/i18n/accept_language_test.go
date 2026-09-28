package i18n

import "testing"

func TestResolveAcceptLanguage(t *testing.T) {
	supported := []string{"tr", "en", "pt-BR"}
	cases := []struct{ header, want string }{
		{"en-US,en;q=0.9,tr;q=0.8", "en"},
		{"fr-CA,pt;q=0.9", "pt-BR"},
		{"en;q=0,tr;q=0.5", "tr"},
		{"en;q=1.5,tr;q=0.7", "tr"},
		{"*;q=1", "tr"},
		{"fr", "tr"},
		{"tr;q=0.8,en;q=0.8", "tr"},
	}
	for _, tc := range cases {
		t.Run(tc.header, func(t *testing.T) {
			if got := ResolveAcceptLanguage(tc.header, supported, "tr"); got != tc.want {
				t.Fatalf("ResolveAcceptLanguage(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}
