package transport

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/hydrohq/hydrogo/internal/api"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

type shipmentActionRequest struct {
	Carrier        string              `json:"carrier"`
	TrackingNumber string              `json:"tracking_number"`
	TrackingURL    string              `json:"tracking_url"`
	OfferID        string              `json:"offer_id"`
	ShipmentID     string              `json:"shipment_id"`
	Parcel         commerce.ParcelSpec `json:"parcel"`
}

func toShipmentInput(req shipmentActionRequest) commerce.ShipmentInput {
	return commerce.ShipmentInput{
		Carrier:        strings.TrimSpace(req.Carrier),
		TrackingNumber: strings.TrimSpace(req.TrackingNumber),
		TrackingURL:    strings.TrimSpace(req.TrackingURL),
		OfferID:        strings.TrimSpace(req.OfferID),
		ShipmentID:     strings.TrimSpace(req.ShipmentID),
		Parcel: commerce.ParcelSpec{
			LengthCm: strings.TrimSpace(req.Parcel.LengthCm),
			WidthCm:  strings.TrimSpace(req.Parcel.WidthCm),
			HeightCm: strings.TrimSpace(req.Parcel.HeightCm),
			WeightKg: strings.TrimSpace(req.Parcel.WeightKg),
		},
	}
}

func (h Handler) getShippingCarriers(w http.ResponseWriter, r *http.Request) {
	if h.ShippingProvider == nil {
		api.WriteData(w, http.StatusOK, map[string]any{
			"configured":      false,
			"default_carrier": "",
			"carriers":        []commerce.CarrierInfo{},
		}, nil)
		return
	}
	if inspector, ok := h.ShippingProvider.(commerce.CarrierInspector); ok {
		api.WriteData(w, http.StatusOK, map[string]any{
			"configured":      true,
			"default_carrier": inspector.DefaultCarrier(),
			"carriers":        inspector.CarrierInfos(),
		}, nil)
		return
	}
	_, supportsOffers := h.ShippingProvider.(commerce.ShipmentOfferer)
	_, supportsCancel := h.ShippingProvider.(commerce.ShipmentOrderCanceller)
	name := strings.TrimSpace(h.ShippingProvider.Name())
	api.WriteData(w, http.StatusOK, map[string]any{
		"configured":      true,
		"default_carrier": name,
		"carriers": []commerce.CarrierInfo{{
			Key:            name,
			SupportsOffers: supportsOffers,
			SupportsCancel: supportsCancel,
		}},
	}, nil)
}

func (h Handler) listShipmentOffers(w http.ResponseWriter, r *http.Request) {
	if h.ShippingProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no shipping provider is configured", nil)
		return
	}
	var input shipmentActionRequest
	ctx := r.Context()
	if r.Body != nil && r.ContentLength != 0 {
		if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
			return
		}
		ctx = commerce.WithShipmentInput(ctx, toShipmentInput(input))
	}
	draft, err := h.Service.ListShipmentOffers(ctx, h.ShippingProvider, r.PathValue("id"))
	if err != nil {
		if errorsIsAny(err, commerce.ErrOrderNotFound, commerce.ErrOrderNotShippable, commerce.ErrShippingActionUnsupported) {
			writeServiceError(w, r, err)
			return
		}
		slog.WarnContext(r.Context(), "shipping offer lookup failed", "order_id", r.PathValue("id"), "error", err)
		api.WriteError(w, r, http.StatusBadGateway, "the carrier could not provide shipping offers", nil)
		return
	}
	api.WriteData(w, http.StatusOK, draft, nil)
}

// createShipment hands a paid order to the configured carrier. Operator
// action only.
func (h Handler) createShipment(w http.ResponseWriter, r *http.Request) {
	if h.ShippingProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no shipping provider is configured", nil)
		return
	}
	var input shipmentActionRequest
	ctx := r.Context()
	if r.Body != nil && r.ContentLength != 0 {
		if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
			return
		}
		ctx = commerce.WithShipmentInput(ctx, toShipmentInput(input))
	}
	order, err := h.Service.CreateShipment(ctx, h.ShippingProvider, r.PathValue("id"))
	if err != nil {
		if errorsIsAny(err, commerce.ErrOrderNotFound, commerce.ErrOrderNotShippable) {
			writeServiceError(w, r, err)
			return
		}
		slog.WarnContext(r.Context(), "shipping create shipment failed", "order_id", r.PathValue("id"), "error", err)
		api.WriteError(w, r, http.StatusBadGateway, "the carrier did not accept the shipment", nil)
		return
	}
	api.WriteData(w, http.StatusOK, order, nil)
}

func (h Handler) cancelShipment(w http.ResponseWriter, r *http.Request) {
	if h.ShippingProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no shipping provider is configured", nil)
		return
	}
	order, err := h.Service.CancelShipment(r.Context(), h.ShippingProvider, r.PathValue("id"))
	if err != nil {
		if errorsIsAny(err, commerce.ErrOrderNotFound, commerce.ErrOrderNotCancellableShipment, commerce.ErrShippingActionUnsupported) {
			writeServiceError(w, r, err)
			return
		}
		slog.WarnContext(r.Context(), "shipping cancel shipment failed", "order_id", r.PathValue("id"), "error", err)
		api.WriteError(w, r, http.StatusBadGateway, "the carrier did not accept the shipment cancellation", nil)
		return
	}
	api.WriteData(w, http.StatusOK, order, nil)
}
