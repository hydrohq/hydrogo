// Package sqlstore persists dynamic collection metadata and JSON records in
// static, versioned tables. It never builds SQL from collection field names:
// list queries pass field paths as bound parameters (see query.go).
package sqlstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	DB                 database.Executor
	Dialect            database.Dialect
	RequireTenantScope bool
	// MaxRecords caps the records of one type per tenant. Zero means
	// collections.DefaultMaxRecords; a negative value removes the cap and
	// with it the per-insert COUNT(*).
	MaxRecords int
	// Events, when set, is called inside the transaction of every record
	// create, update and delete with the topic (content.created,
	// content.updated, content.deleted) and the aggregate id "<type>/<id>",
	// so the event commits or rolls back with the record. An error aborts the
	// write.
	Events func(ctx context.Context, tx database.Executor, topic, aggregateID string, version int) error
}

func (s Store) emit(ctx context.Context, topic, collectionID, id string, version int) error {
	if s.Events == nil {
		return nil
	}
	return s.Events(ctx, s.DB, topic, collectionID+"/"+id, version)
}

func (s Store) recordCap() int {
	if s.MaxRecords == 0 {
		return collections.DefaultMaxRecords
	}
	return s.MaxRecords
}

func (s Store) scope(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}
func (s Store) p(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
func (s Store) within(ctx context.Context, fn func(Store) error) error {
	if db, ok := s.DB.(*sql.DB); ok {
		return database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error { next := s; next.DB = tx; return fn(next) })
	}
	return fn(s)
}
func (s Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if err := database.RequireExecutor(s.DB); err != nil {
		return nil, err
	}
	return s.DB.ExecContext(ctx, query, args...)
}
func (s Store) row(ctx context.Context, query string, args ...any) *sql.Row {
	return s.DB.QueryRowContext(ctx, query, args...)
}
func (s Store) rows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	q, ok := s.DB.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("collection query is unavailable")
	}
	return q.QueryContext(ctx, query, args...)
}

func (s Store) ListSchemas(ctx context.Context) ([]collections.Schema, error) {
	var result []collections.Schema
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := `SELECT schema_json FROM hydro_collections`
		args := []any{}
		if tx.RequireTenantScope {
			query += ` WHERE tenant_id=` + tx.p(1)
			args = append(args, tenantID)
		}
		query += ` ORDER BY id ASC`
		rows, err := tx.rows(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		result = make([]collections.Schema, 0)
		for rows.Next() {
			var raw string
			if err := rows.Scan(&raw); err != nil {
				return err
			}
			var item collections.Schema
			if json.Unmarshal([]byte(raw), &item) != nil || item.Validate() != nil {
				return errors.New("stored collection schema is invalid")
			}
			result = append(result, item)
		}
		return rows.Err()
	})
	return result, err
}

func (s Store) GetSchema(ctx context.Context, id string) (collections.Schema, error) {
	var result collections.Schema
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := `SELECT schema_json FROM hydro_collections WHERE tenant_id=` + tx.p(1) + ` AND id=` + tx.p(2)
		args := []any{"", id}
		if tx.RequireTenantScope {
			args[0] = tenantID
		}
		var raw string
		if err := tx.row(ctx, query, args...).Scan(&raw); err != nil {
			return mapNoRows(err, collections.ErrCollectionNotFound)
		}
		if json.Unmarshal([]byte(raw), &result) != nil || result.Validate() != nil {
			return errors.New("stored collection schema is invalid")
		}
		return nil
	})
	return result, err
}

func (s Store) CreateSchema(ctx context.Context, schema collections.Schema) error {
	if err := schema.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	return s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		query := `INSERT INTO hydro_collections (tenant_id,id,label,schema_json,version,created_at,updated_at) VALUES (` + tx.p(1) + `,` + tx.p(2) + `,` + tx.p(3) + `,` + tx.p(4) + `,1,` + tx.p(5) + `,` + tx.p(6) + `)`
		_, err = tx.exec(ctx, query, tenantID, schema.ID, schema.Label, string(raw), now, now)
		if isConflict(err) {
			return collections.ErrInvalidSchema
		}
		return err
	})
}

