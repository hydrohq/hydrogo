package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// catalogPrices prices cart lines from the catalog content types
// (content-types/commerce): a variant's price_override, else its product's
// base_price. An id that is a product (a product sold without variants)
// prices at its own base_price. Unpublished products are not for sale.
type catalogPrices struct {
	records func() collections.Service
}

func (p catalogPrices) UnitPrice(ctx context.Context, id string) (commerce.Price, error) {
	service := p.records()
	if variant, err := service.GetRecord(ctx, "variants", id); err == nil {
		productID, _ := variant.Data["product"].(string)
		product, err := p.saleableProduct(ctx, service, productID)
		if err != nil {
			return commerce.Price{}, err
		}
		if override, ok := variant.Data["price_override"]; ok && override != nil {
			return toPrice(override, product.Data["currency"])
		}
		return toPrice(product.Data["base_price"], product.Data["currency"])
	} else if !errors.Is(err, collections.ErrRecordNotFound) {
		return commerce.Price{}, err
	}
	product, err := p.saleableProduct(ctx, service, id)
	if err != nil {
		return commerce.Price{}, err
	}
	return toPrice(product.Data["base_price"], product.Data["currency"])
}

func (p catalogPrices) saleableProduct(ctx context.Context, service collections.Service, id string) (collections.Record, error) {
	if id == "" {
		return collections.Record{}, commerce.ErrUnknownVariant
	}
	product, err := service.GetRecord(ctx, "products", id)
	if errors.Is(err, collections.ErrRecordNotFound) {
		return collections.Record{}, commerce.ErrUnknownVariant
	}
	if err != nil {
		return collections.Record{}, err
	}
	if status, ok := product.Data[contenttypes.StatusField].(string); ok && status != contenttypes.StatusPublished {
		return collections.Record{}, commerce.ErrUnknownVariant
	}
	return product, nil
}

// toPrice converts a two-decimal catalog number (499.90) to integer cents.
func toPrice(amount any, currency any) (commerce.Price, error) {
	var value float64
	switch v := amount.(type) {
	case float64:
		value = v
	case int64:
		value = float64(v)
	case int:
		value = float64(v)
	default:
		return commerce.Price{}, fmt.Errorf("%w: catalog price is missing", commerce.ErrUnknownVariant)
	}
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return commerce.Price{}, fmt.Errorf("%w: catalog price is invalid", commerce.ErrUnknownVariant)
	}
	code, _ := currency.(string)
	return commerce.Price{Cents: int64(math.Round(value * 100)), Currency: strings.TrimSpace(code)}, nil
}
