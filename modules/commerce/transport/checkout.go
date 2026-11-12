package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/api"
	"github.com/hydrohq/hydrogo/internal/mailer"
	"github.com/hydrohq/hydrogo/internal/security"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

type checkoutRequest struct {
	GuestName       string `json:"guest_name"`
	GuestEmail      string `json:"guest_email"`
	GuestPhone      string `json:"guest_phone"`
	GuestAddress    string `json:"guest_address"`
	GuestCity       string `json:"guest_city"`
	GuestDistrict   string `json:"guest_district"`
	GuestCountry    string `json:"guest_country"`
	GuestPostalCode string `json:"guest_postal_code"`
	IdempotencyKey  string `json:"idempotency_key"`
	DiscountCode    string `json:"discount_code"`
	// PaymentMethod picks one of the methods a multiplexing PaymentProvider
	// offers (optional; the provider's default applies when empty).
	PaymentMethod string `json:"payment_method"`
	// ItemTitles/ItemSKUs (keyed by variant_id) are receipt text supplied by
	// the storefront. They never affect price.
	ItemTitles map[string]string `json:"item_titles"`
	ItemSKUs   map[string]string `json:"item_skus"`
}

type checkoutResponse struct {
	commerce.Order
	Payment *commerce.PaymentStart `json:"payment,omitempty"`
}

