// Package v1 contains provider-neutral contracts that generated Client
// Editions may import without reaching into HydroGo internals.
package v1

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	MaxQueryConditions = 32
	MaxQueryValueBytes = 256
	MaxQueryLimit      = 100
	MaxQueryOffset     = 10000
)

var queryIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

type QueryField struct {
	Name      string
	Operators map[string]bool
}

type QueryCondition struct {
	Field    string
	Operator string
	Values   []string
	Group    string
}

type QuerySort struct {
	Field string
	Desc  bool
}

type ResourceQuery struct {
	Conditions []QueryCondition
	Sort       []QuerySort
	Limit      int
	Offset     int
}

type QuerySQL struct {
	Where  string
	Args   []any
	Order  string
	Limit  int
	Offset int
}

var queryOperators = map[string]bool{
	"equals": true, "not_equals": true, "contains": true,
	"starts_with": true, "ends_with": true, "in": true, "exists": true,
	"greater_than": true, "less_than": true,
	"greater_than_equal": true, "less_than_equal": true,
}

// ParseResourceQuery parses the bounded where[field][operator] contract.
// Field names and operators must be supplied by the generated resource.
func ParseResourceQuery(values url.Values, fields map[string]QueryField) (ResourceQuery, error) {
	result := ResourceQuery{Limit: 25}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		raw := values[key]
		if key == "limit" || key == "page" || key == "offset" || key == "sort" {
			if err := parseQueryControl(&result, key, raw); err != nil {
				return ResourceQuery{}, err
			}
			continue
		}
		tokens := queryBracketTokens(key)
		if len(tokens) < 3 || tokens[0] != "where" {
			return ResourceQuery{}, fmt.Errorf("unsupported query parameter %q", key)
		}
		group, fieldName, operator := "and", tokens[1], tokens[2]
		if len(tokens) == 5 && (tokens[1] == "and" || tokens[1] == "or") {
			group, fieldName, operator = tokens[1], tokens[3], tokens[4]
		}
		field, ok := fields[fieldName]
		if !ok || !queryIdentifier.MatchString(fieldName) {
			return ResourceQuery{}, fmt.Errorf("field %q is not queryable", fieldName)
		}
		if !queryOperators[operator] || (len(field.Operators) > 0 && !field.Operators[operator]) {
			return ResourceQuery{}, fmt.Errorf("operator %q is not allowed", operator)
		}
		if len(raw) == 0 {
			return ResourceQuery{}, errors.New("query value is required")
		}
		condition := QueryCondition{Field: fieldName, Operator: operator, Group: group}
		for _, value := range raw {
			if len(value) > MaxQueryValueBytes {
				return ResourceQuery{}, errors.New("query value is too large")
			}
			if operator == "in" {
				for _, item := range strings.Split(value, ",") {
					item = strings.TrimSpace(item)
					if item == "" {
						return ResourceQuery{}, errors.New("in query contains an empty value")
					}
					condition.Values = append(condition.Values, item)
				}
			} else {
				condition.Values = append(condition.Values, value)
			}
		}
		result.Conditions = append(result.Conditions, condition)
		if len(result.Conditions) > MaxQueryConditions {
			return ResourceQuery{}, errors.New("query condition budget exceeded")
		}
	}
	return result, nil
}

func parseQueryControl(query *ResourceQuery, key string, values []string) error {
	if len(values) != 1 {
		return fmt.Errorf("query parameter %q must have one value", key)
	}
	if key == "sort" {
		for _, item := range strings.Split(values[0], ",") {
			desc := strings.HasPrefix(item, "-")
			field := strings.TrimPrefix(item, "-")
			if !queryIdentifier.MatchString(field) {
				return fmt.Errorf("invalid sort field %q", field)
			}
			query.Sort = append(query.Sort, QuerySort{Field: field, Desc: desc})
		}
		if len(query.Sort) > 8 {
			return errors.New("sort budget exceeded")
		}
		return nil
	}
	value, err := strconv.Atoi(values[0])
	if err != nil {
		return fmt.Errorf("invalid %s", key)
	}
	switch key {
	case "limit":
		if value < 1 || value > MaxQueryLimit {
			return fmt.Errorf("limit must be between 1 and %d", MaxQueryLimit)
		}
		query.Limit = value
	case "offset":
		if value < 0 || value > MaxQueryOffset {
			return fmt.Errorf("offset is out of bounds")
		}
		query.Offset = value
	case "page":
		if value < 1 || value > MaxQueryOffset {
			return errors.New("page is out of bounds")
		}
		query.Offset = (value - 1) * query.Limit
		if query.Offset > MaxQueryOffset {
			return errors.New("page offset budget exceeded")
		}
	}
	return nil
}

