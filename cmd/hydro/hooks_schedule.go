package main

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/cron"
)

// runHookSchedules calls the services' schedules (config hooks[].schedules)
// at every matching minute, UTC. A schedule whose previous call is still
// running is skipped, and minutes missed while the server was down are not
// caught up. Run one server per schedule: every instance fires its own.
func runHookSchedules(ctx context.Context, cfg config.Config, call contenttypes.HookCaller, now func() time.Time, logger *slog.Logger) {
	scheduler := newHookScheduler(cfg, call, logger)
	if len(scheduler.entries) == 0 {
		return
	}
	for {
		wait := time.Until(now().UTC().Truncate(time.Minute).Add(time.Minute))
		if wait <= 0 || wait > time.Minute {
			wait = time.Minute
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		scheduler.fire(ctx, now().UTC().Truncate(time.Minute))
	}
}

type scheduleEntry struct {
	service, name string
	spec          cron.Spec
	running       atomic.Bool
}

type hookScheduler struct {
	entries []*scheduleEntry
	call    contenttypes.HookCaller
	logger  *slog.Logger
	done    sync.WaitGroup
}

func newHookScheduler(cfg config.Config, call contenttypes.HookCaller, logger *slog.Logger) *hookScheduler {
	scheduler := &hookScheduler{call: call, logger: logger}
	for _, hook := range cfg.Hooks {
		for _, schedule := range hook.Schedules {
			if spec, err := cron.Parse(schedule.Cron); err == nil {
				scheduler.entries = append(scheduler.entries, &scheduleEntry{service: hook.ID, name: schedule.Name, spec: spec})
			}
		}
	}
	return scheduler
}

// fire starts the calls of every schedule matching the minute.
func (s *hookScheduler) fire(ctx context.Context, minute time.Time) {
	for _, item := range s.entries {
		if !item.spec.Matches(minute) || !item.running.CompareAndSwap(false, true) {
			continue
		}
		s.done.Add(1)
		go func(item *scheduleEntry) {
			defer s.done.Done()
			defer item.running.Store(false)
			status, _, err := s.call(ctx, item.service, contenttypes.HookEvent{Type: "schedule", On: item.name, Fields: map[string]any{"at": minute.Format(time.RFC3339)}})
			if err != nil || status < 200 || status >= 300 {
				s.logger.WarnContext(ctx, "scheduled hook call failed", "service", item.service, "schedule", item.name, "status", status, "error", err)
			}
		}(item)
	}
}
