package release

import (
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
)

// Traffic routes requests to the active deployment slot. It is deliberately
// process-local; a reverse proxy integration can use the same gate sequence.
type Traffic struct {
	blue               http.Handler
	green              http.Handler
	generationProvider func() uint64
	active             atomic.Value // trafficState
}

type trafficState struct {
	slot       Slot
	generation uint64
}

func NewTraffic(blue, green http.Handler) (*Traffic, error) {
	if blue == nil || green == nil {
		return nil, errors.New("both blue and green handlers are required")
	}
	t := &Traffic{blue: blue, green: green}
	t.active.Store(trafficState{slot: SlotBlue})
	return t, nil
}

// SetGenerationProvider binds request ingress to the cache generation used by
// the surrounding deployment. It is optional for integrations that use the
// traffic router without a generation-aware cache.
func (t *Traffic) SetGenerationProvider(provider func() uint64) {
	if t != nil {
		t.generationProvider = provider
	}
}

func (t *Traffic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t == nil {
		http.Error(w, "traffic router unavailable", http.StatusServiceUnavailable)
		return
	}
	// Capture the cache generation before reading the active slot. This ordering
	// prevents a request that observed the old slot from pairing it with a
	// generation created by a concurrent switch or rollback. Such a pairing
	// could let stale slot data refill the current cache generation.
	if t.generationProvider != nil {
		r = r.WithContext(cache.WithGeneration(r.Context(), t.generationProvider()))
	}
	state := t.active.Load().(trafficState)
	if state.slot == SlotGreen {
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
	state := t.active.Load().(trafficState)
	t.active.Store(trafficState{slot: SlotGreen, generation: state.generation + 1})
	return nil
}

// Rollback publishes blue only after the deployment state has returned to stable.
func (t *Traffic) Rollback(d Deployment) error {
	if t == nil || d.ActiveSlot != SlotBlue || d.Phase != PhaseStable || !d.SchemaRollbackSafe {
		return ErrNotRollbackSafe
	}
	state := t.active.Load().(trafficState)
	t.active.Store(trafficState{slot: SlotBlue, generation: state.generation + 1})
	return nil
}
