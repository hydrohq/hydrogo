// Package query defines the bounded, provider-neutral query contract used by
// generated transports. It deliberately produces an AST before any SQL is
// assembled; callers never concatenate untrusted field names or values.
package query

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

const (
	MaxConditions = 32
	MaxValueBytes = 256
	MaxLimit      = 100
	MaxOffset     = 10000
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

type Field struct {
	Name      string
	Operators map[string]bool
}

type Condition struct {
	Field    string
	Operator string
	Values   []string
	Group    string
}

type Sort struct {
	Field string
	Desc  bool
}

type Query struct {
	Conditions []Condition
	Sort       []Sort
	Limit      int
	Offset     int
}

var allowedOperators = map[string]bool{
	"equals": true, "not_equals": true, "contains": true, "starts_with": true,
	"ends_with": true, "in": true, "exists": true, "greater_than": true,
	"less_than": true, "greater_than_equal": true, "less_than_equal": true,
}

func Parse(values url.Values, fields map[string]Field) (Query, error) {
	result := Query{Limit: 25}
	for key, rawValues := range values {
		if key == "limit" || key == "page" || key == "offset" || key == "sort" {
			if err := parseControl(&result, key, rawValues); err != nil {
				return Query{}, err
			}
			continue
		}
		tokens := bracketTokens(key)
		if len(tokens) < 3 || tokens[0] != "where" {
			return Query{}, fmt.Errorf("unsupported query parameter %q", key)
		}
		group := "and"
		fieldToken, operatorToken := tokens[1], tokens[2]
		if len(tokens) == 5 && (tokens[1] == "and" || tokens[1] == "or") {
			group, fieldToken, operatorToken = tokens[1], tokens[3], tokens[4]
		}
		field, ok := fields[fieldToken]
		if !ok || !identifier.MatchString(fieldToken) {
			return Query{}, fmt.Errorf("field %q is not queryable", fieldToken)
		}
		if !allowedOperators[operatorToken] || (len(field.Operators) > 0 && !field.Operators[operatorToken]) {
			return Query{}, fmt.Errorf("operator %q is not allowed for field %q", operatorToken, fieldToken)
		}
		if len(rawValues) == 0 {
			return Query{}, errors.New("query value is required")
		}
		condition := Condition{Field: fieldToken, Operator: operatorToken, Group: group}
		for _, value := range rawValues {
			if len(value) > MaxValueBytes {
				return Query{}, errors.New("query value is too large")
			}
			if operatorToken == "in" {
				for _, item := range strings.Split(value, ",") {
					if strings.TrimSpace(item) == "" {
						return Query{}, errors.New("in query contains an empty value")
					}
					condition.Values = append(condition.Values, strings.TrimSpace(item))
				}
			} else {
				condition.Values = append(condition.Values, value)
			}
		}
		result.Conditions = append(result.Conditions, condition)
		if len(result.Conditions) > MaxConditions {
			return Query{}, errors.New("query condition budget exceeded")
		}
	}
	return result, nil
}

func parseControl(query *Query, key string, values []string) error {
	if len(values) != 1 {
		return fmt.Errorf("query parameter %q must have one value", key)
	}
	switch key {
	case "sort":
		for _, item := range strings.Split(values[0], ",") {
			desc := strings.HasPrefix(item, "-")
			field := strings.TrimPrefix(item, "-")
			if !identifier.MatchString(field) {
				return fmt.Errorf("invalid sort field %q", field)
			}
			query.Sort = append(query.Sort, Sort{Field: field, Desc: desc})
		}
		if len(query.Sort) > 8 {
			return errors.New("sort budget exceeded")
		}
		return nil
	default:
		value, err := strconv.Atoi(values[0])
		if err != nil {
			return fmt.Errorf("invalid %s", key)
		}
		switch key {
		case "limit":
			if value < 1 || value > MaxLimit {
				return fmt.Errorf("limit must be between 1 and %d", MaxLimit)
			}
			query.Limit = value
		case "offset":
			if value < 0 || value > MaxOffset {
				return fmt.Errorf("offset must be between 0 and %d", MaxOffset)
			}
			query.Offset = value
		case "page":
			if value < 1 || value > MaxOffset {
				return errors.New("page is out of bounds")
			}
			query.Offset = (value - 1) * query.Limit
			if query.Offset > MaxOffset {
				return errors.New("page offset budget exceeded")
			}
		}
		return nil
	}
}

func bracketTokens(key string) []string {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "where[") || !strings.HasSuffix(key, "]") {
		return nil
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(key, "where["), "]"), "][")
	return append([]string{"where"}, parts...)
}

