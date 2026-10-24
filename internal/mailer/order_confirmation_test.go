package mailer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type capturingSender struct{ got Message }

func (s *capturingSender) Send(_ context.Context, msg Message) error {
	s.got = msg
	return nil
}

func TestSendOrderConfirmationIncludesOrderIDTotalAndItems(t *testing.T) {
	capture := &capturingSender{}
	sender := NotificationSender{Sender: capture}
	items := []OrderConfirmationItem{{Title: "Deri Kemer", Quantity: 2, Total: "240,00 ₺"}}

	if err := sender.SendOrderConfirmation(context.Background(), "guest@example.test", "order_abc", "240,00 ₺", items); err != nil {
		t.Fatal(err)
	}
	if capture.got.To != "guest@example.test" {
		t.Fatalf("to = %q", capture.got.To)
	}
	if !strings.Contains(capture.got.Subject, "order_abc") {
		t.Fatalf("subject = %q, want it to mention the order id", capture.got.Subject)
	}
	if !strings.Contains(capture.got.Body, "order_abc") || !strings.Contains(capture.got.Body, "240,00 ₺") {
		t.Fatalf("body = %q, want it to mention the order id and total", capture.got.Body)
	}
	if !strings.Contains(capture.got.HTMLBody, "Deri Kemer") || !strings.Contains(capture.got.HTMLBody, "240,00 ₺") {
		t.Fatalf("html body = %q, want it to list the item", capture.got.HTMLBody)
	}
}

func TestSendOrderConfirmationFailsClosedWithoutTransport(t *testing.T) {
	sender := NotificationSender{}
	if err := sender.SendOrderConfirmation(context.Background(), "guest@example.test", "order_abc", "100 TRY", nil); !errors.Is(err, ErrSenderRequired) {
		t.Fatalf("error = %v, want %v", err, ErrSenderRequired)
	}
}
