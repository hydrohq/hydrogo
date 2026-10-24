package transport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// maxListOffset bounds OFFSET paging so a crafted request cannot force a
// scan past the whole table on every call.
const maxListOffset = 10000

type setStockRequest struct {
	VariantID string `json:"variant_id"`
	Available int    `json:"available"`
}

func (h Handler) setStock(w http.ResponseWriter, r *http.Request) {
	var input setStockRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.VariantID) == "" || input.Available < 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "variant_id and a non-negative available are required", nil)
		return
	}
	stock, err := h.Service.SetStock(r.Context(), input.VariantID, input.Available)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusOK, stock, nil)
}

type createDiscountCodeRequest struct {
	Code             string `json:"code"`
	Kind             string `json:"kind"` // "percent" or "fixed"
	Value            int64  `json:"value"`
	MinSubtotalCents int64  `json:"min_subtotal_cents"`
	MaxRedemptions   int    `json:"max_redemptions"`
	Active           *bool  `json:"active"`
	StartsAt         string `json:"starts_at"` // RFC3339; empty = no restriction
	EndsAt           string `json:"ends_at"`
}

func (h Handler) createDiscountCode(w http.ResponseWriter, r *http.Request) {
	var input createDiscountCodeRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	code := strings.TrimSpace(input.Code)
	kind := commerce.DiscountKind(input.Kind)
	if code == "" || (kind != commerce.DiscountPercent && kind != commerce.DiscountFixed) || input.Value <= 0 || input.MaxRedemptions < 0 || input.MinSubtotalCents < 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "code, a kind of percent or fixed, and a positive value are required", nil)
		return
	}
	if kind == commerce.DiscountPercent && input.Value > 100 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "a percent discount cannot exceed 100", nil)
		return
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	var startsAt, endsAt time.Time
	var err error
	if input.StartsAt != "" {
		if startsAt, err = time.Parse(time.RFC3339, input.StartsAt); err != nil {
			api.WriteError(w, r, http.StatusUnprocessableEntity, "starts_at must be RFC3339", nil)
			return
		}
	}
	if input.EndsAt != "" {
		if endsAt, err = time.Parse(time.RFC3339, input.EndsAt); err != nil {
			api.WriteError(w, r, http.StatusUnprocessableEntity, "ends_at must be RFC3339", nil)
			return
		}
	}
	token, err := randomToken()
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	record := commerce.DiscountCode{
		ID: "disc_" + token, Code: code, Kind: kind, Value: input.Value, MinSubtotalCents: input.MinSubtotalCents,
		MaxRedemptions: input.MaxRedemptions, Active: active, StartsAt: startsAt, EndsAt: endsAt,
	}
	if err := h.Service.CreateDiscountCode(r.Context(), record); err != nil {
		writeServiceError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusCreated, record, nil)
}

// createShipment hands a paid order to the configured carrier. Operator
// action only.
func (h Handler) createShipment(w http.ResponseWriter, r *http.Request) {
	if h.ShippingProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no shipping provider is configured", nil)
		return
	}
	order, err := h.Service.CreateShipment(r.Context(), h.ShippingProvider, r.PathValue("id"))
	if err != nil {
		if errorsIsAny(err, commerce.ErrOrderNotFound, commerce.ErrOrderNotShippable) {
			writeServiceError(w, r, err)
			return
		}
		api.WriteError(w, r, http.StatusBadGateway, "the carrier did not accept the shipment", nil)
		return
	}
	api.WriteData(w, http.StatusOK, order, nil)
}

func intQueryParam(raw string, fallback, min, max int) int {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

// listOrdersAdmin is the order screen's data source. Seeing every order is
// operator-only; there is no public equivalent.
func (h Handler) listOrdersAdmin(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := intQueryParam(query.Get("limit"), 25, 1, 100)
	offset := intQueryParam(query.Get("offset"), 0, 0, maxListOffset)
	paymentStatus := commerce.PaymentStatus(strings.TrimSpace(query.Get("payment_status")))
	fulfillmentStatus := commerce.FulfillmentStatus(strings.TrimSpace(query.Get("fulfillment_status")))

	orders, err := h.Service.ListOrders(r.Context(), paymentStatus, fulfillmentStatus, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}
	if orders == nil {
		orders = []commerce.Order{}
	}
	api.WriteList(w, orders, api.PaginationMeta{Limit: limit, Offset: offset, HasMore: hasMore})
}

func errorsIsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
