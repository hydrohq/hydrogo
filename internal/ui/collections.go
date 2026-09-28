package ui

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

type CollectionRecordField struct {
	Field    collections.Field
	Value    string
	Checked  bool
	Selected map[string]bool
}

func CollectionRecordFields(schema collections.Schema, data map[string]any) []CollectionRecordField {
	result := make([]CollectionRecordField, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		value := data[field.ID]
		item := CollectionRecordField{Field: field, Selected: map[string]bool{}}
		switch typed := value.(type) {
		case string:
			item.Value = typed
		case bool:
			item.Checked = typed
		case []any:
			for _, part := range typed {
				if text, ok := part.(string); ok {
					item.Selected[text] = true
					if item.Value != "" {
						item.Value += ","
					}
					item.Value += text
				}
			}
		case float64:
			item.Value = strconv.FormatFloat(typed, 'f', -1, 64)
		case nil:
		default:
			encoded, _ := json.Marshal(typed)
			item.Value = string(encoded)
		}
		result = append(result, item)
	}
	return result
}

func collectionFieldName(prefix string, index int) string {
	return fmt.Sprintf("field_%s_%d", prefix, index)
}
func collectionFieldCount(fields []collections.Field) string { return strconv.Itoa(len(fields)) }

func collectionRecordSummary(data map[string]any) string {
	encoded, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return fmt.Sprint(string(encoded))
}
