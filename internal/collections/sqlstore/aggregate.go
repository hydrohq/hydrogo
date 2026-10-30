package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func (s Store) AggregateRecords(ctx context.Context, collectionID string, query collections.AggregateQuery) ([]map[string]any, error) {
	var result []map[string]any
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		statement, args, err := buildAggregateQuery(tx.Dialect, tenantID, collectionID, query)
		if err != nil {
			return err
		}
		rows, err := tx.rows(ctx, statement, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		for rows.Next() {
			scanDest := make([]any, len(cols))
			scanPointers := make([]any, len(cols))
			for i := range scanDest {
				scanPointers[i] = &scanDest[i]
			}
			if err := rows.Scan(scanPointers...); err != nil {
				return err
			}
			rowMap := make(map[string]any, len(cols))
			groupMap := make(map[string]any, len(query.GroupBy))
			for i, col := range cols {
				val := scanDest[i]
				if b, ok := val.([]byte); ok {
					val = string(b)
				}
				isGroup := false
				for _, g := range query.GroupBy {
					if col == g {
						groupMap[g] = val
						isGroup = true
						break
					}
				}
				if !isGroup {
					rowMap[col] = val
				}
			}
			if len(query.GroupBy) > 0 {
				rowMap["group"] = groupMap
			}
			result = append(result, rowMap)
		}
		return rows.Err()
	})
	return result, err
}

func buildAggregateQuery(dialect database.Dialect, tenantID, collectionID string, query collections.AggregateQuery) (string, []any, error) {
	b := &queryBuilder{dialect: dialect}
	var selectParts []string

	// 1. Group By expressions in SELECT
	for _, g := range query.GroupBy {
		if !validFieldID(g) {
			return "", nil, collections.ErrInvalidQuery
		}
		selectParts = append(selectParts, b.text(g)+" AS "+g)
	}

	// 2. Aggregate expressions in SELECT
	for _, agg := range query.Aggregates {
		if agg.Field != "" && !validFieldID(agg.Field) {
			return "", nil, collections.ErrInvalidQuery
		}
		var expr string
		switch agg.Func {
		case collections.AggCount:
			expr = "COUNT(*)"
		case collections.AggSum:
			switch dialect {
			case database.Postgres:
				expr = "COALESCE(SUM((" + b.text(agg.Field) + ")::numeric), 0)"
			case database.MySQL:
				expr = "COALESCE(SUM(CAST(" + b.text(agg.Field) + " AS DECIMAL(20,4))), 0)"
			default: // SQLite
				expr = "COALESCE(SUM(CAST(" + b.text(agg.Field) + " AS REAL)), 0)"
			}
		case collections.AggAvg:
			switch dialect {
			case database.Postgres:
				expr = "COALESCE(AVG((" + b.text(agg.Field) + ")::numeric), 0)"
			case database.MySQL:
				expr = "COALESCE(AVG(CAST(" + b.text(agg.Field) + " AS DECIMAL(20,4))), 0)"
			default: // SQLite
				expr = "COALESCE(AVG(CAST(" + b.text(agg.Field) + " AS REAL)), 0)"
			}
		case collections.AggMin:
			switch dialect {
			case database.Postgres:
				expr = "MIN((" + b.text(agg.Field) + ")::numeric)"
			case database.MySQL:
				expr = "MIN(CAST(" + b.text(agg.Field) + " AS DECIMAL(20,4)))"
			default: // SQLite
				expr = "MIN(CAST(" + b.text(agg.Field) + " AS REAL))"
			}
		case collections.AggMax:
			switch dialect {
			case database.Postgres:
				expr = "MAX((" + b.text(agg.Field) + ")::numeric)"
			case database.MySQL:
				expr = "MAX(CAST(" + b.text(agg.Field) + " AS DECIMAL(20,4)))"
			default: // SQLite
				expr = "MAX(CAST(" + b.text(agg.Field) + " AS REAL))"
			}
		default:
			return "", nil, fmt.Errorf("%w: unknown aggregate function %q", collections.ErrInvalidQuery, agg.Func)
		}
		selectParts = append(selectParts, expr+" AS "+agg.Alias)
	}

	var sql strings.Builder
	sql.WriteString("SELECT " + strings.Join(selectParts, ", ") + " FROM hydro_collection_records WHERE tenant_id = " + b.param(tenantID) + " AND collection_id = " + b.param(collectionID))

	for _, cond := range query.Where {
		clause, err := b.condition(cond)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" AND " + clause)
	}
	if len(query.AnyOf) > 0 {
		clauses := make([]string, 0, len(query.AnyOf))
		for _, condition := range query.AnyOf {
			clause, err := b.condition(condition)
			if err != nil {
				return "", nil, err
			}
			clauses = append(clauses, clause)
		}
		sql.WriteString(" AND (" + strings.Join(clauses, " OR ") + ")")
	}
	if query.AccessRule != "" {
		clause, err := collections.CompileAccessRuleToSQL(b.dialect, query.AccessRule, query.AuthContext, b.param, b.text)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" AND (" + clause + ")")
	}

	if len(query.GroupBy) > 0 {
		groupParts := make([]string, len(query.GroupBy))
		for i, g := range query.GroupBy {
			groupParts[i] = b.text(g)
		}
		sql.WriteString(" GROUP BY " + strings.Join(groupParts, ", "))
		sql.WriteString(" ORDER BY " + strings.Join(groupParts, ", "))
	}

	return sql.String(), b.args, nil
}
