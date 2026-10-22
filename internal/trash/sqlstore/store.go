package sqlstore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
)

type Store struct {
	DB                 database.QueryExecutor
	Dialect            database.Dialect
	RequireTenantScope bool
	Enabled            map[trash.Kind]bool
}

func (s Store) List(ctx context.Context, limit int) ([]trash.Entry, error) {
	if s.DB == nil {
		return nil, database.ErrDatabaseRequired
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	tenantID, scoped, err := s.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	queries := []struct {
		kind  trash.Kind
		table string
		label string
	}{
		{trash.Pages, "pages", "title"},
		{trash.News, "news_items", "title"},
		{trash.Submissions, "form_inbox", "form_key"},
		{trash.Media, "media_assets", "original_filename"},
	}
	items := make([]trash.Entry, 0)
	for _, source := range queries {
		if !s.Enabled[source.kind] {
			continue
		}
		for _, trashed := range []bool{false, true} {
			query := "SELECT id," + source.label + ",deleted_at FROM " + source.table + " WHERE deleted_at IS "
			if trashed {
				query += "NOT NULL"
			} else {
				query += "NULL"
			}
			args := []any{}
			if scoped {
				query += " AND tenant_id=" + s.p(1)
				args = append(args, tenantID)
			}
			if trashed {
				query += " AND deleted_at >= " + s.p(len(args)+1)
				cutoff := time.Now().UTC().Add(-trash.Retention)
				args = append(args, s.dbTimeValue(cutoff))
			}
			query += " ORDER BY created_at DESC, id ASC LIMIT " + s.p(len(args)+1)
			args = append(args, limit)
			rows, err := s.DB.QueryContext(ctx, query, args...)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var item trash.Entry
				var deleted any
				item.Kind, item.Trashed = source.kind, trashed
				if err := rows.Scan(&item.ID, &item.Label, &deleted); err != nil {
					_ = rows.Close()
					return nil, err
				}
				item.Label = safeLabel(source.kind, item.Label)
				item.DeletedAt = parseTime(deleted)
				items = append(items, item)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return nil, err
			}
			_ = rows.Close()
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Trashed != items[j].Trashed {
			return items[i].Trashed
		}
		return items[i].DeletedAt.After(items[j].DeletedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s Store) Move(ctx context.Context, kind trash.Kind, id string, now time.Time) error {
	return s.update(ctx, s.DB, kind, id, `deleted_at=`+s.p(1), now)
}

func (s Store) Restore(ctx context.Context, kind trash.Kind, id string, cutoff time.Time) error {
	return s.update(ctx, s.DB, kind, id, `deleted_at=NULL`, cutoff)
}

func (s Store) MoveWithin(ctx context.Context, tx database.Executor, kind trash.Kind, id string, now time.Time) error {
	return s.update(ctx, tx, kind, id, `deleted_at=`+s.p(1), now)
}

func (s Store) RestoreWithin(ctx context.Context, tx database.Executor, kind trash.Kind, id string, cutoff time.Time) error {
	return s.update(ctx, tx, kind, id, `deleted_at=NULL`, cutoff)
}

func (s Store) update(ctx context.Context, db database.Executor, kind trash.Kind, id, assignment string, stamp time.Time) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	table, ok := tableFor(kind)
	if !ok || !s.Enabled[kind] {
		return trash.ErrNotFound
	}
	query := "UPDATE " + table + " SET " + assignment + " WHERE id=" + s.p(2)
	args := []any{}
	if strings.Contains(assignment, "deleted_at=") && assignment != "deleted_at=NULL" {
		query = "UPDATE " + table + " SET " + assignment + " WHERE id=" + s.p(2) + " AND deleted_at IS NULL"
		args = append(args, s.dbTimeValue(stamp), id)
	} else {
		query = "UPDATE " + table + " SET deleted_at=NULL WHERE id=" + s.p(1) + " AND deleted_at IS NOT NULL AND deleted_at >= " + s.p(2)
		args = append(args, id, s.dbTimeValue(stamp))
	}
	if len(args) == 0 {
		args = append(args, s.dbTimeValue(stamp), id)
	}
	tenantID, scoped, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	if scoped {
		query += " AND tenant_id=" + s.p(len(args)+1)
		args = append(args, tenantID)
	}
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return trash.ErrNotFound
	}
	return nil
}

func (s Store) tenantID(ctx context.Context) (string, bool, error) {
	if !s.RequireTenantScope {
		return "", false, nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", false, err
	}
	return scope.TenantID, true, nil
}

func (s Store) p(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s Store) dbTimeValue(t time.Time) any {
	t = t.UTC()
	if s.Dialect == database.SQLite {
		return t.Format(time.RFC3339Nano)
	}
	return t
}

func tableFor(kind trash.Kind) (string, bool) {
	switch kind {
	case trash.Pages:
		return "pages", true
	case trash.News:
		return "news_items", true
	case trash.Submissions:
		return "form_inbox", true
	case trash.Media:
		return "media_assets", true
	default:
		return "", false
	}
}

func safeLabel(kind trash.Kind, value string) string {
	if kind == trash.Submissions {
		return "Form submission"
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "Untitled"
	}
	if len(value) > 160 {
		return value[:160]
	}
	return value
}

func parseTime(value any) time.Time {
	switch value := value.(type) {
	case time.Time:
		return value.UTC()
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, value); err == nil {
				return parsed.UTC()
			}
		}
	case []byte:
		return parseTime(string(value))
	}
	return time.Time{}
}
