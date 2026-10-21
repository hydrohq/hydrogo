package contenttypes

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Hook asks a side service for permission before a change is saved: the
// server sends the service a signed request (the configured hook named
// Service, see config `hooks:`) and saves only if it answers 2xx. A 4xx
// answer rejects the change with the service's {"message": "..."}; any other
// failure (timeout, 5xx, unreachable) rejects it too, unless FailOpen is set.
// Hooks run before the change and outside its transaction, so the service
// decides on what the caller submitted; the record's version check still
// guards the save itself.
//
//	hooks:
//	  - {on: transition, to: posted, service: ledger}
//	  - {on: create, service: credit-check, fail_open: true}
type Hook struct {
	// On is create, update, delete or transition (a state step; From and To
	// narrow it).
	On       string `json:"on" yaml:"on"`
	From     string `json:"from,omitempty" yaml:"from"`
	To       string `json:"to,omitempty" yaml:"to"`
	Service  string `json:"service" yaml:"service"`
	FailOpen bool   `json:"fail_open,omitempty" yaml:"fail_open"`
}

// HookEvent is what the service receives (as the event data of the signed
// request, topic "hook.<type>.<on>").
type HookEvent struct {
	Type   string         `json:"type"`
	On     string         `json:"on"`
	ID     string         `json:"id,omitempty"`
	From   string         `json:"from,omitempty"`
	To     string         `json:"to,omitempty"`
	Actor  string         `json:"actor,omitempty"`
	Fields map[string]any `json:"fields,omitempty"`
}

// HookCaller delivers one event to the configured hook service and returns
// the HTTP status it answered plus its message (4xx), or an error when it
// could not be reached.
type HookCaller func(ctx context.Context, service string, event HookEvent) (status int, message string, err error)

// hookError is a rejected change: Status is the HTTP status to answer.
type hookError struct {
	status  int
	message string
}

func (e *hookError) Error() string { return e.message }

const (
	hookOn     = "create update delete transition"
	maxHooks   = 10
	maxMessage = 200
)

// runHooks consults every hook of the type that matches the event.
func (a AdminAPI) runHooks(r *http.Request, definition Definition, event HookEvent) error {
	for _, hook := range definition.Hooks {
		if hook.On != event.On || (hook.From != "" && hook.From != event.From) || (hook.To != "" && hook.To != event.To) {
			continue
		}
		if a.Hook == nil {
			if hook.FailOpen {
				continue
			}
			return &hookError{http.StatusServiceUnavailable, "a required check is not available"}
		}
		event.Type, event.Actor = definition.ID, a.actorOf(r)
		status, message, err := a.Hook(r.Context(), hook.Service, event)
		switch {
		case err == nil && status >= 200 && status < 300:
		case err == nil && status >= 400 && status < 500:
			if message == "" {
				message = "rejected by " + hook.Service
			}
			return &hookError{http.StatusUnprocessableEntity, message}
		case hook.FailOpen:
		default:
			return &hookError{http.StatusServiceUnavailable, "the check by " + hook.Service + " is not available"}
		}
	}
	return nil
}

func (a AdminAPI) actorOf(r *http.Request) string {
	if a.Actor == nil {
		return ""
	}
	return a.Actor(r)
}

// hookGuard runs the hooks for one change and answers the error itself.
func (a AdminAPI) hookGuard(w http.ResponseWriter, r *http.Request, definition Definition, event HookEvent) bool {
	if err := a.runHooks(r, definition, event); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return false
	}
	return true
}

func (d Definition) validateHooks() error {
	if len(d.Hooks) > maxHooks {
		return fmt.Errorf("%w: at most %d hooks", ErrInvalidDefinition, maxHooks)
	}
	for _, hook := range d.Hooks {
		switch hook.On {
		case "create", "update", "delete":
			if hook.From != "" || hook.To != "" {
				return fmt.Errorf("%w: from and to only go with on: transition", ErrInvalidDefinition)
			}
		case "transition":
			if d.States == nil {
				return fmt.Errorf("%w: on: transition needs states", ErrInvalidDefinition)
			}
			for _, state := range []string{hook.From, hook.To} {
				if state != "" && !slices.Contains(d.stateNames(), state) {
					return fmt.Errorf("%w: hook names unknown state %q", ErrInvalidDefinition, state)
				}
			}
		default:
			return fmt.Errorf("%w: hook on must be one of: %s", ErrInvalidDefinition, hookOn)
		}
		if !hookService.MatchString(hook.Service) {
			return fmt.Errorf("%w: hook service %q must be a configured hook id", ErrInvalidDefinition, hook.Service)
		}
	}
	return nil
}

func (d Definition) stateNames() []string {
	var names []string
	for _, field := range d.Fields {
		if d.States != nil && field.ID == d.States.Field {
			names = field.Options
		}
	}
	return names
}

var hookService = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,39}$`)
