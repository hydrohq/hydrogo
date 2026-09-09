package mailer

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

// RetrySender retries transport failures with bounded exponential backoff.
// Provider-specific permanent/retryable classification can be supplied through
// ShouldRetry; context cancellation is never retried.
type RetrySender struct {
	Sender      Sender
	Attempts    int
	BaseDelay   time.Duration
	ShouldRetry func(error) bool
}

func (r RetrySender) Send(ctx context.Context, message Message) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if guards.IsNil(r.Sender) {
		return ErrSenderRequired
	}
	attempts := r.Attempts
	if attempts < 1 {
		attempts = 1
	}
	delay := r.BaseDelay
	if delay <= 0 {
		delay = 25 * time.Millisecond
	}
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = r.Sender.Send(ctx, message)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil || (r.ShouldRetry != nil && !r.ShouldRetry(err)) || attempt == attempts-1 {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < time.Second {
			delay *= 2
		}
	}
	return err
}
