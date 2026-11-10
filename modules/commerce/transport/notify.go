package transport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/notify"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// OrderReader provides access to orders and their line items for notification formatting.
type OrderReader interface {
	OrderWithItems(ctx context.Context, orderID string) (commerce.Order, []commerce.OrderItem, error)
}

// NotifyOrderPaid sends an operator notification when an order payment is successfully confirmed.
// It is best-effort: a failure or timeout is only logged and never disrupts the payment flow.
func NotifyOrderPaid(ctx context.Context, notifier notify.Sender, reader OrderReader, orderID, provider string, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	pingCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), notifyTimeout)
	defer cancel()

	var order commerce.Order
	var items []commerce.OrderItem
	if reader != nil {
		var err error
		order, items, err = reader.OrderWithItems(pingCtx, orderID)
		if err != nil && logger != nil {
			logger.WarnContext(pingCtx, "could not load order for paid notification", "order_id", orderID, "error", err)
		}
	}

	total := FormatMoney(order.TotalCents, order.Currency)
	itemSummary := fmt.Sprintf("%d kalem", len(items))
	if len(items) > 0 {
		var titles []string
		for _, item := range items {
			if item.Title != "" {
				titles = append(titles, fmt.Sprintf("%s x%d", item.Title, item.Quantity))
			}
		}
		if len(titles) > 0 {
			itemSummary = fmt.Sprintf("%d kalem (%s)", len(items), strings.Join(titles, ", "))
		}
	}

	body := fmt.Sprintf("%s — %s (%s, %s)", orderID, total, provider, itemSummary)
	if err := notifier.Send(pingCtx, notify.Message{
		Title: "Yeni sipariş (ödendi)",
		Body:  body,
	}); err != nil && logger != nil {
		logger.WarnContext(pingCtx, "could not send paid order notification", "order_id", orderID, "error", err)
	}
}
