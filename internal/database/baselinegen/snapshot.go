// Package baselinegen builds and verifies the schema baseline (ADR 0009). It
// reads a database the migration chain produced, takes a snapshot of its schema
// and seed rows, and renders per-dialect scripts from the difference between
// module sets. Nothing here runs in the server; tools/baselinegen writes the
// scripts and the tests compare chain-built and baseline-built databases.
package baselinegen

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// Object is one schema object in the form the baseline script recreates it.
type Object struct {
	Kind  string // table, index, constraint, function, trigger, rls, policy, view
	Name  string // unique within Kind; "table.name" for objects that belong to a table
	Table string // owning table, empty for functions
	SQL   string // executable statement(s) that create the object
}

// Key identifies the object across snapshots.
func (o Object) Key() string { return o.Kind + ":" + o.Name }

// Snapshot is the schema and seed data of one database. The migration ledger is
// excluded: the runner writes it, not the baseline script.
type Snapshot struct {
	Dialect database.Dialect
	Objects []Object            // in the order the source database created them
	Columns map[string][]string // table -> column names
	Rows    map[string][]string // table -> sorted value tuples, as SQL literals
}

// Take snapshots the connected database.
func Take(ctx context.Context, db *sql.DB, dialect database.Dialect) (*Snapshot, error) {
	var (
		snapshot *Snapshot
		err      error
	)
	switch dialect {
	case database.SQLite:
		snapshot, err = takeSQLite(ctx, db)
	case database.Postgres:
		snapshot, err = takePostgres(ctx, db)
	case database.MySQL:
		snapshot, err = takeMySQL(ctx, db)
	default:
		return nil, fmt.Errorf("baseline snapshots are not supported for dialect %q", dialect)
	}
	if err != nil {
		return nil, err
	}
	snapshot.Dialect = dialect
	return snapshot, nil
}

// Tables lists the table names of the snapshot.
func (s *Snapshot) Tables() []string {
	var tables []string
	for _, object := range s.Objects {
		if object.Kind == "table" {
			tables = append(tables, object.Name)
		}
	}
	sort.Strings(tables)
	return tables
}

// Diff lists how two snapshots differ semantically; empty means equivalent.
func Diff(want, got *Snapshot) []string {
	var problems []string
	wantObjects := make(map[string]Object, len(want.Objects))
	for _, object := range want.Objects {
		wantObjects[object.Key()] = object
	}
	gotObjects := make(map[string]Object, len(got.Objects))
	for _, object := range got.Objects {
		gotObjects[object.Key()] = object
	}
	for key, object := range wantObjects {
		other, ok := gotObjects[key]
		switch {
		case !ok:
			problems = append(problems, "missing "+key)
		case canonical(other) != canonical(object):
			problems = append(problems, fmt.Sprintf("definition differs %s:\n   want %s\n   got  %s", key, canonical(object), canonical(other)))
		}
	}
	for key := range gotObjects {
		if _, ok := wantObjects[key]; !ok {
			problems = append(problems, "unexpected "+key)
		}
	}
	tables := map[string]struct{}{}
	for table := range want.Rows {
		tables[table] = struct{}{}
	}
	for table := range got.Rows {
		tables[table] = struct{}{}
	}
	for table := range tables {
		if strings.Join(want.Rows[table], "\n") != strings.Join(got.Rows[table], "\n") {
			problems = append(problems, fmt.Sprintf("rows differ in %s: want %v got %v", table, want.Rows[table], got.Rows[table]))
		}
	}
	sort.Strings(problems)
	return problems
}

// Normalize makes two spellings of the same definition compare equal.
func Normalize(sqlText string) string { return database.NormalizeSchemaSQL(sqlText) }

func canonical(object Object) string { return database.CanonicalSchemaSQL(object.Kind, object.SQL) }

func splitBody(sqlText string) (prefix string, pieces []string, suffix string, ok bool) {
	return database.SplitTableBody(sqlText)
}
