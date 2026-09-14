package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestPublicListIndexesAreChosen protects the read-path contract from silent
// full scans as list endpoints grow. It deliberately uses SQLite's planner;
// PostgreSQL execution remains part of the external release gate.
func TestPublicListIndexesAreChosen(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE pages (id TEXT, published_locale TEXT, updated_at TEXT, published_revision INTEGER, status TEXT)`,
		`CREATE TABLE news_items (id TEXT, locale TEXT, updated_at TEXT, status TEXT, published_revision INTEGER)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, migration := range []string{"202609140001_pages_list_indexes", "202609140002_news_list_indexes"} {
		sqlPath := filepath.Join("..", "..", "migrations", migration, "sqlite.sql")
		contents, err := os.ReadFile(sqlPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name, query, index string
	}{
		{"pages public", `EXPLAIN QUERY PLAN SELECT id FROM pages WHERE published_locale = 'tr' AND published_revision IS NOT NULL ORDER BY updated_at DESC, id ASC LIMIT 20`, "idx_pages_public_list"},
		{"news public", `EXPLAIN QUERY PLAN SELECT id FROM news_items WHERE locale = 'tr' AND status = 'published' AND published_revision IS NOT NULL ORDER BY updated_at DESC, id ASC LIMIT 20`, "idx_news_public_list"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := db.QueryContext(context.Background(), tc.query)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			var used bool
			for rows.Next() {
				var detail string
				var id, parent, notused int
				if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
					t.Fatal(err)
				}
				used = used || strings.Contains(detail, tc.index)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if !used {
				t.Fatalf("planner did not select %s", tc.index)
			}
		})
	}
}
