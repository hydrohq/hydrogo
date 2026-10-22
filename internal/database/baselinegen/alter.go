package baselinegen

import (
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// A group may build on tables a lower group created: a later migration adds a
// column or swaps an index. The script expresses that as ALTER and DROP
// statements, for the changes the chain actually makes. Anything else (a
// dropped column, a changed type, a changed table constraint) is an error: the
// generator refuses what it cannot render faithfully.

func shortName(object Object) string {
	if _, name, ok := strings.Cut(object.Name, "."); ok && object.Table != "" && object.Kind != "table" {
		return name
	}
	return object.Name
}

// dropStatement undoes one object a group below created.
func dropStatement(dialect database.Dialect, object Object) (string, error) {
	q := func(name string) string { return quoteIdent(dialect, name) }
	switch dialect {
	case database.SQLite:
		switch object.Kind {
		case "index":
			return "DROP INDEX IF EXISTS " + q(object.Name) + ";", nil
		case "trigger":
			return "DROP TRIGGER IF EXISTS " + q(object.Name) + ";", nil
		case "view":
			return "DROP VIEW IF EXISTS " + q(object.Name) + ";", nil
		}
	case database.Postgres:
		switch object.Kind {
		case "index":
			return "DROP INDEX IF EXISTS " + q(shortName(object)) + ";", nil
		case "constraint", "foreign_key":
			return "ALTER TABLE " + q(object.Table) + " DROP CONSTRAINT " + q(shortName(object)) + ";", nil
		case "trigger":
			return "DROP TRIGGER IF EXISTS " + q(shortName(object)) + " ON " + q(object.Table) + ";", nil
		case "policy":
			return "DROP POLICY IF EXISTS " + q(shortName(object)) + " ON " + q(object.Table) + ";", nil
		}
	case database.MySQL:
		if object.Kind == "trigger" {
			return "DROP TRIGGER IF EXISTS " + q(object.Name) + ";", nil
		}
	}
	return "", fmt.Errorf("cannot render the removal of %s", object.Key())
}

var sqliteConstraintStarts = []string{"CONSTRAINT ", "PRIMARY KEY", "UNIQUE", "CHECK", "FOREIGN KEY"}

func isSQLiteTableConstraint(piece string) bool {
	upper := strings.ToUpper(strings.TrimSpace(piece))
	for _, start := range sqliteConstraintStarts {
		if strings.HasPrefix(upper, start) {
			return true
		}
	}
	return false
}

func oneLine(piece string) string { return strings.Join(strings.Fields(piece), " ") }

// alterTable renders the statements that turn the table `below` into `here`.
func alterTable(dialect database.Dialect, below, here Object) ([]string, error) {
	_, belowPieces, _, ok1 := splitBody(below.SQL)
	_, herePieces, _, ok2 := splitBody(here.SQL)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot read the definition of table %s", here.Name)
	}
	count := func(pieces []string) map[string]int {
		counts := map[string]int{}
		for _, piece := range pieces {
			counts[Normalize(piece)]++
		}
		return counts
	}
	belowCount, hereCount := count(belowPieces), count(herePieces)
	var statements []string
	q := func(name string) string { return quoteIdent(dialect, name) }

	// Pieces only the lower definition has: a column or key was removed.
	for _, piece := range belowPieces {
		key := Normalize(piece)
		if hereCount[key] > 0 {
			hereCount[key]--
			continue
		}
		if dialect == database.MySQL {
			upper := strings.ToUpper(strings.TrimSpace(piece))
			switch {
			case strings.HasPrefix(upper, "KEY "), strings.HasPrefix(upper, "UNIQUE KEY "), strings.HasPrefix(upper, "FULLTEXT KEY "):
				statements = append(statements, "ALTER TABLE "+q(here.Name)+" DROP INDEX "+firstQuoted(piece)+";")
				continue
			case strings.HasPrefix(upper, "CONSTRAINT "):
				statements = append(statements, "ALTER TABLE "+q(here.Name)+" DROP FOREIGN KEY "+firstQuoted(piece)+";")
				continue
			}
		}
		return nil, fmt.Errorf("table %s loses %q that a group below created", here.Name, oneLine(piece))
	}
	// Pieces only the new definition has: a column or key was added.
	hereCount = count(herePieces)
	for _, piece := range herePieces {
		key := Normalize(piece)
		if belowCount[key] > 0 {
			belowCount[key]--
			continue
		}
		switch dialect {
		case database.SQLite:
			if isSQLiteTableConstraint(piece) {
				return nil, fmt.Errorf("table %s gains the table constraint %q; SQLite cannot add one to an existing table", here.Name, oneLine(piece))
			}
			statements = append(statements, "ALTER TABLE "+q(here.Name)+" ADD COLUMN "+oneLine(piece)+";")
		case database.Postgres:
			statements = append(statements, "ALTER TABLE "+q(here.Name)+" ADD COLUMN "+oneLine(piece)+";")
		case database.MySQL:
			if strings.HasPrefix(strings.TrimSpace(piece), "`") {
				statements = append(statements, "ALTER TABLE "+q(here.Name)+" ADD COLUMN "+oneLine(piece)+";")
			} else {
				statements = append(statements, "ALTER TABLE "+q(here.Name)+" ADD "+oneLine(piece)+";")
			}
		}
	}
	if len(statements) == 0 {
		return nil, fmt.Errorf("table %s differs from the lower definition only in ways the baseline cannot render", here.Name)
	}
	return statements, nil
}

// firstQuoted returns the first back-quoted identifier of a MySQL key line.
func firstQuoted(piece string) string {
	start := strings.Index(piece, "`")
	if start < 0 {
		return "``"
	}
	end := strings.Index(piece[start+1:], "`")
	if end < 0 {
		return "``"
	}
	return piece[start : start+end+2]
}
