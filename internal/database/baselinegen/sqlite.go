package baselinegen

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"strings"
)

// shadowSuffixes are the tables FTS5 creates for a virtual table; creating the
// virtual table recreates them, so a baseline script must not.
var shadowSuffixes = []string{"_data", "_idx", "_content", "_docsize", "_config"}

func takeSQLite(ctx context.Context, db *sql.DB) (*Snapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT type,name,tbl_name,COALESCE(sql,'') FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY rowid`)
	if err != nil {
		return nil, fmt.Errorf("read SQLite catalog: %w", err)
	}
	type entry struct{ kind, name, table, sql string }
	var entries []entry
	virtual := map[string]bool{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.kind, &e.name, &e.table, &e.sql); err != nil {
			rows.Close()
			return nil, err
		}
		if e.kind == "table" && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(e.sql)), "CREATE VIRTUAL TABLE") {
			virtual[e.name] = true
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	isShadow := func(name string) bool {
		for table := range virtual {
			for _, suffix := range shadowSuffixes {
				if name == table+suffix {
					return true
				}
			}
		}
		return false
	}
	snapshot := &Snapshot{}
	var dataTables []string
	for _, e := range entries {
		if e.name == "schema_migrations" || e.sql == "" || isShadow(e.name) {
			continue
		}
		switch e.kind {
		case "table", "index", "trigger", "view":
		default:
			return nil, fmt.Errorf("unsupported SQLite object %s %s", e.kind, e.name)
		}
		snapshot.Objects = append(snapshot.Objects, Object{Kind: e.kind, Name: e.name, Table: e.table, SQL: strings.TrimSpace(e.sql) + ";"})
		if e.kind == "table" && !virtual[e.name] {
			dataTables = append(dataTables, e.name)
		}
	}
	if err := collectRows(ctx, db, database.SQLite, dataTables, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}
