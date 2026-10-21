package sqlstore

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestComparisonOperatorsFilterRecords(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "filter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	service := collections.Service{Repository: Store{DB: db, Dialect: database.SQLite}}
	companies := collections.Schema{ID: "companies", Label: "Companies", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true}}}
	deals := collections.Schema{ID: "deals", Label: "Deals", Fields: []collections.Field{
		{ID: "name", Label: "Name", Kind: collections.Text, Required: true},
		{ID: "stage", Label: "Stage", Kind: collections.Select, Options: []string{"lead", "won", "lost"}},
		{ID: "amount", Label: "Amount", Kind: collections.Number},
		{ID: "closes", Label: "Closes", Kind: collections.DateTime},
		{ID: "open", Label: "Open", Kind: collections.Boolean},
		{ID: "company", Label: "Company", Kind: collections.Relation, Relation: "companies"},
	}}
	for _, schema := range []collections.Schema{companies, deals} {
		if err := service.CreateSchema(ctx, schema); err != nil {
			t.Fatal(err)
		}
	}
	acme, err := service.CreateRecord(ctx, "companies", map[string]any{"name": "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	globex, err := service.CreateRecord(ctx, "companies", map[string]any{"name": "Globex"})
	if err != nil {
		t.Fatal(err)
	}
	for _, deal := range []map[string]any{
		{"name": "a", "stage": "lead", "amount": 100, "closes": "2026-10-01T00:00:00Z", "open": true, "company": acme.ID},
		{"name": "b", "stage": "won", "amount": 250, "closes": "2026-11-15T00:00:00Z", "open": false, "company": acme.ID},
		{"name": "c", "stage": "lost", "amount": 40, "closes": "2026-12-31T00:00:00Z", "open": false, "company": globex.ID},
		{"name": "d"},
	} {
		if _, err := service.CreateRecord(ctx, "deals", deal); err != nil {
			t.Fatal(err)
		}
	}
	names := func(where ...collections.Condition) string {
		t.Helper()
		records, err := service.QueryRecords(ctx, "deals", collections.Query{Limit: 50, Where: where, Sort: &collections.Sort{Field: "name"}})
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, record := range records {
			out = append(out, record.Data["name"].(string))
		}
		return strings.Join(out, "")
	}
	for _, test := range []struct {
		name  string
		where []collections.Condition
		want  string
	}{
		{"relation equals", []collections.Condition{{Field: "company", Op: collections.Equals, Value: acme.ID}}, "ab"},
		{"relation not equals keeps unset", []collections.Condition{{Field: "company", Op: collections.NotEquals, Value: acme.ID}}, "cd"},
		{"boolean equals", []collections.Condition{{Field: "open", Op: collections.Equals, Value: "false"}}, "bc"},
		{"number gte", []collections.Condition{{Field: "amount", Op: collections.GreaterOrEqual, Value: "100"}}, "ab"},
		{"number range", []collections.Condition{{Field: "amount", Op: collections.GreaterThan, Value: "40"}, {Field: "amount", Op: collections.LessThan, Value: "250"}}, "a"},
		{"date lt", []collections.Condition{{Field: "closes", Op: collections.LessThan, Value: "2026-11-01"}}, "a"},
		{"date range", []collections.Condition{{Field: "closes", Op: collections.GreaterOrEqual, Value: "2026-11-01"}, {Field: "closes", Op: collections.LessOrEqual, Value: "2026-12-31"}}, "bc"},
		{"in list", []collections.Condition{{Field: "stage", Op: collections.In, Value: "lead,lost"}}, "ac"},
		{"in numbers", []collections.Condition{{Field: "amount", Op: collections.In, Value: "40,250"}}, "bc"},
		{"is null", []collections.Condition{{Field: "stage", Op: collections.IsNull, Value: "true"}}, "d"},
		{"is not null", []collections.Condition{{Field: "stage", Op: collections.IsNull, Value: "false"}}, "abc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			checked, err := deals.ValidateQuery(collections.Query{Limit: 50, Where: test.where})
			if err != nil {
				t.Fatal(err)
			}
			if got := names(checked.Where...); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestComparisonOperatorsKeepValuesOutOfSQL(t *testing.T) {
	query := collections.Query{Limit: 10, Where: []collections.Condition{
		{Field: "amount", Op: collections.GreaterOrEqual, Typed: 100.0, Kind: collections.Number},
		{Field: "stage", Op: collections.In, Typed: []any{"a') OR 1=1 --", "b"}, Kind: collections.Select},
		{Field: "company", Op: collections.NotEquals, Typed: "c1", Kind: collections.Relation},
		{Field: "stage", Op: collections.IsNull, Typed: true, Kind: collections.Select},
	}}
	want := map[database.Dialect][]string{
		database.SQLite:   {"json_extract(data_json, ?) >= ?", "json_extract(data_json, ?) IN (?,?)", "(json_extract(data_json, ?) IS NULL OR json_extract(data_json, ?) <> ?)", "json_extract(data_json, ?) IS NULL"},
		database.Postgres: {"(data_json::jsonb -> $3::text) >= $4::jsonb", "(data_json::jsonb ->> $5::text) IN ($6,$7)", "IS NULL OR (data_json::jsonb ->> $9::text) <> $10)"},
		database.MySQL:    {"JSON_EXTRACT(data_json, ?) >= CAST(? AS JSON)", "JSON_UNQUOTE(JSON_EXTRACT(data_json, ?)) IN (?,?)", "IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(data_json, ?)) <> ?)"},
	}
	for dialect, fragments := range want {
		statement, args, err := buildRecordQuery(dialect, "t", "deals", query)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(statement, fragment) {
				t.Errorf("%s: %q missing from\n%s", dialect, fragment, statement)
			}
		}
		if strings.Contains(statement, "OR 1=1") || strings.Contains(statement, "amount") || strings.Contains(statement, "stage") {
			t.Errorf("%s: value or field name leaked into SQL: %s", dialect, statement)
		}
		found := false
		for _, arg := range args {
			if arg == "a') OR 1=1 --" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: list value is not a bound argument: %v", dialect, args)
		}
	}
}
