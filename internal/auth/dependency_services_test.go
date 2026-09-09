package auth

import (
	"context"
	"errors"
	"testing"
)

func TestRecoveryRejectsTypedNilDependencies(t *testing.T) {
	var users *httpUserRepo
	var tokens *httpRecoveryStore
	var sessions *httpSessionRepo
	recovery := PasswordRecovery{Users: users, Tokens: tokens, Sessions: sessions}
	if _, err := recovery.Issue(context.Background(), nil, "user@example.test", 0); !errors.Is(err, ErrRecoveryDependencyRequired) {
		t.Fatalf("Issue error = %v, want %v", err, ErrRecoveryDependencyRequired)
	}
	if err := recovery.Reset(context.Background(), nil, "token", "password"); !errors.Is(err, ErrRecoveryDependencyRequired) {
		t.Fatalf("Reset error = %v, want %v", err, ErrRecoveryDependencyRequired)
	}
}

func TestInvitationRejectsTypedNilDependencies(t *testing.T) {
	var users *httpUserRepo
	var invites *httpRecoveryStore
	invitation := InvitationService{Users: users, Invites: invites}
	if _, err := invitation.Issue(context.Background(), nil, "user@example.test", RoleEditor, 0); !errors.Is(err, ErrInviteDependencyRequired) {
		t.Fatalf("Issue error = %v, want %v", err, ErrInviteDependencyRequired)
	}
	if _, err := invitation.Accept(context.Background(), nil, "token", "password"); !errors.Is(err, ErrInviteDependencyRequired) {
		t.Fatalf("Accept error = %v, want %v", err, ErrInviteDependencyRequired)
	}
}
