package transport

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
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

// goNotification runs fn in a tracked background goroutine. net/http only
// recovers panics in the handler goroutine, so a panicking notifier adapter
// would otherwise take the whole process down after the response was sent.
func goNotification(logger *slog.Logger, fn func()) {
	notificationWG.Add(1)
	go func() {
		defer notificationWG.Done()
		defer func() {
			if value := recover(); value != nil {
				if logger == nil {
					logger = slog.Default()
				}
				logger.Error("commerce notification panicked", "panic", fmt.Sprint(value), "stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}

// DispatchNotificationAsync sends an arbitrary notification in a background goroutine with exponential retry.
func DispatchNotificationAsync(notifier notify.Sender, msg notify.Message, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	goNotification(logger, func() { sendWithRetry(context.Background(), notifier, msg, logger) })
}

func sendWithRetry(ctx context.Context, notifier notify.Sender, msg notify.Message, logger *slog.Logger) {
	baseCtx := context.WithoutCancel(ctx)
	const maxAttempts = 3
	backoff := 50 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sendCtx, cancel := context.WithTimeout(baseCtx, notifyTimeout)
		err := notifier.Send(sendCtx, msg)
		cancel()
		if err == nil {
			return
		}
		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		} else if logger != nil {
			logger.WarnContext(baseCtx, "could not deliver notification after retries", "title", msg.Title, "error", err)
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
	itemSummary := fmt.Sprintf("%d item(s)", len(items))
	if len(items) > 0 {
		var titles []string
		for _, item := range items {
			if item.Title != "" {
				titles = append(titles, fmt.Sprintf("%s x%d", item.Title, item.Quantity))
			}
		}
		if len(titles) > 0 {
			itemSummary = fmt.Sprintf("%d item(s) (%s)", len(items), strings.Join(titles, ", "))
		}
	}

	body := fmt.Sprintf("%s — %s (%s, %s)", orderID, total, provider, itemSummary)
	return notify.Message{
		Title: "New order (paid)",
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
	bgCtx := context.WithoutCancel(ctx)
	msg := formatPaidOrderMessage(bgCtx, reader, orderID, provider, logger)
	goNotification(logger, func() { sendWithRetry(bgCtx, notifier, msg, logger) })
}

// NotifyOperatorAsync sends an alert notification to the operator in a background goroutine
// with exponential backoff retries.
func NotifyOperatorAsync(ctx context.Context, notifier notify.Sender, title, orderID string, amountCents int64, currency, provider string, logger *slog.Logger) {
	if notifier == nil {
		return
	}
	bgCtx := context.WithoutCancel(ctx)
	total := FormatMoney(amountCents, currency)
	body := fmt.Sprintf("Order No: %s, Amount: %s, Provider: %s", orderID, total, provider)
	msg := notify.Message{Title: title, Body: body}
	goNotification(logger, func() { sendWithRetry(bgCtx, notifier, msg, logger) })
}
