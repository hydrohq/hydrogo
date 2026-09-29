package linkcheck

import (
	"context"
	"time"
)

// NextMonthlyScan returns the next UTC first-of-month scan time at 03:00.
func NextMonthlyScan(now time.Time) time.Time {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month()+1, 1, 3, 0, 0, 0, time.UTC)
	if now.Before(time.Date(now.Year(), now.Month(), 1, 3, 0, 0, 0, time.UTC)) {
		return time.Date(now.Year(), now.Month(), 1, 3, 0, 0, 0, time.UTC)
	}
	return next
}

// RunMonthly scans each configured tenant once per month. Coordination across
// server instances is intentionally left to deployment policy.
func RunMonthly(ctx context.Context, manager *Manager, tenantKeys []string, startURL string, reportError func(error)) {
	if manager == nil || startURL == "" {
		return
	}
	for {
		delay := time.Until(NextMonthlyScan(time.Now()))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		for _, tenantKey := range tenantKeys {
			if _, err := manager.Start(tenantKey, startURL); err != nil && reportError != nil {
				reportError(err)
			}
		}
	}
}
