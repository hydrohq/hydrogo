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

var managedName = regexp.MustCompile(`^idx_hcr_(trgm_)?[0-9a-f]{20}$`)

// SyncIndexes builds an expression index per Indexed field and drops indexes
// of fields that no longer need one. Records live in one shared table, so each
// index leads with tenant_id (and partial WHERE collection_id on Postgres/SQLite).
func (s Store) SyncIndexes(ctx context.Context, schemas []collections.Schema) error {
	var trigramReady bool
	if s.Dialect == database.Postgres {
		trigramReady = s.ensureTrigramExtension(ctx)
	}
	wanted := map[string]string{}
	for _, schema := range schemas {
		for _, field := range schema.Fields {
			if !field.Indexed {
				continue
			}
			defs, err := indexStatements(s.Dialect, schema.ID, field, trigramReady)
			if err != nil {
				return err
			}
			for _, def := range defs {
				wanted[def.name] = def.statement
			}
		}
	}
	existing, err := s.managedIndexes(ctx)
	if err != nil {
		return err
	}
	existingMap := make(map[string]bool, len(existing))
	for _, name := range existing {
		existingMap[name] = true
	}
	names := make([]string, 0, len(wanted))
	for name := range wanted {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if existingMap[name] {
			continue
		}
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
		dropSQL := `DROP INDEX IF EXISTS ` + name
		if s.Dialect == database.MySQL {
			dropSQL = `ALTER TABLE hydro_collection_records DROP INDEX ` + name
		}
		if _, err := s.DB.ExecContext(ctx, dropSQL); err != nil {
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
	var (
		rows *sql.Rows
		err  error
	)
	switch s.Dialect {
	case database.Postgres:
		rows, err = q.QueryContext(ctx, `SELECT indexname FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'hydro_collection_records' AND indexname LIKE 'idx\_hcr\_%'`)
	case database.MySQL:
		rows, err = q.QueryContext(ctx, `SELECT DISTINCT index_name FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'hydro_collection_records' AND index_name LIKE 'idx_hcr_%'`)
	default: // SQLite
		rows, err = q.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'hydro_collection_records' AND name LIKE 'idx_hcr_%'`)
	}
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

// ensureTrigramExtension enables pg_trgm so a GIN trigram index can
// accelerate leading-wildcard LIKE searches. It returns false, rather than an
// error, when the extension cannot be created (e.g. a managed Postgres
// without CREATE EXTENSION privilege) so the caller can skip trigram indexes
// and keep every existing index unchanged.
func (s Store) ensureTrigramExtension(ctx context.Context) bool {
	_, err := s.DB.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm`)
	return err == nil
}

type indexDef struct {
	name, statement string
}

// indexStatements renders the CREATE INDEX statements for one field.
func indexStatements(dialect database.Dialect, collectionID string, field collections.Field, trigramReady bool) ([]indexDef, error) {
	if !validFieldID(collectionID) || !validFieldID(field.ID) {
		return nil, collections.ErrInvalidSchema
	}
	sum := sha256.Sum256([]byte(collectionID + "/" + field.ID))
	hash := hex.EncodeToString(sum[:])[:20]
	name := indexPrefix + hash
	var defs []indexDef

	switch dialect {
	case database.Postgres:
		var columns string
		switch field.Kind {
		case collections.Text, collections.RichText, collections.Select, collections.DateTime, collections.Media, collections.Relation:
			columns = `tenant_id, (data_json::jsonb ->> '` + field.ID + `'), updated_at DESC, id`
		case collections.Number, collections.Boolean:
			columns = `tenant_id, (data_json::jsonb -> '` + field.ID + `') DESC, id`
		default:
			return nil, nil
		}
		defs = append(defs, indexDef{name, `CREATE INDEX IF NOT EXISTS ` + name + ` ON hydro_collection_records (` + columns + `) WHERE collection_id = '` + collectionID + `'`})
		if trigramReady && (field.Kind == collections.Text || field.Kind == collections.RichText) {
			trgmName := indexPrefix + "trgm_" + hash
			expr := `lower((data_json::jsonb ->> '` + field.ID + `'))`
			defs = append(defs, indexDef{trgmName, `CREATE INDEX IF NOT EXISTS ` + trgmName + ` ON hydro_collection_records USING gin (` + expr + ` gin_trgm_ops) WHERE collection_id = '` + collectionID + `'`})
		}

	case database.SQLite:
		var columns string
		switch field.Kind {
		case collections.Text, collections.RichText, collections.Select, collections.DateTime, collections.Media, collections.Relation:
			columns = `tenant_id, json_extract(data_json, '$.` + field.ID + `'), updated_at DESC, id`
		case collections.Number, collections.Boolean:
			columns = `tenant_id, json_extract(data_json, '$.` + field.ID + `') DESC, id`
		default:
			return nil, nil
		}
		defs = append(defs, indexDef{name, `CREATE INDEX IF NOT EXISTS ` + name + ` ON hydro_collection_records (` + columns + `) WHERE collection_id = '` + collectionID + `'`})

	case database.MySQL:
		var columnExpr string
		switch field.Kind {
		case collections.Text, collections.RichText, collections.Select, collections.DateTime, collections.Media, collections.Relation:
			columnExpr = `(CAST(JSON_UNQUOTE(JSON_EXTRACT(data_json, '$.` + field.ID + `')) AS CHAR(255)))`
		case collections.Number:
			columnExpr = `(CAST(JSON_EXTRACT(data_json, '$.` + field.ID + `') AS DECIMAL(20, 4)))`
		case collections.Boolean:
			columnExpr = `(CAST(JSON_EXTRACT(data_json, '$.` + field.ID + `') AS UNSIGNED))`
		default:
			return nil, nil
		}
		defs = append(defs, indexDef{name, `CREATE INDEX ` + name + ` ON hydro_collection_records (tenant_id, collection_id, ` + columnExpr + `, updated_at, id)`})
	}

	return defs, nil
}
