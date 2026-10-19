package transport

import "testing"

func TestValidBrandColor(t *testing.T) {
	for _, ok := range []string{"", "#18181b", "#FFAA00"} {
		if !ValidBrandColor(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"red", "#fff", "#12345g", "18181b", "#18181b;x"} {
		if ValidBrandColor(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
