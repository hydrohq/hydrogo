package mailer

import (
	"context"
	"strings"
	"testing"
)

type previewSender struct{ message Message }

func (s *previewSender) Send(_ context.Context, m Message) error { s.message = m; return nil }

func TestEmailVerificationNotifierIncludesOneUseLinkAsHTMLAndText(t *testing.T) {
	sender := &previewSender{}
	link := "https://hydro.example.test/auth/email/verify?token=opaque"
	if err := (NotificationSender{Sender: sender}).SendEmailVerification(context.Background(), "admin@example.test", link); err != nil {
		t.Fatal(err)
	}
	if sender.message.To != "admin@example.test" || !strings.Contains(sender.message.Body, link) || !strings.Contains(sender.message.HTMLBody, link) {
		t.Fatalf("verification email=%+v", sender.message)
	}
}
