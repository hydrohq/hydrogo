package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Translations live in hydro_collection_translations, one row per record and
// non-default locale, deleted with the record by a cascading foreign key.

func (s Store) GetTranslation(ctx context.Context, collectionID, recordID, locale string) (collections.Translation, error) {
	var result collections.Translation
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		if err := tx.requireRecord(ctx, tenantID, collectionID, recordID); err != nil {
			return err
		}
		result, err = tx.readTranslation(ctx, tenantID, collectionID, recordID, locale)
		return err
	})
	return result, err
}

func (s Store) requireRecord(ctx context.Context, tenantID, collectionID, recordID string) error {
	var count int
	query := `SELECT COUNT(*) FROM hydro_collection_records WHERE tenant_id=` + s.p(1) + ` AND collection_id=` + s.p(2) + ` AND id=` + s.p(3)
	if err := s.row(ctx, query, tenantID, collectionID, recordID).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return collections.ErrRecordNotFound
	}
	return nil
}

// readTranslation returns the stored translation or a zero-version one.
func (s Store) readTranslation(ctx context.Context, tenantID, collectionID, recordID, locale string) (collections.Translation, error) {
	result := collections.Translation{RecordID: recordID, Locale: locale, Data: map[string]any{}}
	query := `SELECT data_json,version,updated_at FROM hydro_collection_translations WHERE tenant_id=` + s.p(1) + ` AND collection_id=` + s.p(2) + ` AND record_id=` + s.p(3) + ` AND locale=` + s.p(4)
	var raw, updated string
	err := s.row(ctx, query, tenantID, collectionID, recordID, locale).Scan(&raw, &result.Version, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return collections.Translation{}, err
	}
	if json.Unmarshal([]byte(raw), &result.Data) != nil || result.Data == nil {
		return collections.Translation{}, errors.New("stored collection translation is invalid")
	}
	result.UpdatedAt = parseTime(updated)
	return result, nil
}

func (s Store) ListTranslations(ctx context.Context, collectionID string, recordIDs []string, locale string) (map[string]collections.Translation, error) {
	result := make(map[string]collections.Translation, len(recordIDs))
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		args := []any{tenantID, collectionID, locale}
		placeholders := make([]string, 0, len(recordIDs))
		for _, id := range recordIDs {
			args = append(args, id)
			placeholders = append(placeholders, tx.p(len(args)))
		}
		query := `SELECT record_id,data_json,version,updated_at FROM hydro_collection_translations WHERE tenant_id=` + tx.p(1) + ` AND collection_id=` + tx.p(2) + ` AND locale=` + tx.p(3) + ` AND record_id IN (` + strings.Join(placeholders, ",") + `)`
		rows, err := tx.rows(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item := collections.Translation{Locale: locale}
			var raw, updated string
			if err := rows.Scan(&item.RecordID, &raw, &item.Version, &updated); err != nil {
				return err
			}
			if json.Unmarshal([]byte(raw), &item.Data) != nil || item.Data == nil {
				return errors.New("stored collection translation is invalid")
			}
			item.UpdatedAt = parseTime(updated)
			result[item.RecordID] = item
		}
		return rows.Err()
	})
	return result, err
}

func (s Store) SaveTranslation(ctx context.Context, collectionID, recordID, locale string, data map[string]any, expectedVersion int) (collections.Translation, error) {
	encoded, err := json.Marshal(data)
	if err != nil || len(encoded) > collections.MaxRecordBytes {
		return collections.Translation{}, collections.ErrInvalidRecord
	}
	var result collections.Translation
	err = s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		if err := tx.requireRecord(ctx, tenantID, collectionID, recordID); err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if expectedVersion == 0 {
			insert := `INSERT INTO hydro_collection_translations (tenant_id,collection_id,record_id,locale,data_json,version,updated_at) VALUES (` + tx.p(1) + `,` + tx.p(2) + `,` + tx.p(3) + `,` + tx.p(4) + `,` + tx.p(5) + `,1,` + tx.p(6) + `)`
			if _, err := tx.exec(ctx, insert, tenantID, collectionID, recordID, locale, string(encoded), now); err != nil {
				if isConflict(err) {
					return collections.ErrVersionConflict
				}
				return err
			}
		} else {
			// The version condition makes the write a compare-and-set.
			update := `UPDATE hydro_collection_translations SET data_json=` + tx.p(1) + `,version=version+1,updated_at=` + tx.p(2) + ` WHERE tenant_id=` + tx.p(3) + ` AND collection_id=` + tx.p(4) + ` AND record_id=` + tx.p(5) + ` AND locale=` + tx.p(6) + ` AND version=` + tx.p(7)
			updated, err := tx.exec(ctx, update, string(encoded), now, tenantID, collectionID, recordID, locale, expectedVersion)
			if err != nil {
				return err
			}
			if n, _ := updated.RowsAffected(); n == 0 {
				return collections.ErrVersionConflict
			}
		}
		result = collections.Translation{RecordID: recordID, Locale: locale, Data: data, Version: expectedVersion + 1, UpdatedAt: parseTime(now)}
		return nil
	})
	return result, err
}
