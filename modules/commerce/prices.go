package commerce

import (
	"context"
	"errors"
)

// Price is what a variant sells for right now, in the smallest unit of
// Currency. Currency may be empty when the catalog does not record one; the
// order then takes the cart's currency.
type Price struct {
	Cents    int64
	Currency string
}

// PriceSource is the only place a unit price may come from. The cart and
// checkout never trust a price sent by the client: a shopper who posts
// unit_price_cents=1 must still be charged the catalog price. cmd/hydro
// implements this against the catalog content types (variant
// price_override, else product base_price); tests use StaticPrices.
type PriceSource interface {
	UnitPrice(ctx context.Context, variantID string) (Price, error)
}

var (
	// ErrUnknownVariant means the id is not something this store sells
	// (no such variant/product, or an unpublished draft).
	ErrUnknownVariant = errors.New("this item is not for sale")
	// ErrNoPriceSource means Service.Prices was never configured. Checkout
	// refuses to guess a price.
	ErrNoPriceSource = errors.New("no price source is configured")
	// ErrCurrencyMismatch means an item is priced in a currency other than
	// the cart's; charging it would silently convert at 1:1.
	ErrCurrencyMismatch = errors.New("item is priced in a different currency than the cart")
)

// StaticPrices is a fixed price list, for tests and catalogs that live
// outside the content-type engine.
type StaticPrices map[string]Price

func (p StaticPrices) UnitPrice(_ context.Context, variantID string) (Price, error) {
	price, ok := p[variantID]
	if !ok {
		return Price{}, ErrUnknownVariant
	}
	return price, nil
}

func (s Service) unitPrice(ctx context.Context, variantID, cartCurrency string) (int64, error) {
	if s.Prices == nil {
		return 0, ErrNoPriceSource
	}
	price, err := s.Prices.UnitPrice(ctx, variantID)
	if err != nil {
		return 0, err
	}
	if price.Cents < 0 {
		return 0, ErrUnknownVariant
	}
	if price.Currency != "" && cartCurrency != "" && price.Currency != cartCurrency {
		return 0, ErrCurrencyMismatch
	}
	return price.Cents, nil
}
