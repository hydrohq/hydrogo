package main

import (
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func TestToPriceConvertsTwoDecimalCatalogPricesToCents(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{{499.9, 49990}, {0.1, 10}, {19.99, 1999}, {1234.565, 123457}, {float64(10), 1000}}
	for _, tc := range cases {
		price, err := toPrice(tc.in, "TRY")
		if err != nil || price.Cents != tc.want || price.Currency != "TRY" {
			t.Errorf("toPrice(%v) = %+v, %v; want %d TRY", tc.in, price, err, tc.want)
		}
	}
	for _, bad := range []any{nil, "499.90", -1.0} {
		if _, err := toPrice(bad, "TRY"); !errors.Is(err, commerce.ErrUnknownVariant) {
			t.Errorf("toPrice(%v) must refuse to sell, got %v", bad, err)
		}
	}
}
