package transport

import (
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

// A form's autoresponder sends to whatever e-mail the submitter typed. If
// its subject/body template could interpolate a free-text field, the site's
// own mail server becomes a relay: an attacker puts a victim's address in
// the e-mail field and attacker-written text in a free-text field that the
// template references, and the victim receives attacker content from the
// site's legitimate, DKIM-signed address.
func TestAutoresponderNeverInterpolatesFreeTextFields(t *testing.T) {
	form := forms.Form{
		Title:                  "Contact",
		Autoresponder:          true,
		AutoresponderSubject:   "Re: {{ticket_code}} — {{message}}",
		AutoresponderBody:      "Hello, about: {{message}} (dept {{department}})",
		NotificationRecipients: nil,
		Fields: []forms.Field{
			{ID: "email", Type: forms.Email},
			{ID: "message", Type: forms.Textarea},
			{ID: "department", Type: forms.Select, Options: []string{"sales", "support"}},
		},
	}
	values := map[string]string{
		"email":      "victim@example.test",
		"message":    "CLICK HERE http://evil.example/phish — free-text attacker payload",
		"department": "support",
	}
	notifications := formNotificationsAt(form, values, "form-abc123", time.Now().UTC())
	if len(notifications) != 1 {
		t.Fatalf("expected exactly 1 autoresponder notification, got %d", len(notifications))
	}
	n := notifications[0]
	if n.To != "victim@example.test" {
		t.Fatalf("autoresponder To = %q", n.To)
	}
	if strings.Contains(n.Subject, "attacker") || strings.Contains(n.Body, "attacker") || strings.Contains(n.Subject, "evil.example") || strings.Contains(n.Body, "evil.example") {
		t.Fatalf("free-text field value leaked into the autoresponder: subject=%q body=%q", n.Subject, n.Body)
	}
	if !strings.Contains(n.Body, "dept support") {
		t.Fatalf("a bounded-choice field (select) must still be available to the template, got body=%q", n.Body)
	}
	if !strings.Contains(n.Subject, "ABC123") {
		t.Fatalf("ticket_code must still interpolate, got subject=%q", n.Subject)
	}
}
