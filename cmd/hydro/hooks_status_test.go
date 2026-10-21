package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

func TestHookStatusCountsOnlyBrokenCallsAsFailures(t *testing.T) {
	status := newHookStatus()
	status_, err := 200, error(nil)
	call := status.wrap(func(context.Context, string, contenttypes.HookEvent) (int, string, error) {
		return status_, "", err
	})
	event := contenttypes.HookEvent{Type: "invoices", On: "transition"}
	for _, step := range []struct {
		status int
		err    error
	}{{http.StatusNoContent, nil}, {http.StatusUnprocessableEntity, nil}, {http.StatusBadGateway, nil}, {0, errors.New("dial tcp: refused")}} {
		status_, err = step.status, step.err
		_, _, _ = call(context.Background(), "ledger", event)
	}
	got := status.snapshot("ledger")
	if got.Calls != 4 || got.Failures != 2 || got.LastError != "unreachable" || got.LastEvent != "invoices.transition" {
		t.Fatalf("state = %+v", got)
	}
	if status.snapshot("other").Calls != 0 {
		t.Fatal("an unknown service has history")
	}
}

// The Services tab reads these keys; a missing json tag once rendered them
// as "Name"/"Cron" and the tab showed "undefined (undefined)".
func TestServiceInfoSchedulesUseLowercaseJSONKeys(t *testing.T) {
	raw, err := json.Marshal(serviceInfo{ID: "ledger", Schedules: []config.HookSchedule{{Name: "overdue-check", Cron: "0 2 * * *"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"schedules":[{"name":"overdue-check","cron":"0 2 * * *"}]`) {
		t.Fatalf("json = %s", raw)
	}
}
