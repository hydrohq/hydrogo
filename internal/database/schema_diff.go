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
	if err := ApplyMigrationsForModules(ctx, reference, SQLite, ChainOnly(migrations), enabled); err != nil {
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

// MySQLSchemaDiff builds the selected migration state in a temporary database,
// compares its catalog with the live database, and removes the temporary
// database before returning. The configured database user needs CREATE/DROP
// DATABASE privileges for this inspection.
func MySQLSchemaDiff(ctx context.Context, live *sql.DB, migrations fs.FS, enabled map[string]bool) (changes []SchemaChange, resultErr error) {
	if live == nil {
		return nil, ErrDatabaseRequired
	}
	ctx = nonNilContext(ctx)
	conn, err := live.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("open schema inspection connection: %w", err)
	}
	defer conn.Close()
	var original sql.NullString
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&original); err != nil {
		return nil, fmt.Errorf("read MySQL schema inspection database: %w", err)
	}
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return nil, fmt.Errorf("create temporary schema inspection name: %w", err)
	}
	schema := "hydro_diff_" + hex.EncodeToString(suffix[:])
	if _, err := conn.ExecContext(ctx, "CREATE DATABASE "+quoteMySQLIdentifier(schema)); err != nil {
		return nil, fmt.Errorf("create temporary schema inspection database: %w", err)
	}
	defer func() {
		if original.Valid {
			if _, err := conn.ExecContext(context.Background(), "USE "+quoteMySQLIdentifier(original.String)); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("restore MySQL inspection database: %w", err))
			}
		}
		if _, err := conn.ExecContext(context.Background(), "DROP DATABASE "+quoteMySQLIdentifier(schema)); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove temporary schema inspection database: %w", err))
		}
	}()
	if _, err := conn.ExecContext(ctx, "USE "+quoteMySQLIdentifier(schema)); err != nil {
		return nil, fmt.Errorf("select temporary schema inspection database: %w", err)
	}
	if err := applyMigrationsOnConnection(ctx, conn, MySQL, migrations, enabled, false); err != nil {
		return nil, fmt.Errorf("build migration reference schema: %w", err)
	}
	want, err := mysqlCatalog(ctx, conn)
	if err != nil {
		return nil, err
	}
	if original.Valid {
		if _, err := conn.ExecContext(ctx, "USE "+quoteMySQLIdentifier(original.String)); err != nil {
			return nil, fmt.Errorf("restore MySQL inspection database: %w", err)
		}
	}
	got, err := mysqlCatalog(ctx, conn)
	if err != nil {
		return nil, err
	}
	return compareSchemaCatalogs(want, got), nil
}

func quoteMySQLIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func mysqlCatalog(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) (map[string]string, error) {
	result := make(map[string]string)
	rows, err := db.QueryContext(ctx, `SELECT table_name,table_type FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type IN ('BASE TABLE','VIEW')`)
	if err != nil {
		return nil, fmt.Errorf("inspect MySQL tables: %w", err)
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

	rows, err = db.QueryContext(ctx, `SELECT table_name,column_name,column_type,is_nullable,COALESCE(column_default,''),extra,ordinal_position FROM information_schema.columns WHERE table_schema=DATABASE() ORDER BY table_name,ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("inspect MySQL columns: %w", err)
	}
	for rows.Next() {
		var table, name, columnType, nullable, defaultValue, extra string
		var position int64
		if err := rows.Scan(&table, &name, &columnType, &nullable, &defaultValue, &extra, &position); err != nil {
			rows.Close()
			return nil, err
		}
		result["column:"+table+"."+name] = fmt.Sprintf("%s|%s|%s|%s", strings.ToLower(columnType), nullable, defaultValue, extra) // not the position: baseline and chain place columns differently
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT table_name,index_name,non_unique,seq_in_index,column_name,COALESCE(sub_part,0),index_type FROM information_schema.statistics WHERE table_schema=DATABASE() ORDER BY table_name,index_name,seq_in_index`)
	if err != nil {
		return nil, fmt.Errorf("inspect MySQL indexes: %w", err)
	}
	for rows.Next() {
		var table, name, column, indexType string
		var nonUnique, position, prefix int64
		if err := rows.Scan(&table, &name, &nonUnique, &position, &column, &prefix, &indexType); err != nil {
			rows.Close()
			return nil, err
		}
		key := "index:" + table + "." + name
		entry := fmt.Sprintf("%d|%d|%s|%d|%s", nonUnique, position, column, prefix, strings.ToLower(indexType))
		result[key] = result[key] + ";" + entry
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT k.table_name,k.constraint_name,k.column_name,k.referenced_table_name,k.referenced_column_name,k.ordinal_position,r.update_rule,r.delete_rule FROM information_schema.key_column_usage k JOIN information_schema.referential_constraints r ON r.constraint_schema=k.constraint_schema AND r.constraint_name=k.constraint_name AND r.table_name=k.table_name WHERE k.table_schema=DATABASE() AND k.referenced_table_name IS NOT NULL ORDER BY k.table_name,k.constraint_name,k.ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("inspect MySQL foreign keys: %w", err)
	}
	for rows.Next() {
		var table, name, column, refTable, refColumn, updateRule, deleteRule string
		var position int64
		if err := rows.Scan(&table, &name, &column, &refTable, &refColumn, &position, &updateRule, &deleteRule); err != nil {
			rows.Close()
			return nil, err
		}
		key := "constraint:" + table + "." + name
		entry := fmt.Sprintf("%d|%s|%s|%s|%s|%s", position, column, refTable, refColumn, strings.ToUpper(updateRule), strings.ToUpper(deleteRule))
		result[key] = result[key] + ";" + entry
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	return result, nil
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
		value := fmt.Sprintf("%s|%s|%s|%s|%d|%d|%d", dataType, udt, nullable, defaultValue, charLength, precision, scale) // not the position: baseline and chain place columns differently
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
		} else if CanonicalSchemaSQL(kind, actual) != CanonicalSchemaSQL(kind, expected) {
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

func normalizeSchemaSQL(value string) string { return NormalizeSchemaSQL(value) }

func sortSchemaChanges(changes []SchemaChange) {
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Object == changes[j].Object {
			return changes[i].Name < changes[j].Name
		}
		return changes[i].Object < changes[j].Object
	})
}