type SQL struct {
	Where  string
	Args   []any
	Order  string
	Limit  int
	Offset int
}

func (q Query) BuildSQL(dialect database.Dialect, fields map[string]Field) (SQL, error) {
	if dialect != database.SQLite && dialect != database.Postgres {
		return SQL{}, fmt.Errorf("unsupported query dialect %q", dialect)
	}
	andParts := make([]string, 0, len(q.Conditions))
	orParts := make([]string, 0)
	args := make([]any, 0)
	placeholder := func() string {
		if dialect == database.Postgres {
			return fmt.Sprintf("$%d", len(args)+1)
		}
		return "?"
	}
	for _, condition := range q.Conditions {
		field, ok := fields[condition.Field]
		if !ok || !allowedOperators[condition.Operator] || (len(field.Operators) > 0 && !field.Operators[condition.Operator]) {
			return SQL{}, fmt.Errorf("condition is not allowed for %q", condition.Field)
		}
		if len(condition.Values) == 0 {
			return SQL{}, errors.New("condition has no values")
		}
		column := condition.Field
		var expression string
		switch condition.Operator {
		case "exists":
			expression = column + " IS " + map[bool]string{true: "NOT NULL", false: "NULL"}[condition.Values[0] == "true"]
		case "in":
			markers := make([]string, len(condition.Values))
			for i, value := range condition.Values {
				markers[i] = placeholder()
				args = append(args, value)
			}
			expression = column + " IN (" + strings.Join(markers, ",") + ")"
		default:
			value := condition.Values[0]
			operator := map[string]string{"equals": "=", "not_equals": "<>", "greater_than": ">", "less_than": "<", "greater_than_equal": ">=", "less_than_equal": "<="}[condition.Operator]
			if condition.Operator == "contains" {
				operator, value = "LIKE", "%"+value+"%"
			}
			if condition.Operator == "starts_with" {
				operator, value = "LIKE", value+"%"
			}
			if condition.Operator == "ends_with" {
				operator, value = "LIKE", "%"+value
			}
			if operator == "" {
				return SQL{}, fmt.Errorf("operator %q cannot be built", condition.Operator)
			}
			expression = column + " " + operator + " " + placeholder()
			args = append(args, value)
		}
		if condition.Group == "or" {
			orParts = append(orParts, expression)
		} else {
			andParts = append(andParts, expression)
		}
	}
	result := SQL{Limit: q.Limit, Offset: q.Offset, Args: args}
	if len(orParts) > 0 {
		andParts = append(andParts, "("+strings.Join(orParts, " OR ")+")")
	}
	if len(andParts) > 0 {
		result.Where = strings.Join(andParts, " AND ")
	}
	orders := make([]string, 0, len(q.Sort))
	for _, sort := range q.Sort {
		if field, ok := fields[sort.Field]; !ok || !identifier.MatchString(field.Name) {
			return SQL{}, fmt.Errorf("sort field %q is not allowed", sort.Field)
		} else if sort.Desc {
			orders = append(orders, sort.Field+" DESC")
		} else {
			orders = append(orders, sort.Field+" ASC")
		}
	}
	result.Order = strings.Join(orders, ", ")
	return result, nil
}
