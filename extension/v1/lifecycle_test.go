package v1

import (
	"errors"
	"testing"
)

func TestExtensionPackageLifecycleTransitions(t *testing.T) {
	valid := [][2]PackageState{{StateInstalled, StateApproved}, {StateApproved, StateEnabled}, {StateEnabled, StateSuspended}, {StateSuspended, StateEnabled}, {StateEnabled, StateRevoked}}
	for _, pair := range valid {
		if err := Transition(pair[0], pair[1]); err != nil {
			t.Fatalf("%s -> %s: %v", pair[0], pair[1], err)
		}
	}
	for _, pair := range [][2]PackageState{{StateInstalled, StateEnabled}, {StateRevoked, StateEnabled}, {StateRevoked, StateApproved}} {
		if err := Transition(pair[0], pair[1]); !errors.Is(err, ErrInvalidPackageTransition) {
			t.Fatalf("%s -> %s error = %v", pair[0], pair[1], err)
		}
	}
}
