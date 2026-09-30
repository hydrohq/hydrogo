package canonical

import "testing"

func TestSlugifyTurkishAndUnicode(t *testing.T) {
	for input, want := range map[string]string{"İstanbul Şişli Çarşı": "istanbul-sisli-carsi", "IĞDIR İzmir": "igdir-izmir", "  À la carte  ": "a-la-carte", "Çağrı #1": "cagri-1"} {
		if got := Slugify(input); got != want {
			t.Errorf("Slugify(%q) = %q; want %q", input, got, want)
		}
	}
}
