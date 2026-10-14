package adminstudio

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func parseAdminField(field Field, values []string) (any, error) {
	if field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
		limit := field.Relation.MaxItems
		if limit == 0 {
			limit = 100
		}
		if len(values) > limit {
			return nil, errors.New("too many related records")
		}
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return nil, errors.New("empty related record")
			}
			if _, exists := seen[value]; exists {
				return nil, errors.New("duplicate related record")
			}
			seen[value] = struct{}{}
		}
		return append([]string(nil), values...), nil
	}
	if len(values) != 1 {
		return nil, errors.New("multiple field values")
	}
	value := values[0]
	switch field.Kind {
	case FieldNumber:
		// A blank number is "no value"; the API decides whether it is required.
		if strings.TrimSpace(value) == "" {
			return nil, nil
		}
		return strconv.ParseFloat(value, 64)
	case FieldBoolean:
		return value == "true" || value == "on" || value == "1", nil
	case FieldArray:
		if value == "" {
			return []string{}, nil
		}
		items := strings.Split(value, "\n")
		result := make([]string, 0, len(items))
		for _, item := range items {
			if item = strings.TrimSpace(item); item != "" {
				result = append(result, item)
			}
		}
		return result, nil
	default:
		return value, nil
	}
}

// DisplayValue renders a record value as generated Admin lists, forms and
// revision diffs show it; revision callbacks use it so snapshots compare
// equal to the current record.
func DisplayValue(value any) string { return displayValue(value) }

func displayValue(value any) string {
	if value == nil {
		return ""
	}
	switch value := value.(type) {
	case []string:
		return strings.Join(value, ",")
	case []any:
		items := make([]string, 0, len(value))
		for _, item := range value {
			items = append(items, fmt.Sprint(item))
		}
		return strings.Join(items, ",")
	}
	return fmt.Sprint(value)
}

func displayFormValue(field Field, value any) string {
	if field.Kind == FieldArray {
		switch items := value.(type) {
		case []string, []any:
			encoded, err := json.Marshal(items)
			if err == nil {
				return string(encoded)
			}
		}
	}
	return displayValue(value)
}

func randomAdminID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(value[:])
}
