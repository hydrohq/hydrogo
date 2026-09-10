package commerce

import (
	"errors"
	"testing"
)

func TestValidateWebhookEvent(t *testing.T) {
	event := WebhookEvent{ID: "evt-1", Provider: "test", ProviderEventID: "provider-1", OrderID: "order-1", AmountCents: 100, Currency: "TRY", Status: "paid"}
	if err := ValidateWebhookEvent(event); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWebhookEvent(WebhookEvent{}); !errors.Is(err, ErrInvalidPaymentEvent) {
		t.Fatalf("invalid event = %v", err)
	}
}
