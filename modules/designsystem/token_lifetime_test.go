package designsystem

import (
	"strings"
	"testing"
)

func TestTokenLifetimeFormOffersApprovedDurations(t *testing.T) {
	got := render(t, TokenLifetimeForm(TokenLifetimeFormProps{CSRF: "csrf", Reset: "1h0m0s", Invitation: "72h0m0s"}))
	for _, marker := range []string{`name="password_reset_ttl"`, `value="15m"`, `value="24h"`, `name="invitation_ttl"`, `value="72h"`, `selected`, `name="_csrf"`} {
		if !strings.Contains(got, marker) {
			t.Errorf("token lifetime form missing %q", marker)
		}
	}
}
