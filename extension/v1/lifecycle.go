package v1

import (
	"errors"
	"fmt"
)

type PackageState string

const (
	StateInstalled PackageState = "installed"
	StateApproved  PackageState = "approved"
	StateEnabled   PackageState = "enabled"
	StateSuspended PackageState = "suspended"
	StateRevoked   PackageState = "revoked"
)

var ErrInvalidPackageTransition = errors.New("invalid extension package state transition")

// CanTransition is the runtime lifecycle guard. It deliberately does not
// perform filesystem or authorization work; callers must verify the artifact
// and operator grants before moving into an active state.
func CanTransition(from, to PackageState) bool {
	switch from {
	case StateInstalled:
		return to == StateApproved || to == StateRevoked
	case StateApproved:
		return to == StateEnabled || to == StateSuspended || to == StateRevoked
	case StateEnabled:
		return to == StateSuspended || to == StateRevoked
	case StateSuspended:
		return to == StateEnabled || to == StateRevoked
	case StateRevoked:
		return false
	default:
		return false
	}
}

func Transition(from, to PackageState) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidPackageTransition, from, to)
	}
	return nil
}
