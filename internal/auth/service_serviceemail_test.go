package auth

import "testing"

func TestIsServiceAccountEmail(t *testing.T) {
	for email, want := range map[string]bool{
		"service+seed@hydrogo.invalid":    true,
		"Service+Seed@HYDROGO.INVALID":    true,
		" service+x@hydrogo.invalid ":     true,
		"bob@example.com":                 false,
		"bob@hydrogo.invalid.example.com": false,
	} {
		if got := IsServiceAccountEmail(email); got != want {
			t.Errorf("IsServiceAccountEmail(%q) = %v, want %v", email, got, want)
		}
	}
}
