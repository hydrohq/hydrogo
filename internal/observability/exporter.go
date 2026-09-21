package observability

import (
	"context"
	"errors"
	"time"
)

// SpanRecord is the provider-neutral, already-redacted export boundary.
// Callers must not place query strings, headers, cookies, actor/tenant IDs,
// SQL, or media names in this structure.
type SpanRecord struct {
	Name      string
	Path      string
	RequestID string
	Duration  time.Duration
	Error     bool
}

type Exporter interface {
	Export(context.Context, []SpanRecord) error
}

type ExporterAdapter struct {
	Exporter    Exporter
	Sample      func(SpanRecord) bool
	MaxAttempts int
	Timeout     time.Duration
	Backoff     time.Duration
	Sleep       func(context.Context, time.Duration) error
}

var ErrExporterUnavailable = errors.New("observability exporter unavailable")

// Export applies operator-owned sampling and bounded delivery. It never
// starts an exporter implicitly; a nil exporter is an intentional no-op.
func (a ExporterAdapter) Export(ctx context.Context, records []SpanRecord) error {
	if a.Exporter == nil || len(records) == 0 {
		return nil
	}
	selected := make([]SpanRecord, 0, len(records))
	for _, record := range records {
		if a.Sample == nil || a.Sample(record) {
			selected = append(selected, record)
		}
	}
	if len(selected) == 0 {
		return nil
	}
	attempts := a.MaxAttempts
	if attempts < 1 || attempts > 5 {
		attempts = 3
	}
	timeout := a.Timeout
	if timeout <= 0 || timeout > 30*time.Second {
		timeout = 5 * time.Second
	}
	sleep := a.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		exportCtx, cancel := context.WithTimeout(ctx, timeout)
		last = a.Exporter.Export(exportCtx, selected)
		cancel()
		if last == nil {
			return nil
		}
		if attempt < attempts {
			if err := sleep(ctx, a.Backoff*time.Duration(1<<(attempt-1))); err != nil {
				return err
			}
		}
	}
	if last == nil {
		return ErrExporterUnavailable
	}
	return last
}
