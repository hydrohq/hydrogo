package adminstudio

import (
	"fmt"
	"strings"
)

// DisplayValue renders a record value as one string; revision snapshots use
// it so a stored revision compares equal to the current record field by field.
func DisplayValue(value any) string {
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
