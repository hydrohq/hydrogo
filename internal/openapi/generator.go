package openapi

import (
	"encoding/json"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

// ToPascalCase converts snake_case or kebab-case identifiers to PascalCase.
func ToPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	res := strings.Join(parts, "")
	if len(res) > 0 && res[0] >= '0' && res[0] <= '9' {
		res = "Type" + res
	}
	return res
}

// FieldToSchema converts a contenttypes.Field into OpenAPI JSON schema definition.
func FieldToSchema(f contenttypes.Field) map[string]any {
	var itemSchema map[string]any

	switch f.Kind {
	case "number":
		itemSchema = map[string]any{"type": "number"}
	case "boolean":
		itemSchema = map[string]any{"type": "boolean"}
	case "date":
		itemSchema = map[string]any{"type": "string", "format": "date"}
	case "datetime":
		itemSchema = map[string]any{"type": "string", "format": "date-time"}
	case "select":
		if len(f.Options) > 0 {
			enums := make([]any, len(f.Options))
			for i, opt := range f.Options {
				enums[i] = opt
			}
			itemSchema = map[string]any{"type": "string", "enum": enums}
		} else {
			itemSchema = map[string]any{"type": "string"}
		}
	case "relation":
		if f.Many {
			return map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			}
		}
		itemSchema = map[string]any{"type": "string"}
	case "media":
		itemSchema = map[string]any{"type": "string"}
	default: // text, textarea, richtext, etc.
		itemSchema = map[string]any{"type": "string"}
	}

	if f.List {
		return map[string]any{
			"type":  "array",
			"items": itemSchema,
		}
	}
	return itemSchema
}

// BuildDocument merges content type definitions into an OpenAPI 3.1 JSON document.
func BuildDocument(baseSpec []byte, definitions []contenttypes.Definition) ([]byte, error) {
	doc := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "HydroGo API",
			"version":     "1.0.0",
			"description": "HydroGo CMS API",
		},
		"servers": []any{
			map[string]any{"url": "/"},
		},
		"paths": map[string]any{},
		"components": map[string]any{
			"schemas":   map[string]any{},
			"responses": map[string]any{},
		},
	}

	if len(baseSpec) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(baseSpec, &parsed); err == nil {
			doc = parsed
		}
	}

	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		paths = map[string]any{}
		doc["paths"] = paths
	}

	components, _ := doc["components"].(map[string]any)
	if components == nil {
		components = map[string]any{}
		doc["components"] = components
	}

	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		schemas = map[string]any{}
		components["schemas"] = schemas
	}

	for _, def := range definitions {
		pascal := ToPascalCase(def.ID)
		recordName := pascal + "Record"
		listName := pascal + "List"

		// 1. Build Record Schema
		properties := map[string]any{
			"id":         map[string]any{"type": "string"},
			"created_at": map[string]any{"type": "string", "format": "date-time"},
			"updated_at": map[string]any{"type": "string", "format": "date-time"},
		}
		requiredFields := []string{"id"}

		if def.Drafts {
			properties["status"] = map[string]any{"type": "string", "enum": []any{"draft", "published"}}
		}

		for _, field := range def.Fields {
			properties[field.ID] = FieldToSchema(field)
			if field.Required {
				requiredFields = append(requiredFields, field.ID)
			}
		}

		schemas[recordName] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             requiredFields,
			"properties":           properties,
		}

		// 2. Build List Schema
		schemas[listName] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"data", "metadata"},
			"properties": map[string]any{
				"data": map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/components/schemas/" + recordName},
				},
				"metadata": map[string]any{
					"type":     "object",
					"required": []string{"limit", "has_more"},
					"properties": map[string]any{
						"limit":       map[string]any{"type": "integer"},
						"has_more":    map[string]any{"type": "boolean"},
						"next_cursor": map[string]any{"type": "string"},
					},
				},
			},
		}

		// 3. Register paths
		collectionPath := "/api/v1/content/" + def.ID
		recordPath := "/api/v1/content/" + def.ID + "/{id}"

		paths[collectionPath] = map[string]any{
			"get": map[string]any{
				"operationId": "list" + pascal,
				"summary":     "List " + def.Label,
				"parameters": []any{
					map[string]any{"name": "locale", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
					map[string]any{"name": "limit", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}},
					map[string]any{"name": "offset", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 0}},
					map[string]any{"name": "after", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
					map[string]any{"name": "sort", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
					map[string]any{"name": "search", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
				},
				"responses": map[string]any{
					"200": map[string]any{
						"description": "List of " + def.Label,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{"$ref": "#/components/schemas/" + listName},
							},
						},
					},
				},
			},
			"post": map[string]any{
				"operationId": "create" + pascal,
				"summary":     "Create " + def.Label,
				"requestBody": map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"$ref": "#/components/schemas/" + recordName},
						},
					},
				},
				"responses": map[string]any{
					"201": map[string]any{
						"description": "Created " + def.Label,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"data": map[string]any{"$ref": "#/components/schemas/" + recordName},
									},
								},
							},
						},
					},
				},
			},
		}

		paths[recordPath] = map[string]any{
			"get": map[string]any{
				"operationId": "get" + pascal,
				"summary":     "Get " + def.Label + " by ID",
				"parameters": []any{
					map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}},
					map[string]any{"name": "locale", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
				},
				"responses": map[string]any{
					"200": map[string]any{
						"description": def.Label,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"data": map[string]any{"$ref": "#/components/schemas/" + recordName},
									},
								},
							},
						},
					},
				},
			},
			"put": map[string]any{
				"operationId": "update" + pascal,
				"summary":     "Update " + def.Label,
				"parameters": []any{
					map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}},
				},
				"requestBody": map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"$ref": "#/components/schemas/" + recordName},
						},
					},
				},
				"responses": map[string]any{
					"200": map[string]any{
						"description": "Updated " + def.Label,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"data": map[string]any{"$ref": "#/components/schemas/" + recordName},
									},
								},
							},
						},
					},
				},
			},
			"delete": map[string]any{
				"operationId": "delete" + pascal,
				"summary":     "Delete " + def.Label,
				"parameters": []any{
					map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}},
				},
				"responses": map[string]any{
					"204": map[string]any{
						"description": "Deleted " + def.Label,
					},
				},
			},
		}
	}

	return json.MarshalIndent(doc, "", "  ")
}
