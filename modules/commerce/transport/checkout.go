package transport

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/notify"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
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
	if rateLimited(w, h.CheckoutLimiter, "checkout:"+h.clientIP(r)) {
		return
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
		IdempotencyKey: input.IdempotencyKey, DiscountCode: input.DiscountCode,
		ItemTitles: input.ItemTitles, ItemSKUs: input.ItemSKUs,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response := checkoutResponse{Order: order}
	if order.PaymentStatus == commerce.PaymentPending && h.PaymentProvider != nil {
		// Also on an idempotent retry: the shopper still needs somewhere to pay.
		paymentCtx := commerce.WithClientIP(r.Context(), h.clientIP(r))
		if start, startErr := h.Service.StartPayment(paymentCtx, h.PaymentProvider, order.ID); startErr == nil {
			response.Payment = &start
		}
	}
	if created {
		if _, items, itemsErr := h.Service.OrderWithItems(r.Context(), order.ID); itemsErr == nil {
			h.notifyNewOrder(r.Context(), order, items)
		}
	}
	api.WriteData(w, http.StatusCreated, response, nil)
}

// notifyNewOrder sends the customer's receipt and the operator's chat alert,
// once per order. Best-effort and bounded: a slow mail server or chat API
// costs at most notifyTimeout each, and the calls outlive a shopper who
// closes the tab.
func (h Handler) notifyNewOrder(ctx context.Context, order commerce.Order, items []commerce.OrderItem) {
	ctx = context.WithoutCancel(ctx)
	total := formatMoney(order.TotalCents, order.Currency)
	if h.Mailer != nil && order.GuestEmail != "" {
		mailItems := make([]mailer.OrderConfirmationItem, 0, len(items))
		for _, item := range items {
			mailItems = append(mailItems, mailer.OrderConfirmationItem{Title: item.Title, Quantity: item.Quantity, Total: formatMoney(item.TotalCents, order.Currency)})
		}
		mailCtx, cancel := context.WithTimeout(ctx, notifyTimeout)
		_ = h.Mailer.SendOrderConfirmation(mailCtx, order.GuestEmail, order.ID, total, mailItems)
		cancel()
	}
	if h.Notifier != nil {
		pingCtx, cancel := context.WithTimeout(ctx, notifyTimeout)
		_ = h.Notifier.Send(pingCtx, notify.Message{Title: "Yeni sipariş", Body: fmt.Sprintf("%s — %s", order.ID, total)})
		cancel()
	}
}

func formatMoney(cents int64, currency string) string {
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
	paymentCtx := commerce.WithClientIP(r.Context(), h.clientIP(r))
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
