package adminstudio

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const maxWorkflowStates = 32

type PublicationState string

const (
	StateDraft     PublicationState = "draft"
	StateScheduled PublicationState = "scheduled"
	StatePublished PublicationState = "published"
	StateRetracted PublicationState = "retracted"
)

var ErrInvalidTransition = errors.New("invalid publication transition")

// Workflow is a bounded, provider-neutral state machine. Modules own the
// meaning and persistence of states; Admin Studio only validates transitions.
type Workflow struct {
	States      []PublicationState                      `json:"states"`
	Transitions map[PublicationState][]PublicationState `json:"transitions"`
}

func (w Workflow) Validate() error {
	if len(w.States) == 0 || len(w.States) > maxWorkflowStates {
		return fmt.Errorf("workflow must contain 1-%d states", maxWorkflowStates)
	}
	known := make(map[PublicationState]struct{}, len(w.States))
	for _, state := range w.States {
		if !validWorkflowState(state) {
			return fmt.Errorf("invalid workflow state %q", state)
		}
		if _, exists := known[state]; exists {
			return fmt.Errorf("duplicate workflow state %q", state)
		}
		known[state] = struct{}{}
	}
	for from, targets := range w.Transitions {
		if _, ok := known[from]; !ok {
			return fmt.Errorf("transition source %q is not a workflow state", from)
		}
		if len(targets) > maxWorkflowStates {
			return fmt.Errorf("workflow transition set for %q is too large", from)
		}
		seen := make(map[PublicationState]struct{}, len(targets))
		for _, to := range targets {
			if _, ok := known[to]; !ok {
				return fmt.Errorf("transition target %q is not a workflow state", to)
			}
			if _, exists := seen[to]; exists {
				return fmt.Errorf("duplicate workflow transition %q -> %q", from, to)
			}
			seen[to] = struct{}{}
		}
	}
	return nil
}

func (w Workflow) CanTransition(from, to PublicationState) bool {
	if err := w.Validate(); err != nil {
		return false
	}
	if from == to {
		return true
	}
	for _, target := range w.Transitions[from] {
		if target == to {
			return true
		}
	}
	return false
}

func validWorkflowState(state PublicationState) bool {
	value := string(state)
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for i, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
		if i == 0 && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

// CanTransition centralizes editorial state rules without knowing a module's
// persistence or provider. Scheduling itself remains an application concern.
func CanTransition(from, to PublicationState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateDraft, StateRetracted:
		return to == StateScheduled || to == StatePublished
	case StateScheduled:
		return to == StatePublished || to == StateDraft || to == StateRetracted
	case StatePublished:
		return to == StateRetracted || to == StateDraft
	default:
		return false
	}
}

func ValidateTransition(from, to PublicationState) error {
	if !CanTransition(from, to) {
		return ErrInvalidTransition
	}
	return nil
}

// PreviewURL appends a short-lived preview marker to a same-origin path. The
// caller is responsible for signing/expiring the marker at its auth boundary.
func PreviewURL(path, token string) (string, error) {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || token == "" {
		return "", errors.New("invalid preview target")
	}
	u, err := url.Parse(path)
	if err != nil || u.Host != "" {
		return "", errors.New("invalid preview target")
	}
	q := u.Query()
	q.Set("preview", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
