package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	return compareSchemaCatalogs(want, got), nil
}

// PostgresSchemaDiff compares the current schema with a migration-built
// reference in a uniquely named scratch schema. It requires CREATE/DROP
// SCHEMA permission and removes the scratch schema before returning.
func PostgresSchemaDiff(ctx context.Context, live *sql.DB, migrations fs.FS, enabled map[string]bool) (changes []SchemaChange, resultErr error) {
	if live == nil {
		return nil, ErrDatabaseRequired
	}
	ctx = nonNilContext(ctx)
	conn, err := live.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("open schema inspection connection: %w", err)
	}
	defer conn.Close()
	var originalSearchPath string
	if err := conn.QueryRowContext(ctx, "SELECT current_setting('search_path')").Scan(&originalSearchPath); err != nil {
		return nil, fmt.Errorf("read PostgreSQL schema inspection path: %w", err)
	}

	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return nil, fmt.Errorf("create schema inspection namespace: %w", err)
	}
	schema := "hydro_diff_" + hex.EncodeToString(suffix[:])
	if _, err := conn.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		return nil, fmt.Errorf("create temporary schema inspection namespace: %w", err)
	}
	defer func() {
		if _, err := conn.ExecContext(context.Background(), "SELECT set_config('search_path',$1,false)", originalSearchPath); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("reset schema inspection path: %w", err))
		}
		if _, err := conn.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove temporary schema inspection namespace: %w", err))
		}
	}()
	if _, err := conn.ExecContext(ctx, "SET search_path TO "+schema); err != nil {
		return nil, fmt.Errorf("select temporary schema inspection namespace: %w", err)
	}
	if err := applyMigrationsOnConnection(ctx, conn, Postgres, migrations, enabled, false); err != nil {
		return nil, fmt.Errorf("build migration reference schema: %w", err)
	}
	want, err := postgresCatalog(ctx, conn)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "SELECT set_config('search_path',$1,false)", originalSearchPath); err != nil {
		return nil, fmt.Errorf("restore schema inspection path: %w", err)
	}
	got, err := postgresCatalog(ctx, conn)
	if err != nil {
		return nil, err
	}
	changes = compareSchemaCatalogs(want, got)
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

func postgresCatalog(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (map[string]string, error) {
	result := make(map[string]string)
	var currentSchema string
	if err := db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&currentSchema); err != nil {
		return nil, fmt.Errorf("inspect PostgreSQL current schema: %w", err)
	}
	rows, err := db.QueryContext(ctx, `SELECT table_name,table_type FROM information_schema.tables WHERE table_schema=current_schema() AND table_type IN ('BASE TABLE','VIEW')`)
	if err != nil {
		return nil, fmt.Errorf("inspect PostgreSQL tables: %w", err)
	}
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			rows.Close()
			return nil, err
		}
		result[strings.ToLower(kind)+":"+name] = kind
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT table_name,column_name,data_type,udt_name,is_nullable,COALESCE(column_default,''),COALESCE(character_maximum_length,0),COALESCE(numeric_precision,0),COALESCE(numeric_scale,0),ordinal_position FROM information_schema.columns WHERE table_schema=current_schema() ORDER BY table_name,ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("inspect PostgreSQL columns: %w", err)
	}
	for rows.Next() {
		var table, name, dataType, udt, nullable, defaultValue string
		var charLength, precision, scale, position int64
		if err := rows.Scan(&table, &name, &dataType, &udt, &nullable, &defaultValue, &charLength, &precision, &scale, &position); err != nil {
			rows.Close()
			return nil, err
		}
		defaultValue = strings.ReplaceAll(defaultValue, currentSchema+".", "")
		value := fmt.Sprintf("%s|%s|%s|%s|%d|%d|%d|%d", dataType, udt, nullable, defaultValue, charLength, precision, scale, position)
		result["column:"+table+"."+name] = value
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT tablename,indexname,indexdef FROM pg_indexes WHERE schemaname=current_schema() ORDER BY tablename,indexname`)
	if err != nil {
		return nil, fmt.Errorf("inspect PostgreSQL indexes: %w", err)
	}
	for rows.Next() {
		var table, name, definition string
		if err := rows.Scan(&table, &name, &definition); err != nil {
			rows.Close()
			return nil, err
		}
		definition = strings.ReplaceAll(definition, currentSchema+".", "")
		result["index:"+table+"."+name] = normalizeSchemaSQL(definition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT r.relname,c.conname,c.contype,pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_class r ON r.oid=c.conrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE n.nspname=current_schema() AND c.contype IN ('f','p','u','c') ORDER BY r.relname,c.conname`)
	if err != nil {
		return nil, fmt.Errorf("inspect PostgreSQL constraints: %w", err)
	}
	for rows.Next() {
		var table, name, kind, definition string
		if err := rows.Scan(&table, &name, &kind, &definition); err != nil {
			rows.Close()
			return nil, err
		}
		result["constraint:"+table+"."+name] = kind + "|" + normalizeSchemaSQL(definition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	return result, nil
}

func compareSchemaCatalogs(want, got map[string]string) []SchemaChange {
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
	return changes
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
