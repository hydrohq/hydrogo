package commerce

import "testing"

func TestPaymentMismatchIsDistinct(t *testing.T) {
	if ErrPaymentMismatch == ErrInvalidPaymentEvent {
		t.Fatal("payment mismatch must be distinct from malformed events")
	}
}
