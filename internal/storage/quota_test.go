package storage

import (
	"errors"
	"testing"
)

func TestQuotaGuardReservesCapacityBeforeWrite(t *testing.T) {
	guard := QuotaGuard{MinFreeBytes: 100, Probe: func(string) (uint64, error) { return 150, nil }}
	if err := guard.Check("/media", 50); err != nil {
		t.Fatalf("exact reserve rejected: %v", err)
	}
	if err := guard.Check("/media", 51); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("low space error = %v", err)
	}
	if err := (QuotaGuard{MinFreeBytes: 0}).Check("", 0); err != nil {
		t.Fatalf("disabled guard = %v", err)
	}
}
