package v1

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const MaxWorkflowStates = 32
const MaxWorkflowRoles = 16

type EditorialState string

const (
	EditorialDraft     EditorialState = "draft"
	EditorialScheduled EditorialState = "scheduled"
	EditorialPublished EditorialState = "published"
	EditorialRetracted EditorialState = "retracted"
)

var ErrInvalidEditorialTransition = errors.New("invalid editorial transition")

// EditorialWorkflow is a bounded state machine. Persistence, scheduling and
// authorization remain module-owned policies.
type EditorialWorkflow struct {
	States          []EditorialState                               `json:"states"`
	Transitions     map[EditorialState][]EditorialState            `json:"transitions"`
	RoleTransitions map[string]map[EditorialState][]EditorialState `json:"role_transitions,omitempty"`
}

func (w EditorialWorkflow) Validate() error {
	if len(w.States) == 0 || len(w.States) > MaxWorkflowStates {
		return fmt.Errorf("workflow must contain 1-%d states", MaxWorkflowStates)
	}
	known := make(map[EditorialState]struct{}, len(w.States))
	for _, state := range w.States {
		if !validEditorialState(state) {
			return fmt.Errorf("invalid workflow state %q", state)
		}
		if _, ok := known[state]; ok {
			return fmt.Errorf("duplicate workflow state %q", state)
		}
		known[state] = struct{}{}
	}
	for from, targets := range w.Transitions {
		if _, ok := known[from]; !ok {
			return fmt.Errorf("transition source %q is not a workflow state", from)
		}
		if len(targets) > MaxWorkflowStates {
			return fmt.Errorf("workflow transition set for %q is too large", from)
		}
		seen := map[EditorialState]struct{}{}
		for _, to := range targets {
			if _, ok := known[to]; !ok {
				return fmt.Errorf("transition target %q is not a workflow state", to)
			}
			if _, ok := seen[to]; ok {
				return fmt.Errorf("duplicate workflow transition %q -> %q", from, to)
			}
			seen[to] = struct{}{}
		}
	}
	if len(w.RoleTransitions) > MaxWorkflowRoles {
		return fmt.Errorf("workflow role policy set is too large")
	}
	for role, transitions := range w.RoleTransitions {
		if !validWorkflowRole(role) {
			return fmt.Errorf("invalid workflow role %q", role)
		}
		for from, targets := range transitions {
			if _, ok := known[from]; !ok {
				return fmt.Errorf("role transition source %q is not a workflow state", from)
			}
			if len(targets) > MaxWorkflowStates {
				return fmt.Errorf("role transition set for %q is too large", from)
			}
			seen := map[EditorialState]struct{}{}
			for _, to := range targets {
				if _, ok := known[to]; !ok {
					return fmt.Errorf("role transition target %q is not a workflow state", to)
				}
				if _, ok := seen[to]; ok {
					return fmt.Errorf("duplicate role workflow transition %q -> %q", from, to)
				}
				seen[to] = struct{}{}
			}
		}
	}
	return nil
}

func (w EditorialWorkflow) CanTransition(from, to EditorialState) bool {
	return w.canTransition(from, to, "")
}

// CanTransitionForRole applies an optional role-specific allowlist. Roles
// without an explicit policy retain the workflow's general transition graph.
func (w EditorialWorkflow) CanTransitionForRole(role string, from, to EditorialState) bool {
	return w.canTransition(from, to, role)
}

func (w EditorialWorkflow) canTransition(from, to EditorialState, role string) bool {
	if w.Validate() != nil {
		return false
	}
	if from == to {
		return true
	}
	if role != "" {
		if policy, ok := w.RoleTransitions[role]; ok {
			for _, target := range policy[from] {
				if target == to {
					return true
				}
			}
			return false
		}
	}
	for _, target := range w.Transitions[from] {
		if target == to {
			return true
		}
	}
	return false
}

func validWorkflowRole(role string) bool {
	if len(role) == 0 || len(role) > 64 {
		return false
	}
	for i, r := range role {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
		if i == 0 && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

func ValidateEditorialTransition(w EditorialWorkflow, from, to EditorialState) error {
	if !w.CanTransition(from, to) {
		return ErrInvalidEditorialTransition
	}
	return nil
}

func validEditorialState(state EditorialState) bool {
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

// PreviewPath adds a preview marker to a same-origin path. Token signing and
// expiry are enforced by the host application.
func PreviewPath(path, token string) (string, error) {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || token == "" {
		return "", errors.New("invalid preview target")
	}
	u, err := url.Parse(path)
	if err != nil || u.Host != "" || u.Fragment != "" {
		return "", errors.New("invalid preview target")
	}
	q := u.Query()
	q.Set("preview", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
