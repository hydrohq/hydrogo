package sqlstore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestBuildRecordQueryKeepsFieldNamesOutOfSQL(t *testing.T) {
	query := collections.Query{Limit: 10, Offset: 20,
		Where: []collections.Condition{{Field: "sort_order", Op: collections.Equals, Typed: 3.0, Kind: collections.Number}},
		AnyOf: []collections.Condition{
			{Field: "question", Op: collections.Contains, Typed: "50%_off!", Kind: collections.Text},
			{Field: "answer", Op: collections.Contains, Typed: "Ship", Kind: collections.RichText},
		},
		Sort: &collections.Sort{Field: "question", Desc: true, Kind: collections.Text},
	}
	want := map[database.Dialect][]string{
		database.SQLite:   {"json_extract(data_json, '$.sort_order') = ?", "LOWER(json_extract(data_json, '$.question')) LIKE ? ESCAPE '!'", "ORDER BY LOWER(json_extract(data_json, '$.question')) DESC,id ASC LIMIT ? OFFSET ?"},
		database.Postgres: {"(data_json::jsonb -> 'sort_order') = $3::jsonb", "LOWER((data_json::jsonb ->> 'question')) LIKE $4 ESCAPE '!'", "ORDER BY LOWER((data_json::jsonb ->> 'question')) DESC,id ASC LIMIT $6 OFFSET $7"},
		database.MySQL:    {"JSON_EXTRACT(data_json, '$.sort_order') = CAST(? AS JSON)", "LOWER(JSON_UNQUOTE(JSON_EXTRACT(data_json, '$.question'))) LIKE ? ESCAPE '!'", "LIMIT ? OFFSET ?"},
	}
	for dialect, fragments := range want {
		statement, args, err := buildRecordQuery(dialect, "tenant", "faq", query)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(statement, fragment) {
				t.Errorf("%s: %q missing from\n%s", dialect, fragment, statement)
			}
		}
		if len(args) != 7 || args[0] != "tenant" || args[1] != "faq" || args[5] != 10 || args[6] != 20 {
			t.Errorf("%s: args=%v", dialect, args)
		}
		pattern := args[3]
		if pattern != "%50!%!_off!!%" {
			t.Errorf("%s: LIKE pattern=%v", dialect, pattern)
		}
	}
	statement, args, err := buildRecordQuery(database.SQLite, "", "faq", collections.Query{Limit: 5})
	if err != nil || !strings.HasSuffix(statement, "ORDER BY updated_at DESC,id ASC LIMIT ? OFFSET ?") || len(args) != 4 {
		t.Fatalf("default order: %s %v %v", statement, args, err)
	}
	statement, _, err = buildRecordQuery(database.Postgres, "", "faq", collections.Query{Limit: 5, Sort: &collections.Sort{Field: collections.SortCreatedAt}})
	if err != nil || !strings.Contains(statement, "ORDER BY created_at ASC,id ASC") {
		t.Fatalf("system sort: %s %v", statement, err)
	}
}

// The store must refuse field IDs that are not plain identifiers even if a
// caller skips Schema.ValidateQuery.
func TestBuildRecordQueryRejectsUnvalidatedFieldNames(t *testing.T) {
	for _, field := range []string{"", "a') OR 1=1 --", "a.b", "$.x", "A", "a b", "a\"", strings.Repeat("a", 64)} {
		queries := []collections.Query{
			{Limit: 1, Where: []collections.Condition{{Field: field, Op: collections.Equals, Typed: "x", Kind: collections.Text}}},
			{Limit: 1, AnyOf: []collections.Condition{{Field: field, Op: collections.Contains, Typed: "x", Kind: collections.Text}}},
			{Limit: 1, Sort: &collections.Sort{Field: field, Kind: collections.Text}},
		}
		for _, query := range queries {
			for _, dialect := range []database.Dialect{database.SQLite, database.Postgres, database.MySQL} {
				if _, _, err := buildRecordQuery(dialect, "", "c", query); !errors.Is(err, collections.ErrInvalidQuery) {
					t.Fatalf("%s field %q: err=%v", dialect, field, err)
				}
			}
		}
	}
	untyped := collections.Query{Limit: 1, Where: []collections.Condition{{Field: "a", Op: collections.Equals, Value: "x"}}}
	if _, _, err := buildRecordQuery(database.SQLite, "", "c", untyped); !errors.Is(err, collections.ErrInvalidQuery) {
		t.Fatalf("unvalidated condition: %v", err)
	}
}

func runQueryContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	schema := collections.Schema{ID: "query_faq", Label: "FAQ", Fields: []collections.Field{
		{ID: "question", Label: "Question", Kind: collections.Text, Required: true},
		{ID: "category", Label: "Category", Kind: collections.Select, Options: []string{"general", "billing"}},
		{ID: "sort_order", Label: "Order", Kind: collections.Number},
		{ID: "featured", Label: "Featured", Kind: collections.Boolean},
	}}
	if err := service.CreateSchema(ctx, schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.DeleteSchema(ctx, schema.ID) }()
	for _, data := range []map[string]any{
		{"question": "How long does Shipping take?", "category": "general", "sort_order": 2.0, "featured": true},
		{"question": "Can I pay by invoice?", "category": "billing", "sort_order": 10.0, "featured": false},
		{"question": "Is there a 50% discount?", "category": "billing", "sort_order": 1.0},
		{"question": "What about O'Brien's order?", "category": "general", "sort_order": 3.0},
	} {
		if _, err := service.CreateRecord(ctx, schema.ID, data); err != nil {
			t.Fatal(err)
		}
	}
	questions := func(query collections.Query) []string {
		t.Helper()
		query.Limit = 50
		records, err := service.QueryRecords(ctx, schema.ID, query)
		if err != nil {
			t.Fatalf("%s: query %+v: %v", dialect, query, err)
		}
		result := make([]string, 0, len(records))
		for _, record := range records {
			result = append(result, record.Data["question"].(string))
		}
		return result
	}
	expect := func(name string, got []string, want ...string) {
		t.Helper()
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("%s %s: got %q, want %q", dialect, name, got, want)
		}
	}
	byOrder := &collections.Sort{Field: "sort_order"}
	expect("sort by number", questions(collections.Query{Sort: byOrder}),
		"Is there a 50% discount?", "How long does Shipping take?", "What about O'Brien's order?", "Can I pay by invoice?")
	expect("sort by text desc", questions(collections.Query{Sort: &collections.Sort{Field: "question", Desc: true}}),
		"What about O'Brien's order?", "Is there a 50% discount?", "How long does Shipping take?", "Can I pay by invoice?")
	expect("contains is case-insensitive", questions(collections.Query{AnyOf: []collections.Condition{{Field: "question", Op: collections.Contains, Value: "SHIPPING"}}}),
		"How long does Shipping take?")
	expect("wildcards are literal", questions(collections.Query{AnyOf: []collections.Condition{{Field: "question", Op: collections.Contains, Value: "%"}}}),
		"Is there a 50% discount?")
	expect("underscore is literal", questions(collections.Query{AnyOf: []collections.Condition{{Field: "question", Op: collections.Contains, Value: "_"}}}))
	expect("quotes are data", questions(collections.Query{AnyOf: []collections.Condition{{Field: "question", Op: collections.Contains, Value: "o'brien"}}}),
		"What about O'Brien's order?")
	expect("injection in a value is data", questions(collections.Query{Where: []collections.Condition{{Field: "category", Op: collections.Equals, Value: "general' OR '1'='1"}}}))
	expect("equals select", questions(collections.Query{Where: []collections.Condition{{Field: "category", Op: collections.Equals, Value: "billing"}}, Sort: byOrder}),
		"Is there a 50% discount?", "Can I pay by invoice?")
	expect("equals number", questions(collections.Query{Where: []collections.Condition{{Field: "sort_order", Op: collections.Equals, Value: "10"}}}),
		"Can I pay by invoice?")
	expect("equals boolean", questions(collections.Query{Where: []collections.Condition{{Field: "featured", Op: collections.Equals, Value: "true"}}}),
		"How long does Shipping take?")
	expect("search or, filter and", questions(collections.Query{
		Where: []collections.Condition{{Field: "category", Op: collections.Equals, Value: "general"}},
		AnyOf: []collections.Condition{{Field: "question", Op: collections.Contains, Value: "long"}, {Field: "category", Op: collections.Contains, Value: "bill"}},
	}), "How long does Shipping take?")
	page, err := service.QueryRecords(ctx, schema.ID, collections.Query{Limit: 2, Offset: 1, Sort: byOrder})
	if err != nil || len(page) != 2 || page[0].Data["question"] != "How long does Shipping take?" {
		t.Fatalf("%s paging: %+v %v", dialect, page, err)
	}
	if _, err := service.QueryRecords(ctx, schema.ID, collections.Query{Limit: 1, Sort: &collections.Sort{Field: "question\" DESC; --"}}); !errors.Is(err, collections.ErrInvalidQuery) {
		t.Fatalf("%s: injected sort err=%v", dialect, err)
	}
}
