package sqlstore

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteExpressionIndexCreationAndExplainPlan(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "indexes_test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	store := Store{DB: db, Dialect: database.SQLite}
	colID := "articles"
	_, err = db.ExecContext(ctx, `INSERT INTO hydro_collections (tenant_id, id, label, schema_json, version, created_at, updated_at) VALUES ('', ?, 'Articles', '{}', 1, '2026-01-01', '2026-01-01')`, colID)
	if err != nil {
		t.Fatal(err)
	}
	schema := collections.Schema{
		ID:    colID,
		Label: "Articles",
		Fields: []collections.Field{
			{ID: "title", Label: "Title", Kind: collections.Text},
			{ID: "category", Label: "Category", Kind: collections.Text, Indexed: true},
			{ID: "views", Label: "Views", Kind: collections.Number, Indexed: true},
		},
	}

	if err := store.SyncIndexes(ctx, []collections.Schema{schema}); err != nil {
		t.Fatalf("SyncIndexes: %v", err)
	}

	indexes, err := store.managedIndexes(ctx)
	if err != nil {
		t.Fatalf("managedIndexes: %v", err)
	}
	if len(indexes) != 2 {
		t.Fatalf("expected 2 managed indexes, got %d (%v)", len(indexes), indexes)
	}

	// Insert records
	for i := 0; i < 20; i++ {
		data := fmt.Sprintf(`{"title": "Art %d", "category": "tech", "views": %d}`, i, i*10)
		_, err := db.ExecContext(ctx, `INSERT INTO hydro_collection_records (tenant_id, collection_id, id, data_json, version, created_at, updated_at) VALUES ('', 'articles', ?, ?, 1, '2026-01-01', '2026-01-01')`, fmt.Sprintf("art_%d", i), data)
		if err != nil {
			t.Fatalf("insert record: %v", err)
		}
	}

	// Build query on indexed field 'category'
	q := collections.Query{
		Limit: 10,
		Where: []collections.Condition{
			{Field: "category", Op: collections.Equals, Typed: "tech"},
		},
	}
	sqlStr, args, err := buildRecordQuery(database.SQLite, "", colID, q)
	if err != nil {
		t.Fatalf("buildRecordQuery: %v", err)
	}

	// Run EXPLAIN QUERY PLAN
	rows, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+sqlStr, args...)
	if err != nil {
		t.Fatalf("explain query plan: %v", err)
	}
	defer rows.Close()

	var planLines []string
	usedIndex := false
	for rows.Next() {
		var a, b, c int
		var detail string
		if err := rows.Scan(&a, &b, &c, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		planLines = append(planLines, detail)
		if strings.Contains(detail, "USING INDEX idx_hcr_") {
			usedIndex = true
		}
	}
	if !usedIndex {
		t.Fatalf("expected SQLite to use expression index idx_hcr_..., but got plan:\n%s", strings.Join(planLines, "\n"))
	}

	// Now remove index on category and sync again
	schema.Fields[1].Indexed = false
	if err := store.SyncIndexes(ctx, []collections.Schema{schema}); err != nil {
		t.Fatalf("SyncIndexes prune: %v", err)
	}

	indexesAfter, err := store.managedIndexes(ctx)
	if err != nil {
		t.Fatalf("managedIndexes after prune: %v", err)
	}
	if len(indexesAfter) != 1 {
		t.Fatalf("expected 1 managed index after prune, got %d (%v)", len(indexesAfter), indexesAfter)
	}
}

func TestIndexStatementGenerationDialects(t *testing.T) {
	field := collections.Field{ID: "slug", Kind: collections.Text, Indexed: true}

	pgDefs, err := indexStatements(database.Postgres, "posts", field, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(pgDefs) != 2 { // btree + trgm
		t.Fatalf("expected 2 pg defs, got %d", len(pgDefs))
	}
	if !strings.Contains(pgDefs[0].statement, "USING gin") && !strings.Contains(pgDefs[1].statement, "USING gin") {
		t.Errorf("expected gin trgm index in pg defs")
	}

	sqliteDefs, err := indexStatements(database.SQLite, "posts", field, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sqliteDefs) != 1 {
		t.Fatalf("expected 1 sqlite def, got %d", len(sqliteDefs))
	}
	if !strings.Contains(sqliteDefs[0].statement, "json_extract(data_json, '$.slug')") {
		t.Errorf("expected json_extract in sqlite def: %s", sqliteDefs[0].statement)
	}

	mysqlDefs, err := indexStatements(database.MySQL, "posts", field, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(mysqlDefs) != 1 {
		t.Fatalf("expected 1 mysql def, got %d", len(mysqlDefs))
	}
	if !strings.Contains(mysqlDefs[0].statement, "CAST(JSON_UNQUOTE(JSON_EXTRACT(data_json, '$.slug')) AS CHAR(255))") {
		t.Errorf("expected cast in mysql def: %s", mysqlDefs[0].statement)
	}
}
