package release

import (
	"errors"
	"testing"
)

func TestBlueGreenRequiresReadinessWarmupAndRollbackSafety(t *testing.T) {
	deployment, err := NewDeployment(SlotBlue)
	if err != nil {
		t.Fatal(err)
	}
	if err := deployment.Switch(true); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("switch before readiness error = %v", err)
	}
	for _, step := range []func() error{deployment.PrepareGreen, deployment.MarkReady} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if err := deployment.Warmup(false); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("warmup without retained assets error = %v", err)
	}
	if err := deployment.Warmup(true); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Switch(false); !errors.Is(err, ErrNotRollbackSafe) {
		t.Fatalf("unsafe switch error = %v", err)
	}
	if err := deployment.Switch(true); err != nil {
		t.Fatal(err)
	}
	if err := deployment.DrainBlue(); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Rollback(); err != nil {
		t.Fatal(err)
	}
	if deployment.ActiveSlot != SlotBlue || deployment.Phase != PhaseStable {
		t.Fatalf("rollback state = %#v", deployment)
	}
}

func TestBlueGreenRejectsInvalidStartingSlot(t *testing.T) {
	if _, err := NewDeployment(Slot("unknown")); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("invalid slot error = %v", err)
	}
}
