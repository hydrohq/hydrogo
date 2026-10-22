package main

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

func TestHookSchedulerFiresMatchingMinutesAndSkipsOverlap(t *testing.T) {
	var mu sync.Mutex
	var calls []contenttypes.HookEvent
	release := make(chan struct{})
	call := func(_ context.Context, service string, event contenttypes.HookEvent) (int, string, error) {
		mu.Lock()
		calls = append(calls, event)
		mu.Unlock()
		if event.On == "slow" {
			<-release
		}
		return 200, "", nil
	}
	cfg := config.Config{Hooks: []config.SyncHook{{ID: "ledger", Schedules: []config.HookSchedule{
		{Name: "nightly", Cron: "0 2 * * *"},
		{Name: "slow", Cron: "* * * * *"},
	}}}}
	s := newHookScheduler(cfg, call, slog.New(slog.NewTextHandler(io.Discard, nil)))
	at := func(v string) time.Time { p, _ := time.Parse(time.RFC3339, v); return p }

	s.fire(context.Background(), at("2026-10-21T02:00:00Z")) // nightly + slow
	s.fire(context.Background(), at("2026-10-21T02:01:00Z")) // slow still running: skipped
	close(release)
	s.done.Wait()
	s.fire(context.Background(), at("2026-10-21T03:00:00Z")) // slow only
	s.done.Wait()

	counts := map[string]int{}
	for _, event := range calls {
		if event.Type != "schedule" || event.Fields["at"] == "" {
			t.Fatalf("event = %+v", event)
		}
		counts[event.On]++
	}
	if counts["nightly"] != 1 || counts["slow"] != 2 {
		t.Fatalf("calls = %v", counts)
	}
}
