package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/downloads"
)

type Store struct {
	base    sectoral.Store[downloads.Download]
	Dialect database.Dialect
}

func New(dialect database.Dialect) Store {
	return Store{Dialect: dialect, base: sectoral.Store[downloads.Download]{Dialect: dialect, Table: "sectoral_downloads"}}
}

func (s Store) bind(index int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func (s Store) Create(ctx context.Context, db database.Executor, item downloads.Download) error {
	item.DownloadCount = 0
	return s.base.Create(ctx, db, item)
}

func (s Store) Update(ctx context.Context, db database.Executor, item downloads.Download) error {
	count, err := s.count(ctx, db, item.ID)
	if err != nil {
		return err
	}
	item.DownloadCount = count
	return s.base.Update(ctx, db, item)
}

func (s Store) Get(ctx context.Context, db database.Executor, id string) (downloads.Download, error) {
	item, err := s.base.Get(ctx, db, id)
	if err != nil {
		return downloads.Download{}, err
	}
	item.DownloadCount, err = s.count(ctx, db, id)
	return item, err
}

func (s Store) List(ctx context.Context, db database.QueryExecutor, limit int) ([]downloads.Download, error) {
	items, err := s.base.List(ctx, db, limit)
	if err != nil {
		return nil, err
	}
	return s.withCounts(ctx, db, items)
}

func (s Store) ListPublished(ctx context.Context, db database.QueryExecutor, locale string, limit int) ([]downloads.Download, error) {
	items, err := s.base.ListPublished(ctx, db, locale, limit)
	if err != nil {
		return nil, err
	}
	return s.withCounts(ctx, db, items)
}

func (s Store) Delete(ctx context.Context, db database.Executor, id string) error {
	return s.base.Delete(ctx, db, id)
}

func (s Store) IncrementDownload(ctx context.Context, db database.Executor, id string) (int64, error) {
	if err := database.RequireExecutor(db); err != nil {
		return 0, err
	}
	query := "UPDATE sectoral_downloads SET download_count=download_count+1 WHERE id=" + s.bind(1) + " AND tenant_id=" + s.bind(2) + " RETURNING download_count"
	var count int64
	if err := db.QueryRowContext(ctx, query, id, sectoral.TenantID(ctx)).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s Store) count(ctx context.Context, db database.Executor, id string) (int64, error) {
	var count int64
	err := db.QueryRowContext(ctx, "SELECT download_count FROM sectoral_downloads WHERE id="+s.bind(1)+" AND tenant_id="+s.bind(2), id, sectoral.TenantID(ctx)).Scan(&count)
	return count, err
}

func (s Store) withCounts(ctx context.Context, db database.QueryExecutor, items []downloads.Download) ([]downloads.Download, error) {
	if len(items) == 0 {
		return items, nil
	}
	args := []any{sectoral.TenantID(ctx)}
	query := "SELECT id,download_count FROM sectoral_downloads WHERE tenant_id=" + s.bind(1) + " AND id IN ("
	for i, item := range items {
		if i > 0 {
			query += ","
		}
		args = append(args, item.ID)
		query += s.bind(len(args))
	}
	rows, err := db.QueryContext(ctx, query+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int64, len(items))
	for rows.Next() {
		var id string
		var count int64
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].DownloadCount = counts[items[i].ID]
	}
	return items, nil
}
