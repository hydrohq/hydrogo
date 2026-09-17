package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type PaymentIntent struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	OrderID     string `json:"order_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	ClientToken string `json:"client_token,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

type PaymentTransaction struct {
	OrderID           string `json:"order_id"`
	Provider          string `json:"provider"`
	ProviderPaymentID string `json:"provider_payment_id"`
	AmountCents       int64  `json:"amount_cents"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
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

// IdempotentPaymentProvider is an optional stronger contract for providers
// that persist and replay a payment intent for the supplied retry key.
// Providers that do not implement it remain at-least-once by design.
type IdempotentPaymentProvider interface {
	PaymentProvider
	CreatePaymentIntentWithKey(context.Context, Order, string) (PaymentIntent, error)
}

var ErrInvalidPaymentEvent = errors.New("invalid payment event")
var ErrPaymentMismatch = errors.New("payment does not match order")
var ErrRefundPending = errors.New("refund reconciliation is pending")

func ValidateWebhookEvent(event WebhookEvent) error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Provider) == "" || strings.TrimSpace(event.ProviderEventID) == "" || strings.TrimSpace(event.OrderID) == "" || strings.TrimSpace(event.PaymentID) == "" || event.AmountCents < 0 || strings.TrimSpace(event.Currency) == "" || strings.TrimSpace(event.Status) == "" {
		return ErrInvalidPaymentEvent
	}
	return nil
}

// PaymentWebhookStore is kept separate from the catalog store so a provider
// adapter cannot accidentally bypass the order/payment consistency boundary.
type PaymentWebhookStore interface {
	ClaimWebhook(context.Context, database.Executor, string, string) (bool, error)
	ApplyPaymentEvent(context.Context, database.Executor, WebhookEvent) error
}

type PaymentRefundStore interface {
	GetPaymentTransaction(context.Context, database.Executor, string) (PaymentTransaction, error)
	ListPendingRefunds(context.Context, database.Executor, int) ([]PaymentTransaction, error)
	ClaimPaymentRefund(context.Context, database.Executor, string, string) error
	MarkPaymentRefunded(context.Context, database.Executor, string, string) error
}

type PaymentService struct {
	Transactions database.Transactor
	Store        PaymentWebhookStore
	Providers    map[string]PaymentProvider
}

func (s PaymentService) HandleWebhook(ctx context.Context, providerName string, request *http.Request) error {
	provider, ok := s.Providers[providerName]
	if !ok || provider == nil {
		return fmt.Errorf("payment provider %q is not configured", providerName)
	}
	event, err := provider.VerifyWebhookSignature(request)
	if err != nil {
		return err
	}
	if err := ValidateWebhookEvent(event); err != nil {
		return err
	}
	if event.Provider != providerName {
		return ErrInvalidPaymentEvent
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		claimed, err := s.Store.ClaimWebhook(ctx, tx, event.Provider, event.ProviderEventID)
		if err != nil || !claimed {
			return err
		}
		return s.Store.ApplyPaymentEvent(ctx, tx, event)
	})
}

func (s PaymentService) Refund(ctx context.Context, db database.Executor, orderID, providerName, paymentID string, amountCents int64) error {
	provider, ok := s.Providers[providerName]
	if !ok || provider == nil {
		return fmt.Errorf("payment provider %q is not configured", providerName)
	}
	store, ok := s.Store.(PaymentRefundStore)
	if !ok {
		return errors.New("payment refund store is not configured")
	}
	transaction, err := store.GetPaymentTransaction(ctx, db, orderID)
	if err != nil {
		return err
	}
	if transaction.Provider != providerName || transaction.ProviderPaymentID != paymentID || transaction.Status != "paid" || transaction.AmountCents != amountCents {
		return ErrPaymentMismatch
	}
	// Claim before the external effect. If the provider outcome is ambiguous
	// (timeout, disconnect, or a database failure after success), retries stay
	// pending for reconciliation instead of issuing a duplicate refund.
	if err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		return store.ClaimPaymentRefund(ctx, tx, orderID, paymentID)
	}); err != nil {
		return err
	}
	if err := provider.Refund(ctx, paymentID, amountCents); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return store.MarkPaymentRefunded(ctx, tx, orderID, paymentID) })
}