type orderItemView struct {
	VariantID      string `json:"variant_id"`
	SKU            string `json:"sku"`
	Title          string `json:"title"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	TotalCents     int64  `json:"total_cents"`
}

type orderView struct {
	commerce.Order
	Items []orderItemView `json:"items"`
}

func (h Handler) checkout(w http.ResponseWriter, r *http.Request) {
	shopper, err := h.shopper(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	ip := h.clientIP(r)
	if rateLimited(w, h.CheckoutLimiter, "checkout:"+ip) {
		return
	}
	reservation := ""
	keepReservation := false
	if h.PendingTracker != nil {
		maxPending := h.MaxPendingOrdersPerIP
		if maxPending <= 0 {
			maxPending = 25
		}
		if h.PendingTracker.CountPending(r.Context(), ip, h.Service, time.Now().UTC(), 30*time.Minute) >= maxPending {
			security.WriteRateLimitedResponse(w, "too many open pending checkouts", 30*time.Minute)
			return
		}
		var reserved bool
		reservation, reserved = h.PendingTracker.TryReserve(ip, maxPending)
		if !reserved {
			security.WriteRateLimitedResponse(w, "too many open pending checkouts", 30*time.Minute)
			return
		}
		defer func() {
			if !keepReservation {
				h.PendingTracker.Release(ip, reservation)
			}
		}()
	}
	var input checkoutRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "idempotency_key is required", nil)
		return
	}
	if shopper.MemberID == "" && strings.TrimSpace(input.GuestEmail) == "" {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "guest_email is required for a guest checkout", nil)
		return
	}
	cart, _, err := h.Service.CartFor(r.Context(), shopper)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	order, created, err := h.Service.PlaceOrder(r.Context(), commerce.CheckoutRequest{
		CartID: cart.ID, MemberID: shopper.MemberID, Currency: cart.Currency,
		GuestName: input.GuestName, GuestEmail: input.GuestEmail, GuestPhone: input.GuestPhone,
		GuestAddress: input.GuestAddress, GuestCity: input.GuestCity, GuestDistrict: input.GuestDistrict, GuestCountry: input.GuestCountry, GuestPostalCode: input.GuestPostalCode,
		IdempotencyKey: input.IdempotencyKey, DiscountCode: input.DiscountCode, PaymentMethod: strings.TrimSpace(input.PaymentMethod),
		ItemTitles: input.ItemTitles, ItemSKUs: input.ItemSKUs, MaxLineQuantity: h.MaxLineQuantity,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response := checkoutResponse{Order: order}
	if order.PaymentStatus == commerce.PaymentPending && h.PaymentProvider != nil {
		// Also on an idempotent retry: the shopper still needs somewhere to pay.
		paymentCtx := commerce.WithPaymentMethod(commerce.WithClientIP(r.Context(), h.clientIP(r)), strings.TrimSpace(input.PaymentMethod))
		if start, startErr := h.Service.StartPayment(paymentCtx, h.PaymentProvider, order.ID); startErr == nil {
			response.Payment = &start
		} else {
			slog.WarnContext(r.Context(), "payment could not be started", "order_id", order.ID, "error", startErr)
		}
	}
	if reservation != "" && created && order.PaymentStatus == commerce.PaymentPending {
		h.PendingTracker.Confirm(ip, reservation, order.ID)
		keepReservation = true
	}
	if created {
		if _, items, itemsErr := h.Service.OrderWithItems(r.Context(), order.ID); itemsErr == nil {
			h.notifyNewOrder(r.Context(), order, items)
		}
	}
	api.WriteData(w, http.StatusCreated, response, nil)
}

// notifyNewOrder sends the customer's receipt, once per order.
// Best-effort and bounded: a slow mail server costs at most notifyTimeout,
// and the call outlives a shopper who closes the tab.
func (h Handler) notifyNewOrder(ctx context.Context, order commerce.Order, items []commerce.OrderItem) {
	ctx = context.WithoutCancel(ctx)
	total := FormatMoney(order.TotalCents, order.Currency)
	if h.Mailer != nil && order.GuestEmail != "" {
		mailItems := make([]mailer.OrderConfirmationItem, 0, len(items))
		for _, item := range items {
			mailItems = append(mailItems, mailer.OrderConfirmationItem{Title: item.Title, Quantity: item.Quantity, Total: FormatMoney(item.TotalCents, order.Currency)})
		}
		mailCtx, cancel := context.WithTimeout(ctx, notifyTimeout)
		_ = h.Mailer.SendOrderConfirmation(mailCtx, order.GuestEmail, order.ID, total, mailItems)
		cancel()
	}
}

// FormatMoney formats an amount in cents and currency as "XX.YY CUR".
func FormatMoney(cents int64, currency string) string {
	if currency == "" {
		return fmt.Sprintf("%.2f", float64(cents)/100)
	}
	return fmt.Sprintf("%.2f %s", float64(cents)/100, currency)
}

// getOrder is an order confirmation/status read. A guest has no account, so
// knowing the order's unguessable id is the access check; admins reach the
// same handler through the protected route.
func (h Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	order, items, err := h.Service.OrderWithItems(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	views := make([]orderItemView, 0, len(items))
	for _, item := range items {
		views = append(views, orderItemView{VariantID: item.VariantID, SKU: item.SKU, Title: item.Title, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, TotalCents: item.TotalCents})
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, orderView{Order: order, Items: views}, nil)
}

// startOrderPayment restarts a payment for a pending order after the first
// attempt failed. Same id-knowledge trust model as getOrder.
func (h Handler) startOrderPayment(w http.ResponseWriter, r *http.Request) {
	if h.PaymentProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no payment provider is configured", nil)
		return
	}
	var input struct {
		PaymentMethod string `json:"payment_method"`
	}
	if r.ContentLength != 0 && !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	paymentCtx := commerce.WithPaymentMethod(commerce.WithClientIP(r.Context(), h.clientIP(r)), strings.TrimSpace(input.PaymentMethod))
	start, err := h.Service.StartPayment(paymentCtx, h.PaymentProvider, r.PathValue("id"))
	if err != nil {
		switch {
		case errorsIsAny(err, commerce.ErrOrderNotFound, commerce.ErrOrderNotPayable):
			writeServiceError(w, r, err)
		default:
			api.WriteError(w, r, http.StatusBadGateway, "could not start a payment with the configured provider", nil)
		}
		return
	}
	api.WriteData(w, http.StatusOK, start, nil)
}
