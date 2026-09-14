package ui

import (
	"errors"
	"fmt"
)

type FrontendState string

const (
	StateIdle    FrontendState = "idle"
	StateLoading FrontendState = "loading"
	StateSuccess FrontendState = "success"
	StateEmpty   FrontendState = "empty"
	StateError   FrontendState = "error"
)

type FrontendStateSnapshot struct {
	State       FrontendState `json:"state"`
	ErrorCode   string        `json:"error_code,omitempty"`
	UserMessage string        `json:"user_message,omitempty"`
	Optimistic  bool          `json:"optimistic,omitempty"`
}

var ErrInvalidFrontendState = errors.New("invalid frontend state")

func (s FrontendStateSnapshot) Validate() error {
	switch s.State {
	case StateIdle, StateLoading, StateSuccess, StateEmpty, StateError:
	default:
		return fmt.Errorf("%w: unsupported state %q", ErrInvalidFrontendState, s.State)
	}
	if s.State == StateError && s.UserMessage == "" {
		return fmt.Errorf("%w: error state requires a user message", ErrInvalidFrontendState)
	}
	if s.State != StateError && (s.ErrorCode != "" || s.UserMessage != "") {
		return fmt.Errorf("%w: error details are only valid for error state", ErrInvalidFrontendState)
	}
	if s.Optimistic && s.State != StateLoading && s.State != StateSuccess && s.State != StateError {
		return fmt.Errorf("%w: optimistic state must be loading, success, or error", ErrInvalidFrontendState)
	}
	return nil
}
