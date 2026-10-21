package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// NextSequence implements collections.SequenceRepository. The UPDATE locks the
// counter row until the surrounding transaction ends, so concurrent creates
// queue up and numbers are handed out in commit order without gaps.
func (s Store) NextSequence(ctx context.Context, collectionID, fieldID, scope string) (int64, error) {
	tenantID, err := s.scope(ctx)
	if err != nil {
		return 0, err
	}
	var insert string
	switch s.Dialect {
	case database.Postgres:
		insert = `INSERT INTO hydro_collection_sequences (tenant_id,collection_id,field_id,scope_key,last_value) VALUES ($1,$2,$3,$4,0) ON CONFLICT DO NOTHING`
	case database.MySQL:
		insert = `INSERT IGNORE INTO hydro_collection_sequences (tenant_id,collection_id,field_id,scope_key,last_value) VALUES (?,?,?,?,0)`
	default:
		insert = `INSERT OR IGNORE INTO hydro_collection_sequences (tenant_id,collection_id,field_id,scope_key,last_value) VALUES (?,?,?,?,0)`
	}
	if _, err := s.exec(ctx, insert, tenantID, collectionID, fieldID, scope); err != nil {
		return 0, err
	}
	where := ` WHERE tenant_id=` + s.p(1) + ` AND collection_id=` + s.p(2) + ` AND field_id=` + s.p(3) + ` AND scope_key=` + s.p(4)
	if _, err := s.exec(ctx, `UPDATE hydro_collection_sequences SET last_value=last_value+1`+where, tenantID, collectionID, fieldID, scope); err != nil {
		return 0, err
	}
	var next int64
	if err := s.row(ctx, `SELECT last_value FROM hydro_collection_sequences`+where, tenantID, collectionID, fieldID, scope).Scan(&next); err != nil {
		return 0, fmt.Errorf("read sequence: %w", err)
	}
	return next, nil
}
