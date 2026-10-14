package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type SearchInsight = analytics.SearchInsight

func (s Store) RecordSearchQuery(ctx context.Context, event analytics.SearchQueryEvent) error {
	if err := s.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	day, err := time.Parse("2006-01-02", event.Day)
	term := analytics.NormalizeSearchTerm(event.Term)
	if err != nil || strings.TrimSpace(event.TenantID) == "" || len(event.TenantID) > 128 || term == "" || event.ResultCount < 0 {
		return errors.New("invalid search analytics event")
	}
	zero := 0
	if event.ResultCount == 0 {
		zero = 1
	}
	date := day.UTC().Format("2006-01-02")
	locale := event.Locale
	if locale == "" {
		locale = "tr"
	}
	if len(locale) > 35 || strings.ContainsAny(locale, "\x00\r\n") {
		return errors.New("invalid search analytics locale")
	}
	query := `INSERT INTO analytics_search_daily(day,tenant_id,term,locale,search_count,zero_result_count) VALUES ($1,$2,$3,$4,1,$5) ON CONFLICT(day,tenant_id,term,locale) DO UPDATE SET search_count=analytics_search_daily.search_count+1,zero_result_count=analytics_search_daily.zero_result_count+EXCLUDED.zero_result_count`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO analytics_search_daily(day,tenant_id,term,locale,search_count,zero_result_count) VALUES (?,?,?,?,1,?) ON CONFLICT(day,tenant_id,term,locale) DO UPDATE SET search_count=search_count+1,zero_result_count=zero_result_count+excluded.zero_result_count`
	}
	if s.Dialect == database.MySQL {
		query = `INSERT INTO analytics_search_daily(day,tenant_id,term,locale,search_count,zero_result_count) VALUES (?,?,?,?,1,?) ON DUPLICATE KEY UPDATE search_count=search_count+1,zero_result_count=zero_result_count+VALUES(zero_result_count)`
	}
	_, err = s.Executor.ExecContext(ctx, query, date, event.TenantID, term, locale, zero)
	return err
}

func (s Store) ListSearchInsights(ctx context.Context, tenantID, fromDay, toDay string, limit int) ([]SearchInsight, error) {
	return s.listSearchInsights(ctx, tenantID, fromDay, toDay, limit, false)
}

func (s Store) ListZeroResultInsights(ctx context.Context, tenantID, fromDay, toDay string, limit int) ([]SearchInsight, error) {
	return s.listSearchInsights(ctx, tenantID, fromDay, toDay, limit, true)
}

func (s Store) listSearchInsights(ctx context.Context, tenantID, fromDay, toDay string, limit int, zeroOnly bool) ([]SearchInsight, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tenantID) == "" || len(tenantID) > 128 {
		return nil, errors.New("invalid search analytics tenant")
	}
	from, fromErr := time.Parse("2006-01-02", fromDay)
	to, toErr := time.Parse("2006-01-02", toDay)
	if fromErr != nil || toErr != nil || from.After(to) {
		return nil, errors.New("invalid search analytics range")
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	queryer, ok := s.Executor.(database.QueryExecutor)
	if !ok {
		return nil, errors.New("search insights require query executor")
	}
	filter := ""
	if zeroOnly {
		filter = " HAVING SUM(zero_result_count)>0"
	}
	query := `SELECT term,locale,SUM(search_count),SUM(zero_result_count) FROM analytics_search_daily WHERE day BETWEEN ` + placeholder(s.Dialect, 1) + ` AND ` + placeholder(s.Dialect, 2) + ` AND tenant_id=` + placeholder(s.Dialect, 3) + ` GROUP BY term,locale` + filter + ` ORDER BY `
	if zeroOnly {
		query += `SUM(zero_result_count) DESC,term,locale LIMIT `
	} else {
		query += `SUM(search_count) DESC,term,locale LIMIT `
	}
	query += fmt.Sprint(limit)
	rows, err := queryer.QueryContext(ctx, query, fromDay, toDay, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SearchInsight, 0)
	for rows.Next() {
		var item SearchInsight
		if err := rows.Scan(&item.Term, &item.Locale, &item.SearchCount, &item.ZeroResultCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
