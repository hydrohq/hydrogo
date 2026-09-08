package release

import (
	"errors"
	"net/http"
	"sync/atomic"
)

// Traffic routes requests to the active deployment slot. It is deliberately
// process-local; a reverse proxy integration can use the same gate sequence.
type Traffic struct {
	blue   http.Handler
	green  http.Handler
	active atomic.Value // Slot
}

func NewTraffic(blue, green http.Handler) (*Traffic, error) {
	if blue == nil || green == nil {
		return nil, errors.New("both blue and green handlers are required")
	}
	t := &Traffic{blue: blue, green: green}
	t.active.Store(SlotBlue)
	return t, nil
}

func (t *Traffic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t == nil {
		http.Error(w, "traffic router unavailable", http.StatusServiceUnavailable)
		return
	}
	if t.active.Load().(Slot) == SlotGreen {
		t.green.ServeHTTP(w, r)
		return
	}
	t.blue.ServeHTTP(w, r)
}

// Switch publishes green only after all deployment gates have passed.
func (t *Traffic) Switch(d Deployment) error {
	if t == nil || d.ActiveSlot != SlotGreen || d.Phase != PhaseSwitched {
		return ErrInvalidTransition
	}
	t.active.Store(SlotGreen)
	return nil
}

// Rollback publishes blue only after the deployment state has returned to stable.
func (t *Traffic) Rollback(d Deployment) error {
	if t == nil || d.ActiveSlot != SlotBlue || d.Phase != PhaseStable || !d.SchemaRollbackSafe {
		return ErrNotRollbackSafe
	}
	t.active.Store(SlotBlue)
	return nil
}
