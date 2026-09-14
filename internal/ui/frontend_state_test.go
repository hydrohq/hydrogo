package ui

import (
	"errors"
	"testing"
)

func TestFrontendStateSnapshotsValidate(t *testing.T) {
	valid := []FrontendStateSnapshot{
		{State: StateIdle},
		{State: StateLoading, Optimistic: true},
		{State: StateSuccess},
		{State: StateEmpty},
		{State: StateError, ErrorCode: "network", UserMessage: "Tekrar deneyin."},
	}
	for _, state := range valid {
		if err := state.Validate(); err != nil {
			t.Errorf("valid state %+v rejected: %v", state, err)
		}
	}
	invalid := []FrontendStateSnapshot{
		{State: "unknown"},
		{State: StateError},
		{State: StateSuccess, UserMessage: "oops"},
		{State: StateEmpty, Optimistic: true},
	}
	for _, state := range invalid {
		if err := state.Validate(); !errors.Is(err, ErrInvalidFrontendState) {
			t.Errorf("invalid state %+v returned %v", state, err)
		}
	}
}
