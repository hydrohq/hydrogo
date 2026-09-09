package auth

import (
	"context"
	"errors"
	"testing"
)

func TestMFAServiceFailsClosedWhenStoreIsMissing(t *testing.T) {
	service := MFAService{}
	if _, err := service.Enabled(context.Background(), nil, "user"); !errors.Is(err, ErrMFAStoreRequired) {
		t.Fatalf("missing store Enabled error = %v, want %v", err, ErrMFAStoreRequired)
	}
	if _, err := service.Enroll(context.Background(), nil, "user", "user@example.test"); !errors.Is(err, ErrMFAStoreRequired) {
		t.Fatalf("missing store Enroll error = %v, want %v", err, ErrMFAStoreRequired)
	}
	if err := service.Verify(context.Background(), nil, "user", "123456"); !errors.Is(err, ErrMFAStoreRequired) {
		t.Fatalf("missing store Verify error = %v, want %v", err, ErrMFAStoreRequired)
	}
	if err := service.ConsumeRecoveryCode(context.Background(), nil, "user", "code"); !errors.Is(err, ErrMFAStoreRequired) {
		t.Fatalf("missing store recovery error = %v, want %v", err, ErrMFAStoreRequired)
	}
}
