package graphql

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
)

const (
	maxQueryBytes  = 256 << 10
	maxQueryFields = 100
	maxQueryDepth  = 8
)

// Handler exposes the read-only headless surface. Mutations remain on the
// versioned REST endpoints so their CSRF, idempotency and audit contracts are
// not bypassed by a second transport.
type Handler struct {
	DB           database.Executor
	Pages        service.ContentService
	PagesEnabled bool
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
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxQueryBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.Query == "" {
		http.Error(w, "invalid graphql request", http.StatusBadRequest)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		http.Error(w, "invalid graphql request", http.StatusBadRequest)
		return
	}
	if fields, depth, err := queryBudget(input.Query); err != nil || fields > maxQueryFields || depth > maxQueryDepth {
		http.Error(w, "graphql query exceeds the permitted complexity", http.StatusBadRequest)
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
		for i := range result.Errors {
			result.Errors[i].Message = "graphql request failed"
		}
		w.WriteHeader(http.StatusBadRequest)
	}
	_ = json.NewEncoder(w).Encode(result)
}

// queryBudget rejects deliberately amplified documents before schema execution.
// The limits are intentionally conservative because this endpoint is public and
// each root resolver may perform a database read.
func queryBudget(source string) (fields, depth int, err error) {
	doc, err := parser.Parse(parser.ParseParams{Source: source})
	if err != nil {
		return 0, 0, err
	}
	fragments := make(map[string]*ast.FragmentDefinition)
	for _, definition := range doc.Definitions {
		if fragment, ok := definition.(*ast.FragmentDefinition); ok && fragment.Name != nil {
			fragments[fragment.Name.Value] = fragment
		}
	}
	var walk func(*ast.SelectionSet, int, map[string]bool)
	walk = func(set *ast.SelectionSet, current int, active map[string]bool) {
		if set == nil {
			return
		}
		if current > depth {
			depth = current
		}
		for _, selection := range set.Selections {
			if field, ok := selection.(*ast.Field); ok {
				fields++
				walk(field.SelectionSet, current+1, active)
			} else if inline, ok := selection.(*ast.InlineFragment); ok {
				walk(inline.SelectionSet, current, active)
			} else if spread, ok := selection.(*ast.FragmentSpread); ok && spread.Name != nil {
				name := spread.Name.Value
				fragment := fragments[name]
				if fragment == nil || active[name] {
					continue
				}
				active[name] = true
				walk(fragment.SelectionSet, current, active)
				delete(active, name)
			}
		}
	}
	for _, definition := range doc.Definitions {
		if operation, ok := definition.(*ast.OperationDefinition); ok {
			walk(operation.SelectionSet, 1, make(map[string]bool))
		}
	}
	return fields, depth, nil
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
	fields := graphql.Fields{}
	if h.PagesEnabled {
		fields["page"] = &graphql.Field{Type: pageType, Args: graphql.FieldConfigArgument{
			"locale": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"slug":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		}, Resolve: h.resolvePage}
	}
	root := graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: fields})
	return graphql.NewSchema(graphql.SchemaConfig{Query: root})
}

func (h Handler) resolvePage(p graphql.ResolveParams) (any, error) {
	if !h.PagesEnabled {
		return nil, errors.New("pages module unavailable")
	}
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
