package mailer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const (
	SpoolPending         = "pending"
	SpoolSending         = "sending"
	SpoolDelivered       = "delivered"
	SpoolFailedPermanent = "failed_permanent"
	MaxSpoolAttempts     = 6
	MaxSpoolErrorBytes   = 1024
)

var retryDelays = [...]time.Duration{5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour}

var ErrSpoolLeaseInvalid = errors.New("mail spool lease is no longer current")

type SpoolEntry struct {
	ID, TenantID, EventKey string
	Message                Message
	Status                 string
	Attempts               int
	NextAttemptAt          time.Time
	CreatedAt, UpdatedAt   time.Time
	DeliveredAt            time.Time
	LastError              string
}

// SpoolStore provides durable, fenced delivery operations. Enqueue is called
// with the submission transaction so accepted leads and notification intent
// commit together.
type SpoolStore interface {
	Claim(context.Context, string, time.Time, time.Duration) (SpoolEntry, string, bool, error)
	Delivered(context.Context, string, string, string, time.Time) error
	Retry(context.Context, string, string, string, time.Time, string) error
	FailPermanent(context.Context, string, string, string, string) error
}

type SpoolWorker struct {
	Store        SpoolStore
	Sender       Sender
	WorkerID     string
	Lease        time.Duration
	Now          func() time.Time
	TenantScopes []tenant.Scope
	OnError      func(error)
	OnSchedule   func(context.Context, time.Time) error
}

func RetryDelay(failedAttempts int) time.Duration {
	if failedAttempts < 1 {
		return 0
	}
	if failedAttempts > len(retryDelays) {
		return retryDelays[len(retryDelays)-1]
	}
	return retryDelays[failedAttempts-1]
}

func (w SpoolWorker) RunOnce(ctx context.Context) (bool, error) {
	if w.Store == nil || w.Sender == nil || strings.TrimSpace(w.WorkerID) == "" {
		return false, errors.New("mail spool worker is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC()
	if w.Now != nil {
		now = w.Now().UTC()
	}
	lease := w.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	scopes := w.TenantScopes
	if len(scopes) == 0 {
		return w.runOnceScope(ctx, now, lease)
	}
	for _, scope := range scopes {
		worked, err := w.runOnceScope(tenant.WithScope(ctx, scope), now, lease)
		if err != nil || worked {
			return worked, err
		}
	}
	return false, nil
}

func (w SpoolWorker) runOnceScope(ctx context.Context, now time.Time, lease time.Duration) (bool, error) {
	entry, token, ok, err := w.Store.Claim(ctx, w.WorkerID, now, lease)
	if err != nil || !ok {
		return ok, err
	}
	deliveryErr := w.Sender.Send(ctx, entry.Message)
	if deliveryErr == nil {
		return true, w.Store.Delivered(ctx, entry.ID, w.WorkerID, token, now)
	}
	if ctx.Err() != nil {
		// Leave the lease to expire. A network timeout at shutdown has an
		// ambiguous remote result and must not be replayed immediately.
		return true, nil
	}
	if PermanentDeliveryError(deliveryErr) || entry.Attempts >= MaxSpoolAttempts {
		return true, w.Store.FailPermanent(ctx, entry.ID, w.WorkerID, token, boundedSpoolError(deliveryErr))
	}
	return true, w.Store.Retry(ctx, entry.ID, w.WorkerID, token, now.Add(RetryDelay(entry.Attempts)), boundedSpoolError(deliveryErr))
}

func (w SpoolWorker) Run(ctx context.Context, idle time.Duration) {
	if idle <= 0 {
		idle = 500 * time.Millisecond
	}
	nextSchedule := time.Time{}
	for {
		if ctx.Err() != nil {
			return
		}
		now := time.Now().UTC()
		if w.Now != nil {
			now = w.Now().UTC()
		}
		if w.OnSchedule != nil && (nextSchedule.IsZero() || !now.Before(nextSchedule)) {
			scheduleCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := w.OnSchedule(scheduleCtx, now)
			cancel()
			if err != nil && w.OnError != nil && ctx.Err() == nil {
				w.OnError(err)
			}
			nextSchedule = now.Add(time.Minute)
		}
		worked, err := w.RunOnce(ctx)
		if err != nil && w.OnError != nil && ctx.Err() == nil {
			w.OnError(err)
		}
		if err == nil && worked {
			continue
		}
		timer := time.NewTimer(idle)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func PermanentDeliveryError(err error) bool {
	if errors.Is(err, ErrInvalidMessage) {
		return true
	}
	var response *textproto.Error
	if errors.As(err, &response) {
		return response.Code >= 500 && response.Code <= 599
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return false
	}
	return false
}

func boundedSpoolError(err error) string {
	if err == nil {
		return ""
	}
	return BoundSpoolError(fmt.Sprint(err))
}

func BoundSpoolError(reason string) string {
	value := strings.TrimSpace(reason)
	if len(value) > MaxSpoolErrorBytes {
		value = value[:MaxSpoolErrorBytes]
	}
	return value
}
