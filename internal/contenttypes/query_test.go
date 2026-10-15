package contenttypes

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestParsePublicQueryOnlyFiltersFilterableFields(t *testing.T) {
	definition := Definition{ID: "faq", Fields: []Field{
		{ID: "question", Kind: "text", Searchable: true},
		{ID: "category", Kind: "select", Filterable: true, Options: []string{"a"}},
		{ID: "secret", Kind: "text"},
	}}
	values, _ := url.ParseQuery("filter[category]=a&search=ship&sort=-created_at&utm_source=x")
	query, err := parsePublicQuery(values, definition)
	if err != nil || len(query.Where) != 1 || len(query.AnyOf) != 1 || query.Sort.Field != "created_at" {
		t.Fatalf("query=%+v err=%v", query, err)
	}
	for _, raw := range []string{"filter[secret]=x", "filter[missing]=x", "filter[category]=a&filter[category]=b", "limit=500"} {
		values, _ := url.ParseQuery(raw)
		if _, err := parsePublicQuery(values, definition); !errors.Is(err, collections.ErrInvalidQuery) {
			t.Errorf("%s: err=%v", raw, err)
		}
	}
}

const searchableFAQ = `id: faq
label: FAQ
title_field: question
public: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
  - {id: answer, label: Answer, kind: richtext, searchable: true}
  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
  - {id: sort_order, label: Order, kind: number, list: true, filterable: true}
`

func newServiceFor(t *testing.T, yamlFiles ...string) (collections.Service, []Definition) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "content.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for index, content := range yamlFiles {
		if err := os.WriteFile(filepath.Join(dir, string(rune('a'+index))+".yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	types, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := Schemas(types)
	if err != nil {
		t.Fatal(err)
	}
	service := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite}}
	if err := service.SyncSchemas(ctx, schemas); err != nil {
		t.Fatal(err)
	}
	return service, types
}

func TestPublicAPISortsAndFilters(t *testing.T) {
	service, types := newServiceFor(t, searchableFAQ)
	ctx := context.Background()
	for _, data := range []map[string]any{
		{"question": "B question", "category": "general", "sort_order": 2.0},
		{"question": "A question", "category": "billing", "sort_order": 1.0},
		{"question": "C question", "category": "general", "sort_order": 3.0},
	} {
		if _, err := service.CreateRecord(ctx, "faq", data); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	public := PublicAPI{Service: service, Types: types}
	mux.HandleFunc("GET /api/v1/content/{type}", public.List)
	questions := func(target string) []string {
		t.Helper()
		recorder, body := call(t, mux, http.MethodGet, target, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", target, recorder.Code, recorder.Body)
		}
		result := []string{}
		for _, item := range body["data"].([]any) {
			result = append(result, item.(map[string]any)["question"].(string))
		}
		return result
	}
	if got := strings.Join(questions("/api/v1/content/faq?sort=sort_order"), ","); got != "A question,B question,C question" {
		t.Fatalf("sort: %s", got)
	}
	if got := strings.Join(questions("/api/v1/content/faq?sort=-question&filter[category]=general"), ","); got != "C question,B question" {
		t.Fatalf("filter+sort: %s", got)
	}
	if got := strings.Join(questions("/api/v1/content/faq?search=a+quest"), ","); got != "A question" {
		t.Fatalf("search: %s", got)
	}
	for _, target := range []string{"/api/v1/content/faq?filter[question]=A+question", "/api/v1/content/faq?sort=nope", "/api/v1/content/faq?sort=question%27%3B--"} {
		if recorder, _ := call(t, mux, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", target, recorder.Code)
		}
	}
}