func queryBracketTokens(key string) []string {
	if !strings.HasPrefix(key, "where[") || !strings.HasSuffix(key, "]") {
		return nil
	}
	return append([]string{"where"}, strings.Split(strings.TrimSuffix(strings.TrimPrefix(key, "where["), "]"), "][")...)
}

// BuildSQL turns the validated AST into dialect-specific placeholders. Field
// identifiers are accepted only from the generated allowlist; values are
// always returned as bound arguments.
func (q ResourceQuery) BuildSQL(dialect string, fields map[string]QueryField) (QuerySQL, error) {
	if dialect != "sqlite" && dialect != "postgres" {
		return QuerySQL{}, fmt.Errorf("unsupported query dialect %q", dialect)
	}
	args := make([]any, 0)
	placeholder := func() string {
		if dialect == "postgres" {
			return fmt.Sprintf("$%d", len(args)+1)
		}
		return "?"
	}
	andParts, orParts := make([]string, 0), make([]string, 0)
	for _, condition := range q.Conditions {
		field, ok := fields[condition.Field]
		if !ok || !queryOperators[condition.Operator] || (len(field.Operators) > 0 && !field.Operators[condition.Operator]) {
			return QuerySQL{}, fmt.Errorf("condition is not allowed for %q", condition.Field)
		}
		if len(condition.Values) == 0 {
			return QuerySQL{}, errors.New("condition has no values")
		}
		var expression string
		switch condition.Operator {
		case "exists":
			if condition.Values[0] != "true" && condition.Values[0] != "false" {
				return QuerySQL{}, errors.New("exists requires a boolean")
			}
			if condition.Values[0] == "true" {
				expression = condition.Field + " IS NOT NULL"
			} else {
				expression = condition.Field + " IS NULL"
			}
		case "in":
			markers := make([]string, len(condition.Values))
			for i, value := range condition.Values {
				markers[i] = placeholder()
				args = append(args, value)
			}
			expression = condition.Field + " IN (" + strings.Join(markers, ",") + ")"
		default:
			value := condition.Values[0]
			operator := map[string]string{"equals": "=", "not_equals": "<>", "greater_than": ">", "less_than": "<", "greater_than_equal": ">=", "less_than_equal": "="}[condition.Operator]
			if condition.Operator == "less_than_equal" {
				operator = "<="
			}
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
				return QuerySQL{}, fmt.Errorf("operator %q cannot be built", condition.Operator)
			}
			expression = condition.Field + " " + operator + " " + placeholder()
			args = append(args, value)
		}
		if condition.Group == "or" {
			orParts = append(orParts, expression)
		} else {
			andParts = append(andParts, expression)
		}
	}
	if len(orParts) > 0 {
		andParts = append(andParts, "("+strings.Join(orParts, " OR ")+")")
	}
	orders := make([]string, 0, len(q.Sort))
	for _, item := range q.Sort {
		field, ok := fields[item.Field]
		if !ok || !queryIdentifier.MatchString(field.Name) {
			return QuerySQL{}, fmt.Errorf("sort field %q is not allowed", item.Field)
		}
		direction := "ASC"
		if item.Desc {
			direction = "DESC"
		}
		orders = append(orders, item.Field+" "+direction)
	}
	return QuerySQL{Where: strings.Join(andParts, " AND "), Args: args, Order: strings.Join(orders, ", "), Limit: q.Limit, Offset: q.Offset}, nil
}
