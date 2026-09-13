package graphql

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTPAcceptsReadOnlyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/graphql", strings.NewReader(`{"query":"{ __typename }"}`))
	res := httptest.NewRecorder()
	(Handler{Commerce: true}).ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"data"`) {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}

func TestSchemaOmitsDisabledModuleFields(t *testing.T) {
	for name, handler := range map[string]Handler{
		"pages-only":    {PagesEnabled: true},
		"commerce-only": {Commerce: true},
	} {
		schema, err := handler.schema()
		if err != nil {
			t.Fatalf("%s schema: %v", name, err)
		}
		fields := schema.QueryType().Fields()
		if _, ok := fields["page"]; ok != handler.PagesEnabled {
			t.Fatalf("%s page field presence=%v, enabled=%v", name, ok, handler.PagesEnabled)
		}
		if _, ok := fields["products"]; ok != handler.Commerce {
			t.Fatalf("%s products field presence=%v, enabled=%v", name, ok, handler.Commerce)
		}
	}
}

func TestServeHTTPRejectsMutationSchema(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/graphql", strings.NewReader(`{"query":"mutation { deletePage(id: \"x\") }"}`))
	res := httptest.NewRecorder()
	(Handler{PagesEnabled: true}).ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "graphql request failed") || strings.Contains(res.Body.String(), "Schema is not configured") {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}

func TestQueryBudgetRejectsAmplification(t *testing.T) {
	fields, depth, err := queryBudget(`{ page(locale: "tr", slug: "x") { title } }`)
	if err != nil || fields != 2 || depth != 2 {
		t.Fatalf("queryBudget = fields %d depth %d err %v", fields, depth, err)
	}
	tooDeep := `{ page { a { b { c { d { e { f { g { h } } } } } } } } }`
	_, depth, err = queryBudget(tooDeep)
	if err == nil && depth <= maxQueryDepth {
		t.Fatalf("expected deep query to exceed budget, depth=%d", depth)
	}
}

func TestQueryBudgetCountsFragmentSpreadsAndStopsCycles(t *testing.T) {
	fields, depth, err := queryBudget(`query { page(locale: "tr", slug: "x") { ...PageFields } } fragment PageFields on Page { title }`)
	if err != nil || fields != 2 || depth != 2 {
		t.Fatalf("fragment budget = fields %d depth %d err %v", fields, depth, err)
	}
	if fields, _, err := queryBudget(`query { page(locale: "tr", slug: "x") { ...Loop } } fragment Loop on Page { title ...Loop }`); err != nil || fields != 2 {
		t.Fatalf("cyclic fragment budget = fields %d err %v", fields, err)
	}
}

func FuzzQueryBudget(f *testing.F) {
	for _, seed := range []string{
		`{ page(locale: "tr", slug: "x") { title } }`,
		`query { page { ...Fields } } fragment Fields on Page { title }`,
		`query { page { ...Loop } } fragment Loop on Page { title ...Loop }`,
		`{`,
		``,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		fields, depth, _ := queryBudget(source)
		if fields < 0 || depth < 0 {
			t.Fatalf("negative query budget for %q: fields=%d depth=%d", source, fields, depth)
		}
	})
}
