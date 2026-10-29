package commerce

import "context"

// PaymentProvider lets checkout start an off-site payment without Service or
// its transport knowing which provider is in use. Service never imports a
// provider package, keeping checkout orchestration provider-agnostic.
type PaymentProvider interface {
	StartPayment(ctx context.Context, order Order, items []OrderItem) (PaymentStart, error)
}

// PaymentStart is what a provider returns once it has accepted an order for
// payment. RedirectURL sends the shopper to a hosted payment page; Token is
// an opaque value a frontend can use to render an embedded form instead.
// A provider sets whichever of the two its integration uses.
type PaymentStart struct {
	Token       string `json:"token,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
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
