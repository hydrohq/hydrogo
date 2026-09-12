package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/content"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrInvalidProduct = errors.New("invalid product")

type CatalogStore interface {
	CreateProduct(context.Context, database.Executor, Product) error
	UpdateProduct(context.Context, database.Executor, Product) error
	GetProduct(context.Context, database.Executor, string) (Product, error)
	ListProducts(context.Context, database.Executor, bool) ([]Product, error)
}

type VariantStore interface {
	CreateVariant(context.Context, database.Executor, ProductVariant) error
	ListVariants(context.Context, database.Executor, string, bool) ([]ProductVariant, error)
}

type OrderReader interface {
	ListOrders(context.Context, database.Executor, int) ([]Order, error)
	GetOrder(context.Context, database.Executor, string) (Order, error)
}

// OrderReaderByUserID narrows a private customer projection at the repository
// boundary instead of loading unrelated customer records into application
// memory.
type OrderReaderByUserID interface {
	ListOrdersByUserID(context.Context, database.Executor, string, int) ([]Order, error)
}

type CatalogService struct {
	Transactions database.Transactor
	Store        CatalogStore
	Variants     VariantStore
}

func ValidateVariant(variant ProductVariant) error {
	if strings.TrimSpace(variant.ID) == "" || strings.TrimSpace(variant.ProductID) == "" || strings.TrimSpace(variant.SKU) == "" || strings.TrimSpace(variant.Title) == "" || variant.PriceCents < 0 || variant.Stock < 0 {
		return ErrInvalidProduct
	}
	if variant.Status != ProductDraft && variant.Status != ProductPublished && variant.Status != ProductOutOfStock {
		return ErrInvalidProduct
	}
	return nil
}

func (s CatalogService) CreateVariant(ctx context.Context, variant ProductVariant) error {
	if err := ValidateVariant(variant); err != nil {
		return err
	}
	if s.Variants == nil {
		return errors.New("commerce variant dependency is required")
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Variants.CreateVariant(ctx, tx, variant) })
}

func (s CatalogService) ListVariants(ctx context.Context, db database.Executor, productID string, publicOnly bool) ([]ProductVariant, error) {
	if s.Variants == nil {
		return nil, errors.New("commerce variant dependency is required")
	}
	return s.Variants.ListVariants(ctx, db, productID, publicOnly)
}

func ValidateProduct(product Product) error {
	if err := content.ValidateRichText(product.Description); err != nil {
		return ErrInvalidProduct
	}
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

func (s CatalogService) UpdateProduct(ctx context.Context, product Product) error {
	if err := ValidateProduct(product); err != nil {
		return err
	}
	if s.Store == nil {
		return errors.New("commerce catalog dependency is required")
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.UpdateProduct(ctx, tx, product) })
}

func (s CatalogService) GetProduct(ctx context.Context, db database.Executor, productID string) (Product, error) {
	if s.Store == nil {
		return Product{}, errors.New("commerce catalog dependency is required")
	}
	return s.Store.GetProduct(ctx, db, productID)
}

// GetPublishedProduct is the public catalog projection. Administrative callers
// use GetProduct; public handlers must never expose a draft record merely
// because its opaque ID is known.
func (s CatalogService) GetPublishedProduct(ctx context.Context, db database.Executor, productID string) (Product, error) {
	product, err := s.GetProduct(ctx, db, productID)
	if err != nil {
		return Product{}, err
	}
	if product.Status != ProductPublished && product.Status != ProductOutOfStock {
		return Product{}, errors.New("published product not found")
	}
	return product, nil
}

func (s CatalogService) ListProducts(ctx context.Context, db database.Executor, publicOnly bool) ([]Product, error) {
	if s.Store == nil {
		return nil, errors.New("commerce catalog dependency is required")
	}
	return s.Store.ListProducts(ctx, db, publicOnly)
}
