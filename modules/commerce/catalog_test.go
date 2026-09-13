package commerce

import (
	"context"
	"math"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type catalogStoreStub struct{ product Product }

func (s catalogStoreStub) CreateProduct(context.Context, database.Executor, Product) error {
	return nil
}
func (s catalogStoreStub) UpdateProduct(context.Context, database.Executor, Product) error {
	return nil
}
func (s catalogStoreStub) GetProduct(context.Context, database.Executor, string) (Product, error) {
	return s.product, nil
}
func (s catalogStoreStub) ListProducts(context.Context, database.Executor, bool) ([]Product, error) {
	return nil, nil
}
func (s catalogStoreStub) ListProductsPage(context.Context, database.Executor, bool, int, int) ([]Product, error) {
	return nil, nil
}

func TestValidateProduct(t *testing.T) {
	valid := Product{ID: "p-1", SKU: "sku-1", Title: "Book", PriceCents: 100, Currency: "TRY", Stock: 2, Status: ProductPublished}
	if err := ValidateProduct(valid); err != nil {
		t.Fatal(err)
	}
	valid.Currency = "EURO"
	if err := ValidateProduct(valid); err == nil {
		t.Fatal("invalid currency accepted")
	}
}

func TestCheckedCentsArithmeticRejectsOverflow(t *testing.T) {
	if _, ok := checkedCentsTotal(math.MaxInt64, 2); ok {
		t.Fatal("multiplication overflow accepted")
	}
	if _, ok := checkedCentsAdd(math.MaxInt64, 1); ok {
		t.Fatal("addition overflow accepted")
	}
	if total, ok := checkedCentsTotal(1250, 2); !ok || total != 2500 {
		t.Fatalf("valid cents multiplication = %d/%v", total, ok)
	}
}

func TestGetPublishedProductDoesNotReturnDraft(t *testing.T) {
	service := CatalogService{Store: catalogStoreStub{product: Product{ID: "draft", Status: ProductDraft}}}
	if _, err := service.GetPublishedProduct(context.Background(), nil, "draft"); err == nil {
		t.Fatal("draft product was returned through public catalog")
	}
}
