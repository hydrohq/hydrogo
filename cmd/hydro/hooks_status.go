package main

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

// hookStatus remembers, per hook service, how its calls went since the server
// started. It lives in memory only: it answers "is the service reachable now"
// for the admin, nothing more.
type hookStatus struct {
	mu      sync.Mutex
	entries map[string]*hookState
}

type hookState struct {
	Calls      int       `json:"calls"`
	Failures   int       `json:"failures"`
	LastAt     time.Time `json:"last_at,omitzero"`
	LastStatus int       `json:"last_status,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
	LastEvent  string    `json:"last_event,omitempty"`
}

func newHookStatus() *hookStatus { return &hookStatus{entries: map[string]*hookState{}} }

// wrap records every call of the service and passes the answer through.
func (s *hookStatus) wrap(call contenttypes.HookCaller) contenttypes.HookCaller {
	return func(ctx context.Context, service string, event contenttypes.HookEvent) (int, string, error) {
		status, message, err := call(ctx, service, event)
		s.mu.Lock()
		defer s.mu.Unlock()
		state := s.entries[service]
		if state == nil {
			state = &hookState{}
			s.entries[service] = state
		}
		state.Calls++
		state.LastAt = time.Now().UTC()
		state.LastStatus = status
		state.LastEvent = event.Type + "." + event.On
		state.LastError = ""
		// 4xx is the service saying no on purpose; only errors and 5xx-like
		// answers count as a service that is not working.
		if err != nil || status < 200 || (status >= 300 && (status < 400 || status >= 500)) {
			state.Failures++
			switch {
			case err != nil:
				state.LastError = "unreachable"
			default:
				state.LastError = http.StatusText(status)
			}
		}
		return status, message, err
	}
}

func (s *hookStatus) snapshot(id string) hookState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state := s.entries[id]; state != nil {
		return *state
	}
	return hookState{}
}

type serviceInfo struct {
	ID        string                `json:"id"`
	Host      string                `json:"host"`
	Timeout   string                `json:"timeout"`
	Schedules []config.HookSchedule `json:"schedules"`
	UsedBy    []string              `json:"used_by"`
	hookState
}

// hookCaller returns the one recording caller shared by content types and
// schedules.
func (a *serveApp) hookCaller() contenttypes.HookCaller {
	a.hookOnce.Do(func() {
		a.hookStatus = newHookStatus()
		a.hookCall = a.hookStatus.wrap(syncHookCaller(a.cfg))
	})
	return a.hookCall
}

// adminServicesHandler lists the configured hook services with how their
// calls went. The URL is reduced to its host: paths and queries can carry
// secrets.
func (a *serveApp) adminServicesHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionOperations) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	a.hookCaller()
	services := make([]serviceInfo, 0, len(a.cfg.Hooks))
	for _, hook := range a.cfg.Hooks {
		info := serviceInfo{ID: hook.ID, Timeout: hook.Timeout, Schedules: hook.Schedules, UsedBy: []string{}, hookState: a.hookStatus.snapshot(hook.ID)}
		if info.Schedules == nil {
			info.Schedules = []config.HookSchedule{}
		}
		if parsed, err := url.Parse(hook.URL); err == nil {
			info.Host = parsed.Host
		}
		for _, definition := range a.contentTypes {
			for _, h := range definition.Hooks {
				if h.Service == hook.ID {
					info.UsedBy = append(info.UsedBy, definition.ID)
					break
				}
			}
		}
		sort.Strings(info.UsedBy)
		services = append(services, info)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, map[string]any{"services": services}, nil)
}
