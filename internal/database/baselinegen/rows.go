package baselinegen

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func quoteIdent(dialect database.Dialect, name string) string {
	if dialect == database.MySQL {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteString(dialect database.Dialect, value string) string {
	value = strings.ReplaceAll(value, "'", "''")
	if dialect == database.MySQL {
		value = strings.ReplaceAll(value, `\`, `\\`)
	}
	return "'" + value + "'"
}

// formatValue renders a scanned value as a SQL literal. Unknown types fail so a
// future seed row with an unusual column cannot be dropped silently.
func formatValue(dialect database.Dialect, columnType string, value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "NULL", nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case bool:
		if v {
			return "TRUE", nil
		}
		return "FALSE", nil
	case string:
		return quoteString(dialect, v), nil
	case time.Time:
		if dialect == database.MySQL {
			return quoteString(dialect, v.UTC().Format("2006-01-02 15:04:05.999999")), nil
		}
		return quoteString(dialect, v.UTC().Format(time.RFC3339Nano)), nil
	case []byte:
		if isBinaryColumn(dialect, columnType) {
			switch dialect {
			case database.Postgres:
				return `'\x` + hex.EncodeToString(v) + `'`, nil
			case database.MySQL:
				if len(v) == 0 {
					return "''", nil
				}
				return "0x" + hex.EncodeToString(v), nil
			default:
				return "X'" + hex.EncodeToString(v) + "'", nil
			}
		}
		return quoteString(dialect, string(v)), nil
	default:
		return "", fmt.Errorf("cannot render %T as a baseline seed value", value)
	}
}

func isBinaryColumn(dialect database.Dialect, columnType string) bool {
	name := strings.ToUpper(columnType)
	switch dialect {
	case database.Postgres:
		return name == "BYTEA"
	case database.MySQL:
		return strings.Contains(name, "BLOB") || strings.Contains(name, "BINARY")
	default:
		return name == "BLOB" || name == ""
	}
}

// tableRows returns the column names and the sorted value tuples of a table.
func tableRows(ctx context.Context, db *sql.DB, dialect database.Dialect, table string) ([]string, []string, error) {
	rows, err := db.QueryContext(ctx, "SELECT * FROM "+quoteIdent(dialect, table))
	if err != nil {
		return nil, nil, fmt.Errorf("read rows of %s: %w", table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	var tuples []string
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for index := range values {
			targets[index] = &values[index]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, nil, err
		}
		parts := make([]string, len(columns))
		for index, value := range values {
			text, err := formatValue(dialect, types[index].DatabaseTypeName(), value)
			if err != nil {
				return nil, nil, fmt.Errorf("%s.%s: %w", table, columns[index], err)
			}
			parts[index] = text
		}
		tuples = append(tuples, "("+strings.Join(parts, ", ")+")")
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	sort.Strings(tuples)
	return columns, tuples, nil
}

// collectRows fills Columns and Rows for the given tables, keeping only tables
// that hold data (an empty table contributes nothing to compare or generate).
func collectRows(ctx context.Context, db *sql.DB, dialect database.Dialect, tables []string, snapshot *Snapshot) error {
	snapshot.Columns = map[string][]string{}
	snapshot.Rows = map[string][]string{}
	for _, table := range tables {
		columns, tuples, err := tableRows(ctx, db, dialect, table)
		if err != nil {
			return err
		}
		snapshot.Columns[table] = columns
		if len(tuples) > 0 {
			snapshot.Rows[table] = tuples
		}
	}
	return nil
}
