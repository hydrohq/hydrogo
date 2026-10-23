package mailer

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

// OrderConfirmationItem is a plain, caller-formatted line for the receipt
// -- this package has no notion of a commerce order, variant or currency,
// so the caller (modules/commerce/transport) formats quantities and money
// into strings before calling SendOrderConfirmation.
type OrderConfirmationItem struct {
	Title    string
	Quantity int
	Total    string
}

// SendOrderConfirmation is best-effort, like SendInvite/SendPasswordReset:
// the caller treats a returned error as log-worthy, not fatal -- the order
// itself already succeeded by the time this is called.
func (n NotificationSender) SendOrderConfirmation(ctx context.Context, email, orderID, total string, items []OrderConfirmationItem) error {
	if guards.IsNil(n.Sender) {
		return ErrSenderRequired
	}
	html := renderEmailTemplate("order_confirmation", struct {
		OrderID string
		Total   string
		Items   []OrderConfirmationItem
	}{OrderID: orderID, Total: total, Items: items})
	return n.Sender.Send(ctx, Message{
		To:       email,
		Subject:  "Siparişiniz alındı: " + orderID,
		Body:     fmt.Sprintf("Siparişiniz alındı. Sipariş numarası: %s. Toplam: %s.", orderID, total),
		HTMLBody: html,
	})
}
