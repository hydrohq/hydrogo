package sqlstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// writeRevision stores the snapshot of a just-saved version in the same
// transaction as the record and drops revisions beyond the per-record cap.
func (s Store) writeRevision(ctx context.Context, tenantID, collectionID, recordID string, version int, encoded, now string) error {
	insert := `INSERT INTO hydro_collection_revisions (tenant_id,collection_id,record_id,version,data_json,actor,created_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `)`
	if _, err := s.exec(ctx, insert, tenantID, collectionID, recordID, version, encoded, collections.ActorFromContext(ctx), now); err != nil {
		return err
	}
	oldest := version - collections.MaxRevisionsPerRecord
	if oldest < 1 {
		return nil
	}
	prune := `DELETE FROM hydro_collection_revisions WHERE tenant_id=` + s.p(1) + ` AND collection_id=` + s.p(2) + ` AND record_id=` + s.p(3) + ` AND version<=` + s.p(4)
	_, err := s.exec(ctx, prune, tenantID, collectionID, recordID, oldest)
	return err
}

func (s Store) ListRevisions(ctx context.Context, collectionID, recordID string, limit int) ([]collections.Revision, error) {
	var result []collections.Revision
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := `SELECT version,data_json,actor,created_at FROM hydro_collection_revisions WHERE tenant_id=` + tx.p(1) + ` AND collection_id=` + tx.p(2) + ` AND record_id=` + tx.p(3) + ` ORDER BY version DESC LIMIT ` + tx.p(4)
		rows, err := tx.rows(ctx, query, tenantID, collectionID, recordID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		result = make([]collections.Revision, 0, limit)
		for rows.Next() {
			item := collections.Revision{RecordID: recordID}
			var raw, created string
			if err := rows.Scan(&item.Version, &raw, &item.Actor, &created); err != nil {
				return err
			}
			if json.Unmarshal([]byte(raw), &item.Data) != nil {
				return errors.New("stored collection revision is invalid")
			}
			item.CreatedAt = parseTime(created)
			result = append(result, item)
		}
		return rows.Err()
	})
	return result, err
}

func (s Store) GetRevision(ctx context.Context, collectionID, recordID string, version int) (collections.Revision, error) {
	var result collections.Revision
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := `SELECT data_json,actor,created_at FROM hydro_collection_revisions WHERE tenant_id=` + tx.p(1) + ` AND collection_id=` + tx.p(2) + ` AND record_id=` + tx.p(3) + ` AND version=` + tx.p(4)
		var raw, created string
		result = collections.Revision{RecordID: recordID, Version: version}
		if err := tx.row(ctx, query, tenantID, collectionID, recordID, version).Scan(&raw, &result.Actor, &created); err != nil {
			return mapNoRows(err, collections.ErrRevisionNotFound)
		}
		if json.Unmarshal([]byte(raw), &result.Data) != nil {
			return errors.New("stored collection revision is invalid")
		}
		result.CreatedAt = parseTime(created)
		return nil
	})
	return result, err
}
