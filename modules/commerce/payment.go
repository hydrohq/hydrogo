package commerce

import (
	"context"
	"errors"
	"strings"
	"time"

	"net/http"
)

type PaymentIntent struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	OrderID     string `json:"order_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	ClientToken string `json:"client_token,omitempty"`
}

type WebhookEvent struct {
	ID              string
	Provider        string
	ProviderEventID string
	OrderID         string
	PaymentID       string
	Status          string
	AmountCents     int64
	Currency        string
	OccurredAt      time.Time
}

type PaymentProvider interface {
	Name() string
	CreatePaymentIntent(context.Context, Order) (PaymentIntent, error)
	VerifyWebhookSignature(*http.Request) (WebhookEvent, error)
	Refund(context.Context, string, int64) error
}

var ErrInvalidPaymentEvent = errors.New("invalid payment event")

func ValidateWebhookEvent(event WebhookEvent) error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Provider) == "" || strings.TrimSpace(event.ProviderEventID) == "" || strings.TrimSpace(event.OrderID) == "" || event.AmountCents < 0 || strings.TrimSpace(event.Currency) == "" || strings.TrimSpace(event.Status) == "" {
		return ErrInvalidPaymentEvent
	}
	return nil
}
