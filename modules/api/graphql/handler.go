package graphql

import (
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
	"github.com/graphql-go/graphql"
)

// Handler exposes the read-only headless surface. Mutations remain on the
// versioned REST endpoints so their CSRF, idempotency and audit contracts are
// not bypassed by a second transport.
type Handler struct {
	DB       database.Executor
	Pages    service.ContentService
	Catalog  commerce.CatalogService
	Commerce bool
}

type request struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
	OperationName string         `json:"operationName"`
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil || input.Query == "" {
		http.Error(w, "invalid graphql request", http.StatusBadRequest)
		return
	}
	schema, err := h.schema()
	if err != nil {
		http.Error(w, "graphql unavailable", http.StatusServiceUnavailable)
		return
	}
	result := graphql.Do(graphql.Params{Schema: schema, RequestString: input.Query, VariableValues: input.Variables, OperationName: input.OperationName, Context: r.Context()})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(result.Errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (h Handler) schema() (graphql.Schema, error) {
	blockType := graphql.NewObject(graphql.ObjectConfig{Name: "PageBlock", Fields: graphql.Fields{
		"type":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"payload": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	}})
	pageType := graphql.NewObject(graphql.ObjectConfig{Name: "Page", Fields: graphql.Fields{
		"id":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"locale": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"slug":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"title":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"body":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"blocks": &graphql.Field{Type: graphql.NewList(blockType)},
	}})
	productType := graphql.NewObject(graphql.ObjectConfig{Name: "Product", Fields: graphql.Fields{
		"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"sku":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"description": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"priceCents":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"currency":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"stock":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"status":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	}})
	root := graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: graphql.Fields{
		"page": &graphql.Field{Type: pageType, Args: graphql.FieldConfigArgument{
			"locale": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"slug":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		}, Resolve: h.resolvePage},
		"products": &graphql.Field{Type: graphql.NewList(productType), Resolve: h.resolveProducts},
	}})
	return graphql.NewSchema(graphql.SchemaConfig{Query: root})
}

func (h Handler) resolvePage(p graphql.ResolveParams) (any, error) {
	page, err := h.Pages.GetPublished(p.Context, h.DB, p.Args["locale"].(string), p.Args["slug"].(string))
	if err != nil {
		return nil, err
	}
	blocks := make([]map[string]string, 0, len(page.Blocks))
	for _, block := range page.Blocks {
		blocks = append(blocks, map[string]string{"type": block.Type, "payload": string(block.Payload)})
	}
	return map[string]any{"id": page.ID, "locale": page.Locale, "slug": page.Slug, "title": page.Title, "body": page.Body, "blocks": blocks}, nil
}

func (h Handler) resolveProducts(p graphql.ResolveParams) (any, error) {
	if !h.Commerce {
		return nil, nil
	}
	products, err := h.Catalog.ListProducts(p.Context, h.DB, true)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(products))
	for _, product := range products {
		result = append(result, map[string]any{"id": product.ID, "sku": product.SKU, "title": product.Title, "description": product.Description, "priceCents": product.PriceCents, "currency": product.Currency, "stock": product.Stock, "status": product.Status})
	}
	return result, nil
}
