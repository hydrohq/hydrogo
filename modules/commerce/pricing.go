package commerce

import "math"

// PricingConfig is operator-set, not per-order: cmd/hydro builds one from
// config.Commerce and gives every Service the same values. The zero value
// (TaxRate 0, no shipping) reproduces Phase 2's original behaviour exactly
// -- total is just the subtotal -- so a project that sets none of this up
// is unaffected.
type PricingConfig struct {
	// TaxRate is the KDV/VAT rate, e.g. 0.20 for 20%. Zero disables tax.
	TaxRate float64
	// TaxInclusive means catalog prices already include tax: TaxCents on
	// the resulting breakdown is then informational only (backed out of
	// the price for display/reporting), not added on top of it. False
	// means catalog prices are pre-tax and TaxCents is added to the total.
	TaxInclusive bool
	// ShippingFlatRateCents is a single flat shipping fee. Zero disables
	// shipping entirely (a project not ready to model shipping yet is
	// unaffected).
	ShippingFlatRateCents int64
	// ShippingFreeAboveCents waives ShippingFlatRateCents once the
	// pre-discount subtotal reaches it. Zero means no free-shipping
	// threshold (shipping, if configured, always applies).
	ShippingFreeAboveCents int64
}

// PricingBreakdown is what Checkout stores on the order and what the
// storefront shows as a subtotal/discount/tax/shipping/total receipt.
type PricingBreakdown struct {
	SubtotalCents int64
	DiscountCents int64
	TaxCents      int64
	ShippingCents int64
	TotalCents    int64
}

// Price computes a full breakdown from a cart's subtotal and an
// already-validated discount amount (Service.resolveDiscount computes
// that part; Price only applies it). Discount is applied before tax, the
// ordinary rule business tax law expects, and can never push the taxable
// base below zero.
func (p PricingConfig) Price(subtotalCents, discountCents int64) PricingBreakdown {
	taxableBase := subtotalCents - discountCents
	if taxableBase < 0 {
		taxableBase = 0
	}

	var taxCents, afterTax int64
	switch {
	case p.TaxRate <= 0:
		afterTax = taxableBase
	case p.TaxInclusive:
		preTax := int64(math.Round(float64(taxableBase) / (1 + p.TaxRate)))
		taxCents = taxableBase - preTax
		afterTax = taxableBase
	default:
		taxCents = int64(math.Round(float64(taxableBase) * p.TaxRate))
		afterTax = taxableBase + taxCents
	}

	var shippingCents int64
	if p.ShippingFlatRateCents > 0 && (p.ShippingFreeAboveCents <= 0 || subtotalCents < p.ShippingFreeAboveCents) {
		shippingCents = p.ShippingFlatRateCents
	}

	return PricingBreakdown{
		SubtotalCents: subtotalCents,
		DiscountCents: discountCents,
		TaxCents:      taxCents,
		ShippingCents: shippingCents,
		TotalCents:    afterTax + shippingCents,
	}
}
