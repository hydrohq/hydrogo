package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdminLoginContainsAccessibleMFAChallenge(t *testing.T) {
	var output bytes.Buffer
	if err := RenderAdminLogin(&output, "csrf", "nonce", "/admin"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`id="mfa-challenge"`, `name="mfa_code"`, `data-mfa-countdown`, `data-mfa-next`, "Authenticator uygulamanızdaki güncel kodu"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("login page missing %q", want)
		}
	}
}
