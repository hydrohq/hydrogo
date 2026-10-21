package sqlstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// likeEscape is the LIKE escape character. '!' needs no escaping in string
// literals of any supported dialect (MySQL treats a backslash specially).
const likeEscape = "!"

// QueryRecords lists one page of records matching a query that
// collections.Schema.ValidateQuery has already checked.
func (s Store) QueryRecords(ctx context.Context, collectionID string, query collections.Query) ([]collections.Record, error) {
	var result []collections.Record
	err := s.within(ctx, func(tx Store) error {
		tenantID, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		statement, args, err := buildRecordQuery(tx.Dialect, tenantID, collectionID, query)
		if err != nil {
			return err
		}
		rows, err := tx.rows(ctx, statement, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		result = make([]collections.Record, 0, query.Limit)
		for rows.Next() {
			item := collections.Record{CollectionID: collectionID}
			var raw, created, updated string
			if err := rows.Scan(&item.ID, &raw, &item.Version, &created, &updated); err != nil {
				return err
			}
			if json.Unmarshal([]byte(raw), &item.Data) != nil {
				return errors.New("stored collection record is invalid")
			}
			item.CreatedAt = parseTime(created)
			item.UpdatedAt = parseTime(updated)
			result = append(result, item)
		}
		return rows.Err()
	})
	return result, err
}

// queryBuilder collects bound arguments and renders the dialect's
// placeholders. Field IDs only ever travel as arguments (JSON paths or
// PostgreSQL keys), never as SQL text.
type queryBuilder struct {
	dialect database.Dialect
	args    []any
}

func (b *queryBuilder) param(value any) string {
	b.args = append(b.args, value)
	if b.dialect == database.Postgres {
		return fmt.Sprintf("$%d", len(b.args))
	}
	return "?"
}

// text is the field's value as SQL text (strings unquoted).
func (b *queryBuilder) text(field string) string {
	switch b.dialect {
	case database.Postgres:
		return "(data_json::jsonb ->> " + b.param(field) + "::text)"
	case database.MySQL:
		return "JSON_UNQUOTE(JSON_EXTRACT(data_json, " + b.param("$."+field) + "))"
	default:
		return "json_extract(data_json, " + b.param("$."+field) + ")"
	}
}

// value is the field's value with JSON typing kept, for numbers and booleans.
func (b *queryBuilder) value(field string) string {
	switch b.dialect {
	case database.Postgres:
		return "(data_json::jsonb -> " + b.param(field) + "::text)"
	case database.MySQL:
		return "JSON_EXTRACT(data_json, " + b.param("$."+field) + ")"
	default:
		return "json_extract(data_json, " + b.param("$."+field) + ")"
	}
}

// literal binds a typed comparison value for value().
func (b *queryBuilder) literal(value any) string {
	switch b.dialect {
	case database.Postgres:
		encoded, _ := json.Marshal(value)
		return b.param(string(encoded)) + "::jsonb"
	case database.MySQL:
		encoded, _ := json.Marshal(value)
		return "CAST(" + b.param(string(encoded)) + " AS JSON)"
	default:
		if flag, ok := value.(bool); ok {
			if flag {
				return b.param(1)
			}
			return b.param(0)
		}
		return b.param(value)
	}
}

func (b *queryBuilder) condition(condition collections.Condition) (string, error) {
	if !validFieldID(condition.Field) {
		return "", collections.ErrInvalidQuery
	}
	switch condition.Op {
	case collections.Contains:
		text, ok := condition.Typed.(string)
		if !ok {
			return "", collections.ErrInvalidQuery
		}
		pattern := "%" + escapeLike(strings.ToLower(text)) + "%"
		return "LOWER(" + b.text(condition.Field) + ") LIKE " + b.param(pattern) + " ESCAPE '" + likeEscape + "'", nil
	case collections.Equals:
		return b.compare(condition.Field, "=", condition.Typed)
	case collections.NotEquals:
		// Bind in the order the placeholders appear in the SQL text.
		missing := b.text(condition.Field) + " IS NULL"
		clause, err := b.compare(condition.Field, "<>", condition.Typed)
		if err != nil {
			return "", err
		}
		return "(" + missing + " OR " + clause + ")", nil
	case collections.GreaterThan:
		return b.compare(condition.Field, ">", condition.Typed)
	case collections.GreaterOrEqual:
		return b.compare(condition.Field, ">=", condition.Typed)
	case collections.LessThan:
		return b.compare(condition.Field, "<", condition.Typed)
	case collections.LessOrEqual:
		return b.compare(condition.Field, "<=", condition.Typed)
	case collections.IsNull:
		flag, ok := condition.Typed.(bool)
		if !ok {
			return "", collections.ErrInvalidQuery
		}
		if flag {
			return b.text(condition.Field) + " IS NULL", nil
		}
		return b.text(condition.Field) + " IS NOT NULL", nil
	case collections.In:
		values, ok := condition.Typed.([]any)
		if !ok || len(values) == 0 || len(values) > collections.MaxInValues {
			return "", collections.ErrInvalidQuery
		}
		var left string
		placeholders := make([]string, 0, len(values))
		switch values[0].(type) {
		case string:
			left = b.text(condition.Field)
			for _, value := range values {
				text, ok := value.(string)
				if !ok {
					return "", collections.ErrInvalidQuery
				}
				placeholders = append(placeholders, b.param(text))
			}
		case float64, bool:
			left = b.value(condition.Field)
			for _, value := range values {
				placeholders = append(placeholders, b.literal(value))
			}
		default:
			return "", collections.ErrInvalidQuery
		}
		return left + " IN (" + strings.Join(placeholders, ",") + ")", nil
	}
	return "", collections.ErrInvalidQuery
}

// compare renders "field <op> value" for a validated string, number or
// boolean value.
func (b *queryBuilder) compare(field, op string, typed any) (string, error) {
	switch value := typed.(type) {
	case string:
		return b.text(field) + " " + op + " " + b.param(value), nil
	case float64, bool:
		return b.value(field) + " " + op + " " + b.literal(value), nil
	}
	return "", collections.ErrInvalidQuery
}

func (b *queryBuilder) order(sort *collections.Sort) (string, error) {
	if sort == nil {
		return "updated_at DESC,id ASC", nil
	}
	direction := " ASC"
	if sort.Desc {
		direction = " DESC"
	}
	var expression string
	switch {
	case sort.Kind == "" && sort.Field == collections.SortCreatedAt:
		expression = "created_at"
	case sort.Kind == "" && sort.Field == collections.SortUpdatedAt:
		expression = "updated_at"
	case !validFieldID(sort.Field):
		return "", collections.ErrInvalidQuery
	case sort.Kind == collections.Number || sort.Kind == collections.Boolean:
		expression = b.value(sort.Field)
	case sort.Kind == collections.DateTime:
		expression = b.text(sort.Field)
	case sort.Kind == "":
		return "", collections.ErrInvalidQuery
	default:
		expression = "LOWER(" + b.text(sort.Field) + ")"
	}
	return expression + direction + ",id ASC", nil
}

// buildRecordQuery renders the SELECT for one page of a collection.
func buildRecordQuery(dialect database.Dialect, tenantID, collectionID string, query collections.Query) (string, []any, error) {
	if query.Limit < 1 || query.Limit > 100 || query.Offset < 0 || len(query.Where)+len(query.AnyOf) > collections.MaxQueryConditions {
		return "", nil, collections.ErrInvalidQuery
	}
	b := &queryBuilder{dialect: dialect}
	var sql strings.Builder
	sql.WriteString("SELECT id,data_json,version,created_at,updated_at FROM hydro_collection_records WHERE tenant_id=")
	sql.WriteString(b.param(tenantID))
	sql.WriteString(" AND collection_id=")
	sql.WriteString(b.param(collectionID))
	for _, condition := range query.Where {
		clause, err := b.condition(condition)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" AND " + clause)
	}
	if after := query.After; after != nil {
		if (after.Field != collections.SortCreatedAt && after.Field != collections.SortUpdatedAt) || len(after.Value) == 0 || len(after.ID) == 0 {
			return "", nil, collections.ErrInvalidQuery
		}
		// Same order as the ORDER BY: the column (direction as sorted), then
		// id ascending. The column name comes from the allowlist above.
		compare := ">"
		if after.Desc {
			compare = "<"
		}
		sql.WriteString(" AND (" + after.Field + " " + compare + " " + b.param(after.Value) +
			" OR (" + after.Field + " = " + b.param(after.Value) + " AND id > " + b.param(after.ID) + "))")
	}
	if len(query.IDs) > 0 {
		if len(query.IDs) > collections.MaxQueryIDs {
			return "", nil, collections.ErrInvalidQuery
		}
		placeholders := make([]string, 0, len(query.IDs))
		for _, id := range query.IDs {
			placeholders = append(placeholders, b.param(id))
		}
		sql.WriteString(" AND id IN (" + strings.Join(placeholders, ",") + ")")
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
	order, err := b.order(query.Sort)
	if err != nil {
		return "", nil, err
	}
	sql.WriteString(" ORDER BY " + order)
	sql.WriteString(" LIMIT " + b.param(query.Limit) + " OFFSET " + b.param(query.Offset))
	return sql.String(), b.args, nil
}

// validFieldID repeats the schema identifier rule as a second line of
// defence: a field ID is only ever a bound argument, but it must still be a
// plain identifier so it cannot change the meaning of a JSON path.
func validFieldID(id string) bool {
	if id == "" || len(id) > 63 || id[0] < 'a' || id[0] > 'z' {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(likeEscape, likeEscape+likeEscape, "%", likeEscape+"%", "_", likeEscape+"_")
	return replacer.Replace(value)
}
