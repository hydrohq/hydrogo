package main

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingDesk     = errors.New("editorial desk is required")
	ErrMissingApproval = errors.New("editorial approval is required before scheduling")
	ErrMissingHeadline = errors.New("editorial headline is required")
)

// EditorialDraft is a News Edition-specific review envelope. It stays in the
// external edition so the Core News contract remains generic.
type EditorialDraft struct {
	Headline   string
	Desk       string
	ApprovedBy string
	Scheduled  time.Time
}

func ValidateEditorialDraft(draft EditorialDraft) error {
	if strings.TrimSpace(draft.Headline) == "" {
		return ErrMissingHeadline
	}
	if strings.TrimSpace(draft.Desk) == "" {
		return ErrMissingDesk
	}
	if !draft.Scheduled.IsZero() && strings.TrimSpace(draft.ApprovedBy) == "" {
		return ErrMissingApproval
	}
	return nil
}