func (s Store) UpdateSchema(ctx context.Context, schema collections.Schema) error {
	if err := schema.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	return s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		query := `UPDATE hydro_collections SET label=` + tx.p(1) + `,schema_json=` + tx.p(2) + `,version=version+1,updated_at=` + tx.p(3) + ` WHERE tenant_id=` + tx.p(4) + ` AND id=` + tx.p(5)
		result, err := tx.exec(ctx, query, schema.Label, string(raw), now, tenantID, schema.ID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return collections.ErrCollectionNotFound
		}
		return nil
	})
}

func (s Store) DeleteSchema(ctx context.Context, id string) error {
	return s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := `DELETE FROM hydro_collections WHERE tenant_id=` + tx.p(1) + ` AND id=` + tx.p(2)
		result, err := tx.exec(ctx, query, tenantID, id)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return collections.ErrCollectionNotFound
		}
		return nil
	})
}

func (s Store) GetRecord(ctx context.Context, collectionID, id string) (collections.Record, error) {
	var result collections.Record
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := `SELECT data_json,version,created_at,updated_at FROM hydro_collection_records WHERE tenant_id=` + tx.p(1) + ` AND collection_id=` + tx.p(2) + ` AND id=` + tx.p(3)
		var raw, created, updated string
		result = collections.Record{ID: id, CollectionID: collectionID}
		if err := tx.row(ctx, query, tenantID, collectionID, id).Scan(&raw, &result.Version, &created, &updated); err != nil {
			return mapNoRows(err, collections.ErrRecordNotFound)
		}
		if json.Unmarshal([]byte(raw), &result.Data) != nil {
			return errors.New("stored collection record is invalid")
		}
		result.CreatedAt = parseTime(created)
		result.UpdatedAt = parseTime(updated)
		return nil
	})
	return result, err
}

func (s Store) SaveRecord(ctx context.Context, collectionID, id string, data map[string]any, unique []collections.Field, expectedVersion int) (collections.Record, error) {
	var result collections.Record
	err := s.within(ctx, func(tx Store) error {
		return tx.saveRecord(ctx, collectionID, id, data, unique, expectedVersion, &result)
	})
	return result, err
}

