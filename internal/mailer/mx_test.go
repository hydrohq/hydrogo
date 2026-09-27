package mailer

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
)

func TestMXSenderTriesSortedRecipientMXAndContinuesAfterFailure(t *testing.T) {
	var lookedUp string
	var tried []string
	sender := MXSender{
		From: "contact@sender.example",
		LookupMX: func(_ context.Context, domain string) ([]*net.MX, error) {
			lookedUp = domain
			return []*net.MX{{Host: "backup.example.", Pref: 20}, {Host: "primary.example.", Pref: 10}}, nil
		},
		DeliverMX: func(_ context.Context, host, from string, message Message) error {
			tried = append(tried, host+"|"+from+"|"+message.To)
			if host == "primary.example" {
				return errors.New("connection refused")
			}
			return nil
		},
	}
	if err := sender.Send(context.Background(), Message{To: "visitor@recipient.example", Subject: "Received", Body: "Thanks"}); err != nil {
		t.Fatal(err)
	}
	if lookedUp != "recipient.example" {
		t.Fatalf("lookup domain = %q", lookedUp)
	}
	want := []string{"primary.example|contact@sender.example|visitor@recipient.example", "backup.example|contact@sender.example|visitor@recipient.example"}
	if !reflect.DeepEqual(tried, want) {
		t.Fatalf("MX attempts = %#v, want %#v", tried, want)
	}
}

func TestMXSenderRejectsMissingSenderAndMX(t *testing.T) {
	message := Message{To: "visitor@example.test", Subject: "Received"}
	if err := (MXSender{}).Send(context.Background(), message); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("missing from error = %v", err)
	}
	sender := MXSender{From: "contact@example.test", LookupMX: func(context.Context, string) ([]*net.MX, error) { return nil, nil }}
	if err := sender.Send(context.Background(), message); err == nil {
		t.Fatal("empty MX list accepted")
	}
}
