package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

var ErrPhysicalSchemaDiffUnsupported = errors.New("physical schema inspection is not certified for this database dialect")

type SchemaChange struct {
	Kind   string `json:"kind"`
	Object string `json:"object"`
	Name   string `json:"name"`
}

// SQLiteSchemaDiff compares the live SQLite catalog with a temporary schema
// built from the selected migration bundle. The live database is read-only.
func SQLiteSchemaDiff(ctx context.Context, live *sql.DB, migrations fs.FS, enabled map[string]bool) ([]SchemaChange, error) {
	if live == nil {
		return nil, ErrDatabaseRequired
	}
	reference, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	reference.SetMaxOpenConns(1)
	defer reference.Close()
	if err := ApplyMigrationsForModules(ctx, reference, SQLite, migrations, enabled); err != nil {
		return nil, fmt.Errorf("build migration reference schema: %w", err)
	}
	want, err := sqliteCatalog(ctx, reference)
	if err != nil {
		return nil, err
	}
	got, err := sqliteCatalog(ctx, live)
	if err != nil {
		return nil, err
	}
	changes := make([]SchemaChange, 0)
	for key, expected := range want {
		actual, ok := got[key]
		kind, name, _ := strings.Cut(key, ":")
		if !ok {
			changes = append(changes, SchemaChange{Kind: "missing", Object: kind, Name: name})
		} else if normalizeSchemaSQL(actual) != normalizeSchemaSQL(expected) {
			changes = append(changes, SchemaChange{Kind: "definition_mismatch", Object: kind, Name: name})
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			kind, name, _ := strings.Cut(key, ":")
			changes = append(changes, SchemaChange{Kind: "unexpected", Object: kind, Name: name})
		}
	}
	sortSchemaChanges(changes)
	return changes, nil
}

func sqliteCatalog(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT type,name,COALESCE(sql,'') FROM sqlite_master WHERE type IN ('table','index','view','trigger') AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("inspect SQLite schema catalog: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var kind, name, definition string
		if err := rows.Scan(&kind, &name, &definition); err != nil {
			return nil, err
		}
		result[kind+":"+name] = definition
	}
	return result, rows.Err()
}

func normalizeSchemaSQL(value string) string { return strings.Join(strings.Fields(value), " ") }

func sortSchemaChanges(changes []SchemaChange) {
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Object == changes[j].Object {
			return changes[i].Name < changes[j].Name
		}
		return changes[i].Object < changes[j].Object
	})
}
