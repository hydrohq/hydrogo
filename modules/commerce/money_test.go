package commerce

import (
	"fmt"
	"math"
	"testing"
)

func TestFormatCentsAvoidsFloatingPointLoss(t *testing.T) {
	tests := map[int64]string{
		0: "0.00", 1: "0.01", 105: "1.05", 10000000000000001: "100000000000000.01",
		-1: "-0.01", math.MinInt64: "-92233720368547758.08",
	}
	for cents, want := range tests {
		if got := FormatCents(cents); got != want {
			t.Errorf("FormatCents(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestExactCentsFormatsLegacyFloatVerbExactly(t *testing.T) {
	if got := fmt.Sprintf("%.2f", ExactCents(10000000000000001)); got != "100000000000000.01" {
		t.Fatalf("legacy formatter = %q", got)
	}
}
