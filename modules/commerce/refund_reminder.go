package commerce

import (
	"context"
	"fmt"
	"time"

	"github.com/hydrohq/hydrogo/internal/notify"
)

// RefundReminderWorker checks periodically for unresolved refund_required items
// and sends an alert to the operator via notifier if any exist.
type RefundReminderWorker struct {
	Service  Service
	Notifier notify.Sender
	Interval time.Duration
}

func (w RefundReminderWorker) interval() time.Duration {
	if w.Interval <= 0 {
		return 6 * time.Hour
	}
	return w.Interval
}

// RemindDue checks for unresolved refund_required records. If count > 0, it sends
// a reminder message to the operator via the configured notifier.
func (w RefundReminderWorker) RemindDue(ctx context.Context) (int, error) {
	count, err := w.Service.CountUnresolvedRefundRequired(ctx)
	if err != nil {
		return 0, err
	}
	if count > 0 && w.Notifier != nil {
		msg := notify.Message{
			Title: "Pending Refunds",
			Body:  fmt.Sprintf("%d refund request(s) are waiting to be resolved.", count),
		}
		if err := w.Notifier.Send(ctx, msg); err != nil {
			return count, err
		}
	}
	return count, nil
}
