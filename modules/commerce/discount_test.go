package commerce

import (
	"testing"
	"time"
)

func TestDiscountForAppliesAPercentageToTheSubtotal(t *testing.T) {
	code := DiscountCode{Active: true, Kind: DiscountPercent, Value: 10}
	got, err := code.DiscountFor(10000, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got != 1000 {
		t.Fatalf("discount = %d, want 1000 (10%% of 10000)", got)
	}
}

func TestDiscountForAppliesAFixedAmount(t *testing.T) {
	code := DiscountCode{Active: true, Kind: DiscountFixed, Value: 1500}
	got, err := code.DiscountFor(10000, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got != 1500 {
		t.Fatalf("discount = %d, want 1500", got)
	}
}

func TestDiscountForCapsAFixedAmountAtTheSubtotal(t *testing.T) {
	code := DiscountCode{Active: true, Kind: DiscountFixed, Value: 50000}
	got, err := code.DiscountFor(10000, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got != 10000 {
		t.Fatalf("discount = %d, want capped at the 10000 subtotal", got)
	}
}

func TestDiscountForRejectsAnInactiveCode(t *testing.T) {
	code := DiscountCode{Active: false, Kind: DiscountFixed, Value: 100}
	if _, err := code.DiscountFor(10000, time.Now()); err == nil {
		t.Fatal("expected an error for an inactive code")
	}
}

func TestDiscountForRejectsOutsideItsDateWindow(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	notYet := DiscountCode{Active: true, Kind: DiscountFixed, Value: 100, StartsAt: now.Add(24 * time.Hour)}
	if _, err := notYet.DiscountFor(10000, now); err == nil {
		t.Fatal("expected an error for a code that has not started yet")
	}
	expired := DiscountCode{Active: true, Kind: DiscountFixed, Value: 100, EndsAt: now.Add(-24 * time.Hour)}
	if _, err := expired.DiscountFor(10000, now); err == nil {
		t.Fatal("expected an error for an expired code")
	}
	withinWindow := DiscountCode{Active: true, Kind: DiscountFixed, Value: 100, StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour)}
	if _, err := withinWindow.DiscountFor(10000, now); err != nil {
		t.Fatalf("code within its window should apply: %v", err)
	}
}

func TestDiscountForRejectsWhenRedemptionsAreExhausted(t *testing.T) {
	code := DiscountCode{Active: true, Kind: DiscountFixed, Value: 100, MaxRedemptions: 2, RedeemedCount: 2}
	if _, err := code.DiscountFor(10000, time.Now()); err == nil {
		t.Fatal("expected an error once max_redemptions is reached")
	}
}

func TestDiscountForRejectsBelowTheMinimumSubtotal(t *testing.T) {
	code := DiscountCode{Active: true, Kind: DiscountFixed, Value: 100, MinSubtotalCents: 20000}
	if _, err := code.DiscountFor(10000, time.Now()); err == nil {
		t.Fatal("expected an error when the subtotal is below the code's minimum")
	}
	if _, err := code.DiscountFor(20000, time.Now()); err != nil {
		t.Fatalf("subtotal at the minimum should apply: %v", err)
	}
}
