package sdkdist

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func schemaTypeScript(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "JsonValue"
	}
	return typescriptSchema(raw)
}

func schemaAllowsNull(raw json.RawMessage) bool {
	var schema struct {
		Type     json.RawMessage   `json:"type"`
		Nullable bool              `json:"nullable"`
		AnyOf    []json.RawMessage `json:"anyOf"`
		OneOf    []json.RawMessage `json:"oneOf"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return false
	}
	_, nullable := jsonSchemaTypes(schema.Type)
	if nullable || schema.Nullable {
		return true
	}
	for _, variants := range [][]json.RawMessage{schema.AnyOf, schema.OneOf} {
		for _, variant := range variants {
			if types, _ := jsonSchemaTypes(variant); len(types) == 0 {
				var value any
				if json.Unmarshal(variant, &value) == nil && value == nil {
					return true
				}
			}
		}
	}
	return false
}

func typeName(value string) string {
	value = strings.TrimPrefix(value, "#/components/schemas/")
	parts := strings.FieldsFunc(value, func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') })
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	name := strings.Join(parts, "")
	if name == "" || name[0] >= '0' && name[0] <= '9' {
		name = "Schema" + name
	}
	return name
}

func typescriptSchema(raw json.RawMessage) string {
	var schema struct {
		Type       json.RawMessage            `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Items      json.RawMessage            `json:"items"`
		Enum       []json.RawMessage          `json:"enum"`
		Ref        string                     `json:"$ref"`
		AnyOf      []json.RawMessage          `json:"anyOf"`
		OneOf      []json.RawMessage          `json:"oneOf"`
		AllOf      []json.RawMessage          `json:"allOf"`
		Additional json.RawMessage            `json:"additionalProperties"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return "{ [key: string]: JsonValue }"
	}
	if schema.Ref != "" {
		return typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/"))
	}
	composition := schema.AnyOf
	if len(composition) == 0 {
		composition = schema.OneOf
	}
	if len(composition) > 0 {
		parts := make([]string, 0, len(composition))
		for _, part := range composition {
			parts = append(parts, typescriptSchema(part))
		}
		return strings.Join(parts, " | ")
	}
	if len(schema.AllOf) > 0 {
		parts := make([]string, 0, len(schema.AllOf))
		for _, part := range schema.AllOf {
			parts = append(parts, typescriptSchema(part))
		}
		return strings.Join(parts, " & ")
	}
	if len(schema.Enum) > 0 {
		values := make([]string, 0, len(schema.Enum))
		for _, item := range schema.Enum {
			values = append(values, string(item))
		}
		return strings.Join(values, " | ")
	}
	types, includesNull := jsonSchemaTypes(schema.Type)
	var result string
	switch {
	case len(types) > 1:
		parts := make([]string, 0, len(types))
		for _, typ := range types {
			if typ == "null" {
				continue
			}
			parts = append(parts, typescriptSchema(json.RawMessage(strconv.Quote(typ))))
		}
		result = strings.Join(parts, " | ")
	case len(types) == 0:
		return "JsonValue"
	case types[0] == "string":
		result = "string"
	case types[0] == "integer" || types[0] == "number":
		result = "number"
	case types[0] == "boolean":
		result = "boolean"
	case types[0] == "array":
		result = typescriptSchema(schema.Items) + "[]"
	case types[0] == "object":
		keys := make([]string, 0, len(schema.Properties))
		for key := range schema.Properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		required := map[string]bool{}
		for _, key := range schema.Required {
			required[key] = true
		}
		if len(keys) == 0 {
			if len(schema.Additional) == 0 || string(schema.Additional) == "true" {
				result = "{ [key: string]: JsonValue }"
			} else if string(schema.Additional) == "false" {
				result = "Record<string, never>"
			} else {
				result = "{ [key: string]: " + typescriptSchema(schema.Additional) + " }"
			}
			break
		}
		var b strings.Builder
		b.WriteString("{\n")
		for _, key := range keys {
			fmt.Fprintf(&b, "  %q%s: %s;\n", key, map[bool]string{true: "", false: "?"}[required[key]], typescriptSchema(schema.Properties[key]))
		}
		b.WriteString("}")
		result = b.String()
	default:
		return "JsonValue"
	}
	if includesNull {
		result += " | null"
	}
	return result
}

func jsonSchemaTypes(raw json.RawMessage) ([]string, bool) {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return []string{single}, false
	}
	var values []string
	if json.Unmarshal(raw, &values) != nil {
		return nil, false
	}
	result := make([]string, 0, len(values))
	nullable := false
	for _, value := range values {
		if value == "null" {
			nullable = true
			continue
		}
		result = append(result, value)
	}
	return result, nullable
}
