package commerce

import "context"

// ShippingProvider lets an operator hand a paid order to a carrier without
// Service or its transport knowing which one is in use -- the same shape as
// PaymentProvider, for the same reason: provider implementations live in
// decoupled adapter packages or starters, and Service never imports a provider package.
type ShippingProvider interface {
	// Name identifies the carrier for storage on the order (e.g.
	// "fedex", "dhl", "ups") and for choosing which provider handles a
	// tracking-number lookup later.
	Name() string
	CreateShipment(ctx context.Context, order Order, items []OrderItem) (ShipmentResult, error)
}

// CarrierAdapter is an optional extended interface for carriers that support tracking and cancellation.
type CarrierAdapter interface {
	ShippingProvider
	TrackShipment(ctx context.Context, trackingNumber string) (ShipmentResult, error)
	CancelShipment(ctx context.Context, trackingNumber string) error
}

// ShipmentResult is what a carrier returns once it has accepted an order
// for delivery. TrackingURL is the carrier's own public tracking page when
// it has one; a carrier that only returns a number leaves it empty and the
// storefront falls back to showing the number alone.
type ShipmentResult struct {
	// Carrier names the carrier that took the order when the provider chooses
	// among several; empty means ShippingProvider.Name().
	Carrier        string `json:"carrier,omitempty"`
	TrackingNumber string `json:"tracking_number"`
	TrackingURL    string `json:"tracking_url,omitempty"`
}

// ShipmentInput is what an operator supplies when handing an order over:
// needed by carriers booked outside the system (the shipping module's manual
// carrier), where the tracking number comes from the operator, not an API.
type ShipmentInput struct {
	Carrier        string
	TrackingNumber string
	TrackingURL    string
}

type shipmentInputKey struct{}

// WithShipmentInput attaches the operator's input to ctx for CreateShipment.
func WithShipmentInput(ctx context.Context, in ShipmentInput) context.Context {
	return context.WithValue(ctx, shipmentInputKey{}, in)
}

// ShipmentInputFrom returns what WithShipmentInput attached (zero if none).
func ShipmentInputFrom(ctx context.Context) ShipmentInput {
	in, _ := ctx.Value(shipmentInputKey{}).(ShipmentInput)
	return in
}
