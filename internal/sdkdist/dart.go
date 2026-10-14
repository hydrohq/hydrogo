package sdkdist

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type sdkDartField struct {
	Name, Type string
	Required   bool
}

func dartFields(raw json.RawMessage) []sdkDartField {
	properties, required := dartObjectShape(raw)
	if len(properties) == 0 {
		return nil
	}
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]sdkDartField, 0, len(keys))
	for _, key := range keys {
		typeName := strings.TrimSuffix(dartType(properties[key]), "?")
		if !required[key] || schemaAllowsNull(properties[key]) {
			typeName += "?"
		}
		fields = append(fields, sdkDartField{Name: safeIdentifier(key, "field_"), Type: typeName, Required: required[key]})
	}
	return fields
}

func dartConstructor(raw json.RawMessage) string {
	fields := dartFields(raw)
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		prefix := ""
		if field.Required {
			prefix = "required "
		}
		values = append(values, prefix+"this."+field.Name)
	}
	return strings.Join(values, ", ")
}

func dartDecodeFields(raw json.RawMessage) string {
	properties, required := dartObjectShape(raw)
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, safeIdentifier(key, "field_")+": "+dartDecodeValue(properties[key], "json["+strconv.Quote(key)+"]", required[key]))
	}
	return strings.Join(values, ", ")
}

func dartEncodeFields(raw json.RawMessage) string {
	properties, _ := dartObjectShape(raw)
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]string, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, strconv.Quote(key)+": "+safeIdentifier(key, "field_"))
	}
	return strings.Join(fields, ", ")
}

func dartObjectShape(raw json.RawMessage) (map[string]json.RawMessage, map[string]bool) {
	properties := make(map[string]json.RawMessage)
	required := make(map[string]bool)
	var collect func(json.RawMessage) bool
	collect = func(document json.RawMessage) bool {
		var schema struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
			Required   []string                   `json:"required"`
			AllOf      []json.RawMessage          `json:"allOf"`
		}
		if json.Unmarshal(document, &schema) != nil {
			return false
		}
		isObject := schema.Type == "object" || len(schema.Properties) > 0
		for key, value := range schema.Properties {
			properties[key] = value
		}
		for _, key := range schema.Required {
			required[key] = true
		}
		for _, part := range schema.AllOf {
			isObject = collect(part) || isObject
		}
		return isObject
	}
	if !collect(raw) {
		return nil, nil
	}
	return properties, required
}

func dartDecodeValue(raw json.RawMessage, value string, required bool) string {
	var schema struct {
		Type  json.RawMessage `json:"type"`
		Ref   string          `json:"$ref"`
		Items json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return value
	}
	if schema.Ref != "" {
		call := typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/")) + ".fromJson(Map<String, Object?>.from(" + value + " as Map))"
		if schemaAllowsNull(raw) {
			return value + " == null ? null : " + call
		}
		return call
	}
	types, _ := jsonSchemaTypes(schema.Type)
	if len(types) != 1 {
		return value
	}
	nonNullable := required && !schemaAllowsNull(raw)
	switch types[0] {
	case "string":
		if nonNullable {
			return value + " as String"
		}
		return value + " as String?"
	case "integer":
		if nonNullable {
			return "(" + value + " as num).toInt()"
		}
		return "(" + value + " as num?)?.toInt()"
	case "number":
		if nonNullable {
			return "(" + value + " as num).toDouble()"
		}
		return "(" + value + " as num?)?.toDouble()"
	case "boolean":
		if nonNullable {
			return value + " as bool"
		}
		return value + " as bool?"
	case "array":
		if nonNullable {
			return "(" + value + " as List).map((item) => " + dartDecodeValue(schema.Items, "item", false) + ").toList()"
		}
		return "(" + value + " as List?)?.map((item) => " + dartDecodeValue(schema.Items, "item", false) + ").toList()"
	case "object":
		return "Map<String, Object?>.from(" + value + " as Map)"
	default:
		return value
	}
}
func schemaDart(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "Object?"
	}
	return dartType(raw)
}
func schemaDartDecode(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(value) => value"
	}
	var schema struct {
		Ref  string `json:"$ref"`
		Type string `json:"type"`
	}
	_ = json.Unmarshal(raw, &schema)
	if schema.Ref != "" {
		return "(value) => " + typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/")) + ".fromJson(Map<String, Object?>.from(value as Map))"
	}
	switch schema.Type {
	case "object":
		return "(value) => Map<String, Object?>.from(value as Map)"
	case "array":
		return "(value) => (value as List).cast<Object?>()"
	default:
		return "(value) => value as " + strings.TrimSuffix(dartType(raw), "?")
	}
}

func dartType(raw json.RawMessage) string {
	var schema struct {
		Type  json.RawMessage   `json:"type"`
		Ref   string            `json:"$ref"`
		Items json.RawMessage   `json:"items"`
		AnyOf []json.RawMessage `json:"anyOf"`
		OneOf []json.RawMessage `json:"oneOf"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return "Object?"
	}
	if schema.Ref != "" {
		return typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/"))
	}
	composition := schema.AnyOf
	if len(composition) == 0 {
		composition = schema.OneOf
	}
	if len(composition) > 0 {
		return "Object?"
	}
	types, nullable := jsonSchemaTypes(schema.Type)
	if len(types) != 1 {
		return "Object?"
	}
	var result string
	switch types[0] {
	case "string":
		result = "String"
	case "integer":
		result = "int"
	case "number":
		result = "double"
	case "boolean":
		result = "bool"
	case "array":
		result = "List<" + strings.TrimSuffix(dartType(schema.Items), "?") + ">"
	case "object":
		result = "Map<String, Object?>"
	default:
		return "Object?"
	}
	if nullable {
		result += "?"
	}
	return result
}
