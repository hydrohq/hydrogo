package commerce_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/notify"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

type mockReminderNotifier struct {
	mu       sync.Mutex
	messages []notify.Message
	err      error
}

func (m *mockReminderNotifier) Send(_ context.Context, msg notify.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, msg)
	return nil
}

func TestRefundReminderWorker(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	notifier := &mockReminderNotifier{}
	worker := commerce.RefundReminderWorker{
		Service:  svc,
		Notifier: notifier,
		Interval: 6 * time.Hour,
	}

	// 1. When no unresolved refunds exist, count is 0, no notification sent.
	count, err := worker.RemindDue(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}
	if len(notifier.messages) != 0 {
		t.Fatalf("expected 0 messages sent, got %d", len(notifier.messages))
	}

	// 2. Add an unresolved refund requirement.
	err = svc.Repository.RecordRefundRequired(ctx, db, commerce.RefundRequired{
		ID:          "rrq-test-reminder",
		OrderID:     "order-remind-1",
		Provider:    "gateway",
		AmountCents: 5000,
		Currency:    "TRY",
		Reason:      "callback after expired order",
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to record refund required: %v", err)
	}

	// 3. RemindDue should find count 1 and send a message.
	count, err = worker.RemindDue(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(notifier.messages))
	}
	if !strings.Contains(notifier.messages[0].Body, "1") {
		t.Fatalf("expected message body to contain count 1, got: %s", notifier.messages[0].Body)
	}

	// 4. Resolve the refund requirement.
	ok, err := svc.ResolveRefundRequired(ctx, "rrq-test-reminder", "resolved by test")
	if err != nil || !ok {
		t.Fatalf("failed to resolve refund required: %v", err)
	}

	// 5. RemindDue should now return count 0 and not send a new message.
	count, err = worker.RemindDue(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("expected still 1 message total, got %d", len(notifier.messages))
	}
}