func (s Store) saveRecord(ctx context.Context, collectionID, id string, data map[string]any, unique []collections.Field, expectedVersion int, result *collections.Record) error {
	tenantID, err := s.scope(ctx)
	if err != nil {
		return err
	}
	creating := id == ""
	if id == "" {
		id, err = newID()
		if err != nil {
			return err
		}
	}
	encoded, err := json.Marshal(data)
	if err != nil || len(encoded) > collections.MaxRecordBytes {
		return collections.ErrInvalidRecord
	}
	if limit := s.recordCap(); creating && limit > 0 {
		var count int
		if err := s.row(ctx, `SELECT COUNT(*) FROM hydro_collection_records WHERE tenant_id=`+s.p(1)+` AND collection_id=`+s.p(2), tenantID, collectionID).Scan(&count); err != nil {
			return err
		}
		if count >= limit {
			return collections.ErrInvalidRecord
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var created string
	var version int
	topic := "content.updated"
	selectQuery := `SELECT created_at,version FROM hydro_collection_records WHERE tenant_id=` + s.p(1) + ` AND collection_id=` + s.p(2) + ` AND id=` + s.p(3)
	if err := s.row(ctx, selectQuery, tenantID, collectionID, id).Scan(&created, &version); err == nil {
		if expectedVersion > 0 && version != expectedVersion {
			return collections.ErrVersionConflict
		}
		// Conditioning on the version just read makes the write atomic even
		// when another transaction updated the row in between.
		update := `UPDATE hydro_collection_records SET data_json=` + s.p(1) + `,version=` + s.p(2) + `,updated_at=` + s.p(3) + ` WHERE tenant_id=` + s.p(4) + ` AND collection_id=` + s.p(5) + ` AND id=` + s.p(6) + ` AND version=` + s.p(7)
		updated, err := s.exec(ctx, update, string(encoded), version+1, now, tenantID, collectionID, id, version)
		if err != nil {
			return err
		}
		if n, _ := updated.RowsAffected(); n == 0 {
			return collections.ErrVersionConflict
		}
		version++
	} else if errors.Is(err, sql.ErrNoRows) {
		created = now
		version = 1
		topic = "content.created"
		insert := `INSERT INTO hydro_collection_records (tenant_id,collection_id,id,data_json,version,created_at,updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,1,` + s.p(5) + `,` + s.p(6) + `)`
		if _, err := s.exec(ctx, insert, tenantID, collectionID, id, string(encoded), now, now); err != nil {
			return err
		}
	} else {
		return err
	}
	if _, err := s.exec(ctx, `DELETE FROM hydro_collection_unique WHERE tenant_id=`+s.p(1)+` AND collection_id=`+s.p(2)+` AND record_id=`+s.p(3), tenantID, collectionID, id); err != nil {
		return err
	}
	if _, err := s.exec(ctx, `DELETE FROM hydro_collection_relations WHERE tenant_id=`+s.p(1)+` AND source_collection_id=`+s.p(2)+` AND source_record_id=`+s.p(3), tenantID, collectionID, id); err != nil {
		return err
	}
	for _, field := range unique {
		value, ok := data[field.ID]
		if !ok {
			continue
		}
		raw, _ := json.Marshal(value)
		digest := sha256.Sum256(raw)
		query := `INSERT INTO hydro_collection_unique (tenant_id,collection_id,field_id,value_hash,record_id) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `)`
		if _, err := s.exec(ctx, query, tenantID, collectionID, field.ID, hex.EncodeToString(digest[:]), id); err != nil {
			if isConflict(err) {
				return collections.ErrUniqueValue
			}
			return err
		}
	}
	// Relation links are materialized in a static table so both sides are
	// protected by database foreign keys even though the schema is dynamic.
	schema, err := s.GetSchema(ctx, collectionID)
	if err != nil {
		return err
	}
	for _, field := range schema.Fields {
		if field.Kind != collections.Relation && field.Kind != collections.RelationMany {
			continue
		}
		value, ok := data[field.ID]
		if !ok {
			continue
		}
		targets := []string{}
		if field.Kind == collections.Relation {
			targets = append(targets, value.(string))
		} else {
			for _, item := range value.([]any) {
				targets = append(targets, item.(string))
			}
		}
		for _, target := range targets {
			query := `INSERT INTO hydro_collection_relations (tenant_id,source_collection_id,source_record_id,field_id,target_collection_id,target_record_id) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `)`
			if _, err := s.exec(ctx, query, tenantID, collectionID, id, field.ID, field.Relation, target); err != nil {
				if isConflict(err) {
					return collections.ErrInvalidRecord
				}
				return err
			}
		}
	}
	if schema.Revisions {
		if err := s.writeRevision(ctx, tenantID, collectionID, id, version, string(encoded), now); err != nil {
			return err
		}
	}
	if err := s.emit(ctx, topic, collectionID, id, version); err != nil {
		return err
	}
	*result = collections.Record{ID: id, CollectionID: collectionID, Data: data, Version: version, CreatedAt: parseTime(created), UpdatedAt: parseTime(now)}
	return nil
}

func (s Store) DeleteRecord(ctx context.Context, collectionID, id string) error {
	return s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		result, err := tx.exec(ctx, `DELETE FROM hydro_collection_records WHERE tenant_id=`+tx.p(1)+` AND collection_id=`+tx.p(2)+` AND id=`+tx.p(3), tenantID, collectionID, id)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return collections.ErrRecordNotFound
		}
		return tx.emit(ctx, "content.deleted", collectionID, id, 0)
	})
}

func (s Store) RecordExists(ctx context.Context, collectionID, id string) (bool, error) {
	var count int
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		return tx.row(ctx, `SELECT COUNT(*) FROM hydro_collection_records WHERE tenant_id=`+tx.p(1)+` AND collection_id=`+tx.p(2)+` AND id=`+tx.p(3), tenantID, collectionID, id).Scan(&count)
	})
	return count == 1, err
}

// newID returns "r_" and 30 hex digits: 32 characters, the width of the
// MySQL id columns.
func newID() (string, error) {
	raw := make([]byte, 15)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "r_" + hex.EncodeToString(raw), nil
}
func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}
func mapNoRows(err, target error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return target
	}
	return err
}
func isConflict(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unique") || strings.Contains(text, "duplicate")
}
