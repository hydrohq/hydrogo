package v1

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxAdminViewName    = 64
	MaxAdminViewColumns = 64
	MaxAdminViewFilters = 32
)

var adminViewIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{0,63}$`)

type AdminViewFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
}

type AdminView struct {
	ID      string
	Name    string
	Columns []string
	Filters []AdminViewFilter
	Sort    []QuerySort
}

// ValidateAdminView validates user-saved list metadata without evaluating a
// query or granting resource access. The caller must still run filters through
// ParseResourceQuery and the resource policy before executing them.
func ValidateAdminView(view AdminView, fields map[string]QueryField) (AdminView, error) {
	view.ID = strings.TrimSpace(view.ID)
	view.Name = strings.TrimSpace(view.Name)
	if !adminViewIdentifier.MatchString(view.ID) || len(view.Name) == 0 || len(view.Name) > MaxAdminViewName {
		return AdminView{}, errors.New("invalid admin view identity")
	}
	if len(view.Columns) > MaxAdminViewColumns || len(view.Filters) > MaxAdminViewFilters {
		return AdminView{}, errors.New("admin view budget exceeded")
	}
	seen := make(map[string]struct{}, len(view.Columns))
	for i, column := range view.Columns {
		column = strings.TrimSpace(column)
		if _, ok := fields[column]; !ok || !adminViewIdentifier.MatchString(column) {
			return AdminView{}, fmt.Errorf("admin view column %q is not allowed", column)
		}
		if _, ok := seen[column]; ok {
			return AdminView{}, fmt.Errorf("duplicate admin view column %q", column)
		}
		seen[column] = struct{}{}
		view.Columns[i] = column
	}
	for i, filter := range view.Filters {
		filter.Field = strings.TrimSpace(filter.Field)
		filter.Operator = strings.TrimSpace(filter.Operator)
		if _, ok := fields[filter.Field]; !ok || !adminViewIdentifier.MatchString(filter.Field) || len(filter.Value) > MaxQueryValueBytes {
			return AdminView{}, fmt.Errorf("admin view filter %q is not allowed", filter.Field)
		}
		field := fields[filter.Field]
		if !queryOperators[filter.Operator] || (len(field.Operators) > 0 && !field.Operators[filter.Operator]) {
			return AdminView{}, fmt.Errorf("admin view operator %q is not allowed", filter.Operator)
		}
		view.Filters[i] = filter
	}
	for _, item := range view.Sort {
		if _, ok := fields[item.Field]; !ok {
			return AdminView{}, fmt.Errorf("admin view sort %q is not allowed", item.Field)
		}
	}
	return view, nil
}
