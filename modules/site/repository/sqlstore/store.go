package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) p(n int) string {
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
		return "?"
	}
	return "$" + strconv.Itoa(n)
}

func (s Store) tenantFilter(ctx context.Context, position int) (string, any, error) {
	if !s.RequireTenantScope {
		return " AND tenant_id = " + s.p(position), "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", nil, err
	}
	return " AND tenant_id = " + s.p(position), scope.TenantID, nil
}

func parseBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int64:
		return value != 0
	case int:
		return value != 0
	case []byte:
		return string(value) == "1" || strings.EqualFold(string(value), "true")
	case string:
		return value == "1" || strings.EqualFold(value, "true")
	default:
		return false
	}
}

type queryExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryContext(ctx context.Context, db database.Executor, query string, args ...any) (*sql.Rows, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(queryExecutor)
	if !ok {
		return nil, fmt.Errorf("database executor does not support row queries")
	}
	return queryer.QueryContext(ctx, query, args...)
}

func parseTime(value any) time.Time {
	switch value := value.(type) {
	case time.Time:
		return value.UTC()
	case string:
		parsed, _ := time.Parse(time.RFC3339Nano, value)
		return parsed.UTC()
	case []byte:
		parsed, _ := time.Parse(time.RFC3339Nano, string(value))
		return parsed.UTC()
	default:
		return time.Time{}
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
