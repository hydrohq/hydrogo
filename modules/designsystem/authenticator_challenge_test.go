package designsystem

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAuthenticatorChallengeHasAccessibleTOTPControls(t *testing.T) {
	var output bytes.Buffer
	if err := AuthenticatorChallenge().Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"id=\"mfa-challenge\"", `name="mfa_code"`, `autocomplete="one-time-code"`, `aria-live="polite"`, "Sonraki kodu kullan"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("authenticator challenge missing %q: %s", want, output.String())
		}
	}
}
