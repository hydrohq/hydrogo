package auth

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidTokenLifetime = errors.New("unsupported security token lifetime")

type TokenLifetimes struct {
	PasswordReset time.Duration
	Invitation    time.Duration
}

type TokenLifetimeSettings interface {
	Read(context.Context) (TokenLifetimes, bool, error)
	Write(context.Context, TokenLifetimes) error
}

func (v TokenLifetimes) Validate() error {
	validReset := v.PasswordReset == 15*time.Minute || v.PasswordReset == time.Hour || v.PasswordReset == 24*time.Hour
	validInvite := v.Invitation == 24*time.Hour || v.Invitation == 72*time.Hour || v.Invitation == 7*24*time.Hour
	if !validReset || !validInvite {
		return ErrInvalidTokenLifetime
	}
	return nil
}
