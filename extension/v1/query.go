// Package v1 contains provider-neutral contracts that generated Client
// Editions may import without reaching into HydroGo internals.
package v1

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
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
	for key, raw := range values {
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
