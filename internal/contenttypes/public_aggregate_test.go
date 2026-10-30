package contenttypes

import (
	"net/http"
	"testing"
)

const publicProductsYAML = `id: products
label: Products
public: true
fields:
  - {id: category, label: Category, kind: text, filterable: true}
  - {id: price, label: Price, kind: number, decimals: 2}
  - {id: in_stock, label: In Stock, kind: boolean, filterable: true}
`

func TestPublicAPIAggregationsAndGroupBy(t *testing.T) {
	service, types := newServiceFor(t, publicProductsYAML)
	adminBase := AdminAPIPath + "/products"
	adminHandler := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})

	for _, body := range []string{
		`{"category":"hardware","price":100.50,"in_stock":true}`,
		`{"category":"hardware","price":200.00,"in_stock":true}`,
		`{"category":"hardware","price":50.00,"in_stock":false}`,
		`{"category":"software","price":80.00,"in_stock":true}`,
	} {
		if r, _ := call(t, adminHandler, http.MethodPost, adminBase, body); r.Code != http.StatusCreated {
			t.Fatalf("seed: %d", r.Code)
		}
	}

	pubHandler := baasPublicMux(PublicAPI{Service: service, Types: types})

	// Call GET /api/v1/content/products?aggregate=count,sum(price),avg(price)&group_by=category
	r, res := call(t, pubHandler, http.MethodGet, "/api/v1/content/products?aggregate=count,sum(price),avg(price)&group_by=category", "")
	if r.Code != http.StatusOK {
		t.Fatalf("aggregate status: %d res: %v", r.Code, res)
	}

	dataMap, _ := res["data"].(map[string]any)
	items, _ := dataMap["items"].([]any)
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}
	aggs, _ := dataMap["aggregates"].([]any)
	if len(aggs) != 2 {
		t.Fatalf("expected 2 aggregate groups, got %d", len(aggs))
	}

	hw, _ := aggs[0].(map[string]any)
	grp, _ := hw["group"].(map[string]any)
	if grp["category"] != "hardware" {
		t.Errorf("expected hardware first, got %v", grp["category"])
	}
	if count, _ := hw["count"].(float64); count != 3 {
		t.Errorf("hardware count: want 3, got %v", count)
	}
	if sum, _ := hw["sum_price"].(float64); sum != 350.5 {
		t.Errorf("hardware sum_price: want 350.5, got %v", sum)
	}

	// Filtered aggregate: in_stock = true
	rFlt, resFlt := call(t, pubHandler, http.MethodGet, "/api/v1/content/products?filter[in_stock]=true&aggregate=count,sum(price)", "")
	if rFlt.Code != http.StatusOK {
		t.Fatalf("filter agg status: %d res: %v", rFlt.Code, resFlt)
	}
	dataFlt, _ := resFlt["data"].(map[string]any)
	aggsFlt, _ := dataFlt["aggregates"].([]any)
	if len(aggsFlt) != 1 {
		t.Fatalf("expected 1 aggregate result, got %d", len(aggsFlt))
	}
	firstFlt, _ := aggsFlt[0].(map[string]any)
	if count, _ := firstFlt["count"].(float64); count != 3 {
		t.Errorf("filtered count: want 3, got %v", count)
	}
	if sum, _ := firstFlt["sum_price"].(float64); sum != 380.5 {
		t.Errorf("filtered sum_price: want 380.5, got %v", sum)
	}
}
