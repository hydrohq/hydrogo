package commerce

import "context"

// PaymentProvider lets checkout start an off-site payment without Service or
// its transport knowing which provider is in use. Service never imports a
// provider package, keeping checkout orchestration provider-agnostic.
type PaymentProvider interface {
	StartPayment(ctx context.Context, order Order, items []OrderItem) (PaymentStart, error)
}

// AdvancedPaymentProvider allows optional full authorization/capture/refund lifecycle.
type AdvancedPaymentProvider interface {
	PaymentProvider
	Authorize(ctx context.Context, order Order, amountCents int64) error
	Capture(ctx context.Context, order Order, amountCents int64) error
	Refund(ctx context.Context, order Order, amountCents int64) error
}

// PaymentStart is what a provider returns once it has accepted an order for
// payment. RedirectURL sends the shopper to a hosted payment page; Token is
// an opaque value a frontend can use to render an embedded form instead.
// A provider sets whichever of the two its integration uses.
//
// Method names the payment method that accepted the order, and Instructions
// carries what a shopper must do next when there is nothing to redirect to
// (for example the bank details of a manual transfer).
type PaymentStart struct {
	Token        string            `json:"token,omitempty"`
	RedirectURL  string            `json:"redirect_url,omitempty"`
	Method       string            `json:"method,omitempty"`
	Instructions map[string]string `json:"instructions,omitempty"`
	// ProviderRef carries an attempt-specific gateway reference (such as PayTR's
	// per-attempt merchant_oid) back to Service.StartPayment for persistence.
	ProviderRef string `json:"-"`
}

type paymentMethodKey struct{}

// WithPaymentMethod attaches the payment method the shopper chose to ctx, for
// a provider that multiplexes several methods. Like the client IP it is
// request-scoped metadata, not an Order column.
func WithPaymentMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, paymentMethodKey{}, method)
}

// PaymentMethod returns the method WithPaymentMethod attached, or "".
func PaymentMethod(ctx context.Context) string {
	method, _ := ctx.Value(paymentMethodKey{}).(string)
	return method
}

type clientIPKey struct{}

// WithClientIP attaches the shopper's IP to ctx for a provider that needs it.
// It is request-scoped transport metadata, not order data, so it travels on
// ctx rather than as a field on Order -- the commerce schema does not need a
// column that only one provider cares about.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

// ClientIP returns the IP WithClientIP attached, or "" if none was.
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

// PendingPaymentChecker is an optional capability of a payment method: look at orders that are still
// unpaid and settle the ones the provider says were paid. It covers a shopper who paid and closed the
// browser, when neither the return redirect nor a notification ever reached us.
type PendingPaymentChecker interface {
	CheckPending(ctx context.Context, orders []Order)
}
