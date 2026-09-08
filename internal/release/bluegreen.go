package release

import "errors"

var (
	ErrInvalidTransition = errors.New("invalid blue-green deployment transition")
	ErrNotRollbackSafe   = errors.New("deployment is not rollback safe")
)

type Slot string

const (
	SlotBlue  Slot = "blue"
	SlotGreen Slot = "green"
)

type Phase string

const (
	PhaseStable   Phase = "stable"
	PhasePrepared Phase = "prepared"
	PhaseReady    Phase = "ready"
	PhaseWarmed   Phase = "warmed"
	PhaseSwitched Phase = "switched"
	PhaseDrained  Phase = "drained"
)

// Deployment is the local, deterministic part of the ADR-0005 blue-green
// sequence. A proxy integration may perform the actual traffic switch, but it
// must not skip these gates.
type Deployment struct {
	ActiveSlot             Slot
	Phase                  Phase
	PreviousAssetsRetained bool
	SchemaRollbackSafe     bool
}

func NewDeployment(active Slot) (Deployment, error) {
	if active != SlotBlue && active != SlotGreen {
		return Deployment{}, ErrInvalidTransition
	}
	return Deployment{ActiveSlot: active, Phase: PhaseStable}, nil
}

func (d *Deployment) PrepareGreen() error {
	if d == nil || d.Phase != PhaseStable || d.ActiveSlot != SlotBlue {
		return ErrInvalidTransition
	}
	d.Phase = PhasePrepared
	return nil
}

func (d *Deployment) MarkReady() error {
	if d == nil || d.Phase != PhasePrepared {
		return ErrInvalidTransition
	}
	d.Phase = PhaseReady
	return nil
}

func (d *Deployment) Warmup(retainedAssets bool) error {
	if d == nil || d.Phase != PhaseReady || !retainedAssets {
		return ErrInvalidTransition
	}
	d.PreviousAssetsRetained = true
	d.Phase = PhaseWarmed
	return nil
}

func (d *Deployment) Switch(schemaRollbackSafe bool) error {
	if d == nil || d.Phase != PhaseWarmed || !d.PreviousAssetsRetained {
		return ErrInvalidTransition
	}
	if !schemaRollbackSafe {
		return ErrNotRollbackSafe
	}
	d.SchemaRollbackSafe = true
	d.ActiveSlot = SlotGreen
	d.Phase = PhaseSwitched
	return nil
}

func (d *Deployment) DrainBlue() error {
	if d == nil || d.Phase != PhaseSwitched || d.ActiveSlot != SlotGreen {
		return ErrInvalidTransition
	}
	d.Phase = PhaseDrained
	return nil
}

func (d *Deployment) Rollback() error {
	if d == nil || (d.Phase != PhaseSwitched && d.Phase != PhaseDrained) || d.ActiveSlot != SlotGreen {
		return ErrInvalidTransition
	}
	if !d.SchemaRollbackSafe || !d.PreviousAssetsRetained {
		return ErrNotRollbackSafe
	}
	d.ActiveSlot = SlotBlue
	d.Phase = PhaseStable
	return nil
}
