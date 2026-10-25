package mailer

import (
	"context"
	"strings"
	"testing"
)

type captureSender struct{ sent []Message }

func (c *captureSender) Send(_ context.Context, m Message) error {
	c.sent = append(c.sent, m)
	return nil
}

func TestNotificationSenderLinksOrFallsBackToToken(t *testing.T) {
	ctx := context.Background()

	linked := &captureSender{}
	n := NotificationSender{
		Sender:     linked,
		ResetLink:  func(token string) string { return "https://crm.example.test/admin/reset-password?token=" + token },
		InviteLink: func(token string) string { return "https://crm.example.test/admin/accept-invite?token=" + token },
	}
	if err := n.SendPasswordReset(ctx, "a@example.test", "R1"); err != nil {
		t.Fatal(err)
	}
	if err := n.SendInvite(ctx, "b@example.test", "I1"); err != nil {
		t.Fatal(err)
	}
	if body := linked.sent[0].Body; !strings.Contains(body, "https://crm.example.test/admin/reset-password?token=R1") || strings.Contains(body, "token: ") {
		t.Errorf("reset mail must carry the link: %q", body)
	}
	if body := linked.sent[1].Body; !strings.Contains(body, "https://crm.example.test/admin/accept-invite?token=I1") {
		t.Errorf("invite mail must carry the link: %q", body)
	}

	bare := &captureSender{}
	if err := (NotificationSender{Sender: bare}).SendInvite(ctx, "c@example.test", "I2"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bare.sent[0].Body, "I2") {
		t.Errorf("without a public origin the token must still be delivered: %q", bare.sent[0].Body)
	}
}
