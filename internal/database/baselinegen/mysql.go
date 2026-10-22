package baselinegen

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var (
	autoIncrementValue = regexp.MustCompile(` AUTO_INCREMENT=\d+`)
	definerClause      = regexp.MustCompile("DEFINER=`[^`]*`@`[^`]*` ")
)

func takeMySQL(ctx context.Context, db *sql.DB) (*Snapshot, error) {
	snapshot := &Snapshot{}
	var unsupported int
	if err := db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema=DATABASE()) + (SELECT COUNT(*) FROM information_schema.events WHERE event_schema=DATABASE()) + (SELECT COUNT(*) FROM information_schema.views WHERE table_schema=DATABASE())`).Scan(&unsupported); err != nil {
		return nil, fmt.Errorf("check MySQL routines, events and views: %w", err)
	}
	if unsupported != 0 {
		return nil, fmt.Errorf("the MySQL schema has %d routines, events or views; the baseline renders tables and triggers only", unsupported)
	}
	names, err := stringColumn(ctx, db, `SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type='BASE TABLE' AND table_name <> 'schema_migrations' ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("list MySQL tables: %w", err)
	}
	for _, name := range names {
		var ignored, definition string
		if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE "+quoteIdent(database.MySQL, name)).Scan(&ignored, &definition); err != nil {
			return nil, fmt.Errorf("show create table %s: %w", name, err)
		}
		definition = autoIncrementValue.ReplaceAllString(definition, "")
		definition = strings.Replace(definition, "CREATE TABLE ", "CREATE TABLE IF NOT EXISTS ", 1)
		snapshot.Objects = append(snapshot.Objects, Object{Kind: "table", Name: name, Table: name, SQL: definition + ";"})
	}
	triggers, err := stringColumn(ctx, db, `SELECT trigger_name FROM information_schema.triggers WHERE trigger_schema=DATABASE() ORDER BY trigger_name`)
	if err != nil {
		return nil, fmt.Errorf("list MySQL triggers: %w", err)
	}
	for _, name := range triggers {
		var tableName string
		if err := db.QueryRowContext(ctx, `SELECT event_object_table FROM information_schema.triggers WHERE trigger_schema=DATABASE() AND trigger_name=?`, name).Scan(&tableName); err != nil {
			return nil, err
		}
		var ignored1, ignored2, definition, ignored3, ignored4, ignored5, ignored6 sql.NullString
		if err := db.QueryRowContext(ctx, "SHOW CREATE TRIGGER "+quoteIdent(database.MySQL, name)).Scan(&ignored1, &ignored2, &definition, &ignored3, &ignored4, &ignored5, &ignored6); err != nil {
			return nil, fmt.Errorf("show create trigger %s: %w", name, err)
		}
		statement := definerClause.ReplaceAllString(definition.String, "")
		snapshot.Objects = append(snapshot.Objects, Object{Kind: "trigger", Name: name, Table: tableName,
			SQL: "DROP TRIGGER IF EXISTS " + quoteIdent(database.MySQL, name) + ";\n" + statement + ";"})
	}
	if err := collectRows(ctx, db, database.MySQL, names, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func stringColumn(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
