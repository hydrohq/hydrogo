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
	ShipmentID     string `json:"shipment_id,omitempty"`
	LabelURL       string `json:"label_url,omitempty"`
}

// ParcelSpec carries optional package dimensions and weight when creating or
// quoting a carrier shipment. Empty fields fall back to provider defaults.
type ParcelSpec struct {
	LengthCm string `json:"length_cm,omitempty"`
	WidthCm  string `json:"width_cm,omitempty"`
	HeightCm string `json:"height_cm,omitempty"`
	WeightKg string `json:"weight_kg,omitempty"`
}

// ShipmentInput is what an operator supplies when handing an order over:
// manual tracking fields for carriers booked outside, or an offer/shipment
// selection plus parcel overrides for aggregator carriers.
type ShipmentInput struct {
	Carrier        string
	TrackingNumber string
	TrackingURL    string
	OfferID        string
	ShipmentID     string
	Parcel         ParcelSpec
}

// ShipmentOffer is one carrier quote returned by an aggregator such as Geliver.
type ShipmentOffer struct {
	ID                string `json:"id"`
	Carrier           string `json:"carrier"`
	CarrierName       string `json:"carrier_name"`
	ServiceCode       string `json:"service_code,omitempty"`
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	EstimatedDelivery string `json:"estimated_delivery,omitempty"`
}

// ShipmentOfferDraft holds the provider shipment identifier and available quotes.
type ShipmentOfferDraft struct {
	ShipmentID string          `json:"shipment_id"`
	CheapestID string          `json:"cheapest_id,omitempty"`
	FastestID  string          `json:"fastest_id,omitempty"`
	Offers     []ShipmentOffer `json:"offers"`
}

// ShipmentOfferer is optionally implemented by shipping providers that quote
// multiple carriers before purchasing a label.
type ShipmentOfferer interface {
	ListShipmentOffers(ctx context.Context, order Order, items []OrderItem) (ShipmentOfferDraft, error)
}

// ShipmentOrderCanceller is optionally implemented by shipping providers that can
// cancel an already created shipment for an order.
type ShipmentOrderCanceller interface {
	CancelOrderShipment(ctx context.Context, order Order) error
}

// CarrierInfo describes one enabled carrier and its optional capabilities.
type CarrierInfo struct {
	Key            string `json:"key"`
	SupportsOffers bool   `json:"supports_offers"`
	SupportsCancel bool   `json:"supports_cancel"`
}

// CarrierInspector is optionally implemented by composite shipping routers so
// the admin UI knows which carriers and actions are enabled.
type CarrierInspector interface {
	DefaultCarrier() string
	CarrierInfos() []CarrierInfo
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
