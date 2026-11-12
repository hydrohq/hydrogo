package transport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hydrohq/hydrogo/internal/notify"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

// OrderReader provides access to orders and their line items for notification formatting.
type OrderReader interface {
	OrderWithItems(ctx context.Context, orderID string) (commerce.Order, []commerce.OrderItem, error)
}

var notificationWG sync.WaitGroup

// WaitNotifications blocks until all in-flight asynchronous notifications complete.
// Intended for tests and graceful shutdown.
func WaitNotifications() {
	notificationWG.Wait()
}

// DispatchNotificationAsync sends an arbitrary notification in a background goroutine with exponential retry.
func DispatchNotificationAsync(notifier notify.Sender, msg notify.Message, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	notificationWG.Add(1)
	go func() {
		defer notificationWG.Done()
		sendWithRetry(notifier, msg, logger)
	}()
}

func sendWithRetry(notifier notify.Sender, msg notify.Message, logger *slog.Logger) {
	const maxAttempts = 3
	backoff := 50 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sendCtx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
		err := notifier.Send(sendCtx, msg)
		cancel()
		if err == nil {
			return
		}
		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		} else if logger != nil {
			logger.Warn("could not deliver notification after retries", "title", msg.Title, "error", err)
		}
	}
}

func formatPaidOrderMessage(ctx context.Context, reader OrderReader, orderID, provider string, logger *slog.Logger) notify.Message {
	var order commerce.Order
	var items []commerce.OrderItem
	if reader != nil {
		readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), notifyTimeout)
		var err error
		order, items, err = reader.OrderWithItems(readCtx, orderID)
		cancel()
		if err != nil && logger != nil {
			logger.WarnContext(ctx, "could not load order for paid notification", "order_id", orderID, "error", err)
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
	return notify.Message{
		Title: "Yeni sipariş (ödendi)",
		Body:  body,
	}
}

// NotifyOrderPaid sends an operator notification synchronously when an order payment is confirmed.
func NotifyOrderPaid(ctx context.Context, notifier notify.Sender, reader OrderReader, orderID, provider string, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	msg := formatPaidOrderMessage(ctx, reader, orderID, provider, logger)
	pingCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), notifyTimeout)
	defer cancel()
	if err := notifier.Send(pingCtx, msg); err != nil && logger != nil {
		logger.WarnContext(pingCtx, "could not send paid order notification", "order_id", orderID, "error", err)
	}
}

// NotifyOrderPaidAsync formats and dispatches the paid order notification in a background
// goroutine with exponential backoff retries, so the caller is never blocked.
func NotifyOrderPaidAsync(ctx context.Context, notifier notify.Sender, reader OrderReader, orderID, provider string, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	msg := formatPaidOrderMessage(ctx, reader, orderID, provider, logger)
	notificationWG.Add(1)
	go func() {
		defer notificationWG.Done()
		sendWithRetry(notifier, msg, logger)
	}()
}

// NotifyOperator sends an alert notification synchronously to the operator.
func NotifyOperator(ctx context.Context, notifier notify.Sender, title, orderID string, amountCents int64, currency, provider string, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	total := FormatMoney(amountCents, currency)
	body := fmt.Sprintf("Sipariş No: %s, Tutar: %s, Sağlayıcı: %s", orderID, total, provider)
	pingCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), notifyTimeout)
	defer cancel()
	if err := notifier.Send(pingCtx, notify.Message{
		Title: title,
		Body:  body,
	}); err != nil && logger != nil {
		logger.WarnContext(pingCtx, "could not send operator notification", "order_id", orderID, "error", err)
	}
}

// NotifyOperatorAsync sends an alert notification to the operator in a background goroutine
// with exponential backoff retries.
func NotifyOperatorAsync(ctx context.Context, notifier notify.Sender, title, orderID string, amountCents int64, currency, provider string, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	total := FormatMoney(amountCents, currency)
	body := fmt.Sprintf("Sipariş No: %s, Tutar: %s, Sağlayıcı: %s", orderID, total, provider)
	msg := notify.Message{Title: title, Body: body}
	notificationWG.Add(1)
	go func() {
		defer notificationWG.Done()
		sendWithRetry(notifier, msg, logger)
	}()
}
