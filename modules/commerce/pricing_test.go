package commerce

import "testing"

func TestPricingWithNoTaxOrShippingConfiguredJustPassesSubtotalThrough(t *testing.T) {
	var pricing PricingConfig
	got := pricing.Price(10000, 0)
	want := PricingBreakdown{SubtotalCents: 10000, TotalCents: 10000}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPricingExclusiveTaxAddsOnTopOfSubtotal(t *testing.T) {
	pricing := PricingConfig{TaxRate: 0.20}
	got := pricing.Price(10000, 0)
	if got.TaxCents != 2000 {
		t.Fatalf("tax = %d, want 2000 (20%% of 10000)", got.TaxCents)
	}
	if got.TotalCents != 12000 {
		t.Fatalf("total = %d, want 12000", got.TotalCents)
	}
}

func TestPricingInclusiveTaxIsBackedOutNotAddedOnTop(t *testing.T) {
	pricing := PricingConfig{TaxRate: 0.20, TaxInclusive: true}
	got := pricing.Price(12000, 0)
	// 12000 already includes 20% KDV: the pre-tax base is 10000, tax is 2000,
	// but the total charged stays 12000 -- the rate is informational only.
	if got.TaxCents != 2000 {
		t.Fatalf("tax = %d, want 2000 (backed out of the inclusive price)", got.TaxCents)
	}
	if got.TotalCents != 12000 {
		t.Fatalf("total = %d, want 12000 (tax is already in the price, not added again)", got.TotalCents)
	}
}

func TestPricingDiscountAppliesBeforeTax(t *testing.T) {
	pricing := PricingConfig{TaxRate: 0.20}
	got := pricing.Price(10000, 2000)
	// (10000 - 2000) * 1.20 = 9600
	if got.TaxCents != 1600 {
		t.Fatalf("tax = %d, want 1600 (20%% of the discounted 8000)", got.TaxCents)
	}
	if got.TotalCents != 9600 {
		t.Fatalf("total = %d, want 9600", got.TotalCents)
	}
	if got.DiscountCents != 2000 {
		t.Fatalf("discount = %d, want 2000", got.DiscountCents)
	}
}

func TestPricingFlatShippingAddsOnTopOfTotal(t *testing.T) {
	pricing := PricingConfig{ShippingFlatRateCents: 5000}
	got := pricing.Price(10000, 0)
	if got.ShippingCents != 5000 || got.TotalCents != 15000 {
		t.Fatalf("got %+v", got)
	}
}

func TestPricingFreeShippingThresholdWaivesTheFlatRate(t *testing.T) {
	pricing := PricingConfig{ShippingFlatRateCents: 5000, ShippingFreeAboveCents: 10000}
	below := pricing.Price(9999, 0)
	if below.ShippingCents != 5000 {
		t.Fatalf("below threshold: shipping = %d, want 5000", below.ShippingCents)
	}
	atThreshold := pricing.Price(10000, 0)
	if atThreshold.ShippingCents != 0 {
		t.Fatalf("at threshold: shipping = %d, want 0 (free)", atThreshold.ShippingCents)
	}
}

func TestPricingDiscountCannotMakeTheTaxableBaseNegative(t *testing.T) {
	pricing := PricingConfig{TaxRate: 0.20}
	got := pricing.Price(1000, 5000)
	if got.TotalCents < 0 {
		t.Fatalf("total went negative: %+v", got)
	}
	if got.TotalCents != 0 {
		t.Fatalf("an over-large discount should floor the total at 0, got %+v", got)
	}
}
