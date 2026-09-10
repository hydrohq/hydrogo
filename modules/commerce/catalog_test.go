package commerce

import "testing"

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
