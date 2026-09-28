package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/portfolio"
)

type Store struct {
	sectoral.Store[portfolio.Project]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[portfolio.Project]{Dialect: dialect, Table: "sectoral_portfolio"}}
}

func (s Store) Create(ctx context.Context, db database.Executor, item portfolio.Project) error {
	if err := s.Store.Create(ctx, db, item); err != nil {
		return err
	}
	return s.replaceTags(ctx, db, item)
}

func (s Store) Update(ctx context.Context, db database.Executor, item portfolio.Project) error {
	if err := s.Store.Update(ctx, db, item); err != nil {
		return err
	}
	return s.replaceTags(ctx, db, item)
}

func (s Store) replaceTags(ctx context.Context, db database.Executor, item portfolio.Project) error {
	base := item.SectoralBase()
	placeholder := func(n int) string {
		if s.Store.Dialect == database.Postgres {
			return "$" + strconv.Itoa(n)
		}
		return "?"
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM sectoral_portfolio_tags WHERE tenant_id="+placeholder(1)+" AND content_id="+placeholder(2), sectoral.TenantID(ctx), base.ID); err != nil {
		return err
	}
	terms := make([]string, 0, len(base.Tags)+1)
	for _, tag := range base.Tags {
		tag = "tag:" + strings.ToLower(strings.TrimSpace(tag))
		if tag != "tag:" {
			terms = append(terms, tag)
		}
	}
	if industry := strings.ToLower(strings.TrimSpace(item.Industry)); industry != "" {
		terms = append(terms, "category:"+industry)
	}
	for _, tag := range terms {
		if _, err := db.ExecContext(ctx, "INSERT INTO sectoral_portfolio_tags (tenant_id,content_id,tag) VALUES ("+placeholder(1)+","+placeholder(2)+","+placeholder(3)+")", sectoral.TenantID(ctx), base.ID, tag); err != nil {
			return err
		}
	}
	return nil
}

// ListRelatedPublished returns up to ten published projects with the greatest
// number of shared tags. The composite lookup index bounds relation access.
func (s Store) ListRelatedPublished(ctx context.Context, db database.QueryExecutor, sourceID, locale string, limit int) ([]portfolio.Project, error) {
	if database.RequireExecutor(db) != nil || limit < 1 || limit > 10 {
		return nil, sectoral.ErrInvalid
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("portfolio related query requires row queries")
	}
	b1, b2, b3, b4, b5 := "?", "?", "?", "?", "?"
	if s.Store.Dialect == database.Postgres {
		b1, b2, b3, b4, b5 = "$1", "$2", "$3", "$1", "$4"
	}
	query := `SELECT p.payload FROM sectoral_portfolio p JOIN sectoral_portfolio_tags candidate ON candidate.tenant_id=p.tenant_id AND candidate.content_id=p.id JOIN sectoral_portfolio_tags source ON source.tenant_id=candidate.tenant_id AND source.tag=candidate.tag AND source.content_id=` + b1 + ` WHERE p.tenant_id=` + b2 + ` AND p.locale=` + b3 + ` AND p.status='published' AND p.id<>` + b4 + ` GROUP BY p.id,p.payload,p.updated_at ORDER BY COUNT(source.tag) DESC,p.updated_at DESC,p.id ASC LIMIT ` + b5
	args := []any{sourceID, sectoral.TenantID(ctx), locale, sourceID, limit}
	if s.Store.Dialect == database.Postgres {
		args = []any{sourceID, sectoral.TenantID(ctx), locale, limit}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]portfolio.Project, 0, limit)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var item portfolio.Project
		if err := json.Unmarshal([]byte(payload), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
