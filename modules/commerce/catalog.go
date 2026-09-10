package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrInvalidProduct = errors.New("invalid product")

type CatalogStore interface {
	CreateProduct(context.Context, database.Executor, Product) error
	ListProducts(context.Context, database.Executor, bool) ([]Product, error)
}

type CatalogService struct {
	Transactions database.Transactor
	Store        CatalogStore
}

func ValidateProduct(product Product) error {
	if strings.TrimSpace(product.ID) == "" || strings.TrimSpace(product.SKU) == "" || strings.TrimSpace(product.Title) == "" || product.PriceCents < 0 || product.Stock < 0 {
		return ErrInvalidProduct
	}
	currency := strings.ToUpper(strings.TrimSpace(product.Currency))
	if len(currency) != 3 || product.Status == "" {
		return ErrInvalidProduct
	}
	if product.Status != ProductDraft && product.Status != ProductPublished && product.Status != ProductOutOfStock {
		return ErrInvalidProduct
	}
	return nil
}

func (s CatalogService) CreateProduct(ctx context.Context, product Product) error {
	if err := ValidateProduct(product); err != nil {
		return err
	}
	if s.Store == nil {
		return errors.New("commerce catalog dependency is required")
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.CreateProduct(ctx, tx, product) })
}

func (s CatalogService) ListProducts(ctx context.Context, db database.Executor, publicOnly bool) ([]Product, error) {
	if s.Store == nil {
		return nil, errors.New("commerce catalog dependency is required")
	}
	return s.Store.ListProducts(ctx, db, publicOnly)
}
