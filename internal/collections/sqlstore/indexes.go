package sqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// indexPrefix names the indexes this file manages, so it can drop the ones
// whose field is no longer indexed without touching anything else.
const indexPrefix = "idx_hcr_"

var managedName = regexp.MustCompile(`^idx_hcr_[0-9a-f]{20}$`)

// SyncIndexes builds an expression index per Indexed field and drops indexes
// of fields that no longer need one. Records live in one shared table, so each
// index is partial (one collection) and leads with tenant_id. Queries filter
// with the same expressions as buildRecordQuery; PostgreSQL plans each
// statement with its bound values, so the bound field name still matches the
// index. Only PostgreSQL is supported: SQLite and MySQL fall back to scans
// (SQLite installs are small, and MySQL needs functional indexes on
// generated JSON expressions that are not verified here).
func (s Store) SyncIndexes(ctx context.Context, schemas []collections.Schema) error {
	if s.Dialect != database.Postgres {
		return nil
	}
	wanted := map[string]string{}
	for _, schema := range schemas {
		for _, field := range schema.Fields {
			if !field.Indexed {
				continue
			}
			name, statement, err := indexStatement(schema.ID, field)
			if err != nil {
				return err
			}
			if statement != "" {
				wanted[name] = statement
			}
		}
	}
	existing, err := s.managedIndexes(ctx)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(wanted))
	for name := range wanted {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := s.DB.ExecContext(ctx, wanted[name]); err != nil {
			return fmt.Errorf("create %s: %w", name, err)
		}
	}
	for _, name := range existing {
		if _, keep := wanted[name]; keep {
			continue
		}
		// The name is inlined in DDL, so only drop names of the exact shape
		// this file generates.
		if !managedName.MatchString(name) {
			continue
		}
		if _, err := s.DB.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
	}
	return nil
}

func (s Store) managedIndexes(ctx context.Context) ([]string, error) {
	q, ok := s.DB.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("index query is unavailable")
	}
	rows, err := q.QueryContext(ctx, `SELECT indexname FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'hydro_collection_records' AND indexname LIKE 'idx\_hcr\_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// indexStatement renders the CREATE INDEX for one field, or "" when the
// field's kind has no scalar value to index. Collection and field IDs are
// inlined (DDL cannot bind them), so both are re-checked as plain identifiers.
func indexStatement(collectionID string, field collections.Field) (name, statement string, err error) {
	if !validFieldID(collectionID) || !validFieldID(field.ID) {
		return "", "", collections.ErrInvalidSchema
	}
	sum := sha256.Sum256([]byte(collectionID + "/" + field.ID))
	name = indexPrefix + hex.EncodeToString(sum[:])[:20]
	var columns string
	switch field.Kind {
	case collections.Text, collections.RichText, collections.Select, collections.DateTime, collections.Media, collections.Relation:
		columns = `tenant_id, (data_json::jsonb ->> '` + field.ID + `'), updated_at DESC, id`
	case collections.Number, collections.Boolean:
		columns = `tenant_id, (data_json::jsonb -> '` + field.ID + `') DESC, id`
	default:
		return name, "", nil
	}
	return name, `CREATE INDEX IF NOT EXISTS ` + name + ` ON hydro_collection_records (` + columns + `) WHERE collection_id = '` + collectionID + `'`, nil
}
