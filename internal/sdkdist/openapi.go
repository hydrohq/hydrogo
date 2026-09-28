package sdkdist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// SDKLanguage selects generated client artifacts.
type SDKLanguage string

const (
	SDKTypeScript SDKLanguage = "typescript"
	SDKDart       SDKLanguage = "dart"
	SDKAll        SDKLanguage = "all"
)

type openAPIDocument struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"info"`
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components struct {
		Schemas map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

type openAPIOperation struct {
	OperationID string             `json:"operationId"`
	Parameters  []openAPIParameter `json:"parameters"`
	RequestBody struct {
		Required bool `json:"required"`
		Content  map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

type openAPIParameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}
type openAPIPathItem struct {
	Parameters []openAPIParameter `json:"parameters"`
	Get        openAPIOperation   `json:"get"`
	Post       openAPIOperation   `json:"post"`
	Put        openAPIOperation   `json:"put"`
	Patch      openAPIOperation   `json:"patch"`
	Delete     openAPIOperation   `json:"delete"`
}

type sdkOperation struct {
	Name         string
	Method       string
	Path         string
	Parameters   []sdkParameter
	BodySchema   json.RawMessage
	BodyRequired bool
	ReturnSchema json.RawMessage
}

type sdkParameter struct {
	Name     string
	WireName string
	In       string
	Required bool
}

var sdkIdentifier = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// GenerateOpenAPISDK reads an OpenAPI 3.1 JSON document and writes dependency-free
// TypeScript and/or Dart clients. The destination is created if it does not exist.
func GenerateOpenAPISDK(specPath, out string, language SDKLanguage) ([]string, error) {
	if language != SDKAll && language != SDKTypeScript && language != SDKDart {
		return nil, fmt.Errorf("unsupported SDK language %q", language)
	}
	if strings.TrimSpace(specPath) == "" || strings.TrimSpace(out) == "" {
		return nil, errors.New("OpenAPI input and output directory are required")
	}
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI document: %w", err)
	}
	var doc openAPIDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.1.") || len(doc.Paths) == 0 {
		return nil, errors.New("an OpenAPI 3.1 document with at least one path is required")
	}
	operations := sdkOperations(doc.Paths)
	files := map[string][]byte{}
	if language == SDKAll || language == SDKTypeScript {
		body, err := renderTypeScript(doc, operations)
		if err != nil {
			return nil, err
		}
		files["typescript/client.ts"] = body
	}
	if language == SDKAll || language == SDKDart {
		body, err := renderDart(doc, operations)
		if err != nil {
			return nil, err
		}
		files["dart/lib/hydro_client.dart"] = body
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	written := make([]string, 0, len(files))
	for name, body := range files {
		path := filepath.Join(out, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := writeSDKFile(path, body); err != nil {
			return nil, err
		}
		written = append(written, path)
	}
	sort.Strings(written)
	return written, nil
}

func writeSDKFile(path string, body []byte) error {
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return err
	}
	return nil
}

func sdkOperations(paths map[string]openAPIPathItem) []sdkOperation {
	var result []sdkOperation
	for path, pathItem := range paths {
		methods := map[string]openAPIOperation{"get": pathItem.Get, "post": pathItem.Post, "put": pathItem.Put, "patch": pathItem.Patch, "delete": pathItem.Delete}
		for method, operation := range methods {
			if operation.OperationID == "" && len(operation.Parameters) == 0 && len(operation.Responses) == 0 && len(operation.RequestBody.Content) == 0 {
				continue
			}
			name := operation.OperationID
			if name == "" {
				name = method + path
			}
			name = sdkIdentifier.ReplaceAllString(name, "_")
			name = safeIdentifier(name, "request_")
			allParams := append(append([]openAPIParameter(nil), pathItem.Parameters...), operation.Parameters...)
			params := make([]sdkParameter, 0, len(allParams))
			for _, p := range allParams {
				if p.In == "query" || p.In == "path" {
					params = append(params, sdkParameter{Name: safeIdentifier(p.Name, "arg_"), WireName: p.Name, In: p.In, Required: p.Required || p.In == "path"})
				}
			}
			sort.Slice(params, func(i, j int) bool {
				if params[i].Required != params[j].Required {
					return params[i].Required
				}
				return params[i].Name < params[j].Name
			})
			item := sdkOperation{Name: name, Method: strings.ToUpper(method), Path: path, Parameters: params, BodyRequired: operation.RequestBody.Required}
			item.BodySchema = mediaSchema(operation.RequestBody.Content)
			item.ReturnSchema = successSchema(operation.Responses)
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Method < result[j].Method
	})
	return result
}

func safeIdentifier(value, fallback string) string {
	value = sdkIdentifier.ReplaceAllString(strings.TrimSpace(value), "_")
	if value == "" || !(value[0] >= 'a' && value[0] <= 'z' || value[0] >= 'A' && value[0] <= 'Z' || value[0] == '_') {
		value = fallback + value
	}
	switch value {
	case "class", "default", "function", "new", "return", "var", "const", "let", "await", "yield", "enum", "extends", "implements", "interface", "package", "private", "protected", "public", "static", "super", "this", "throw", "try", "catch", "finally", "switch", "case", "break", "continue", "for", "while", "if", "else", "import", "export", "with", "delete", "typeof", "void", "in", "instanceof", "as", "async", "required", "String", "int", "double", "bool", "Object", "dynamic", "final", "true", "false", "null":
		value = "arg_" + value
	}
	return value
}

func mediaSchema(content map[string]struct {
	Schema json.RawMessage `json:"schema"`
}) json.RawMessage {
	if item, ok := content["application/json"]; ok {
		return item.Schema
	}
	keys := make([]string, 0, len(content))
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return content[keys[0]].Schema
	}
	return nil
}

func successSchema(responses map[string]struct {
	Content map[string]struct {
		Schema json.RawMessage `json:"schema"`
	} `json:"content"`
}) json.RawMessage {
	codes := make([]string, 0, len(responses))
	for code := range responses {
		if len(code) == 3 && code[0] == '2' {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	for _, code := range codes {
		if raw := mediaSchema(responses[code].Content); len(raw) > 0 {
			return raw
		}
	}
	return nil
}

func renderTypeScript(doc openAPIDocument, operations []sdkOperation) ([]byte, error) {
	const source = `// Generated from OpenAPI {{.Version}}. Do not edit.
export type JsonValue = null | boolean | number | string | JsonValue[] | { [key: string]: JsonValue };
export interface HydroTransport { request<T>(path: string, init?: RequestInit): Promise<T>; }
export class FetchTransport implements HydroTransport {
  constructor(private readonly baseUrl = "", private readonly token?: string) {}
  async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers);
    if (this.token) headers.set("Authorization", "Bearer " + this.token);
    const response = await fetch(this.baseUrl + path, { ...init, headers });
    if (!response.ok) throw new Error("Hydro API request failed: " + response.status);
    return await response.json() as T;
  }
}
{{range .Schemas}}
export type {{.Name}} = {{.Body}};
{{end}}
export class HydroClient {
  constructor(private readonly transport: HydroTransport) {}
{{range .Operations}}
  {{.Name}}({{.TSArgs}}): Promise<{{.TSReturnType}}> {
    const params = new URLSearchParams();
{{range .QueryParameters}}    if ({{.Name}} !== undefined) params.set({{.TSWireName}}, {{.Name}});
{{end}}    const base = {{.TSPath}};
    const path = base + (params.size ? "?" + params.toString() : "");
    const init: RequestInit = { method: "{{.Method}}" };
{{if .HasBody}}    if (body !== undefined) { init.headers = { "Content-Type": "application/json" }; init.body = JSON.stringify(body); }
{{end}}    return this.transport.request<{{.TSReturnType}}>(path, init);
  }
{{end}}
}
`
	model := buildSDKTemplateData(doc, operations)
	return executeTemplate("typescript", source, model)
}

func renderDart(doc openAPIDocument, operations []sdkOperation) ([]byte, error) {
	const source = `// Generated from OpenAPI {{.Version}}. Do not edit.
import 'dart:async';

abstract interface class HydroTransport {
  Future<T> request<T>(String path, {String method = 'GET', Object? body, required T Function(Object?) decode});
}
{{range .Schemas}}
class {{.Name}} {
  const {{.Name}}({
    {{.DartConstructor}}
  });
{{range .DartFields}}  final {{.Type}} {{.Name}};
{{end}}{{range .DartFields}}
{{end}}  factory {{.Name}}.fromJson(Map<String, Object?> json) => {{.Name}}({{.DartDecodeFields}});
}
{{end}}

class HydroClient {
  HydroClient(this.transport);
  final HydroTransport transport;
{{range .Operations}}
  Future<{{.DartReturnType}}> {{.Name}}({
    {{.DartArgs}}
  }) {
    final query = <String, String>{};
{{range .QueryParameters}}    {{if .Required}}query[{{.DartWireName}}] = {{.Name}};
{{else}}if ({{.Name}} != null) query[{{.DartWireName}}] = {{.Name}};
{{end}}{{end}}    final suffix = query.isEmpty ? '' : '?' + Uri(queryParameters: query).query;
    return transport.request<{{.DartReturnType}}>({{.DartPath}} + suffix, method: '{{.Method}}'{{if .HasBody}}, body: body{{end}}, decode: {{.DartDecode}});
  }
{{end}}
}
`
	return executeTemplate("dart", source, buildSDKTemplateData(doc, operations))
}

type sdkSchema struct {
	Name, Body, DartConstructor, DartDecodeFields string
	DartFields                                    []sdkDartField
}
type sdkDartField struct {
	Name, Type string
	Required   bool
}
type sdkTemplateData struct {
	Version, Title string
	Schemas        []sdkSchema
	Operations     []sdkOperationData
}
type sdkOperationData struct {
	Name, Method, TSPath, DartPath                                       string
	TSArgs, DartArgs, TSReturnType, DartReturnType, DartDecode, BodyName string
	Parameters, QueryParameters                                          []sdkParameterData
	HasBody                                                              bool
}
type sdkParameterData struct {
	Name, WireName, In       string
	TSWireName, DartWireName string
	Required                 bool
}

func buildSDKTemplateData(doc openAPIDocument, operations []sdkOperation) sdkTemplateData {
	data := sdkTemplateData{Version: doc.Info.Version, Title: doc.Info.Title}
	schemaNames := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)
	for _, name := range schemaNames {
		schemaName := typeName(name)
		data.Schemas = append(data.Schemas, sdkSchema{Name: schemaName, Body: typescriptSchema(doc.Components.Schemas[name]), DartConstructor: dartConstructor(doc.Components.Schemas[name]), DartFields: dartFields(doc.Components.Schemas[name]), DartDecodeFields: dartDecodeFields(doc.Components.Schemas[name])})
	}
	for _, operation := range operations {
		item := sdkOperationData{Name: operation.Name, Method: operation.Method, TSReturnType: schemaTypeScript(operation.ReturnSchema), DartReturnType: schemaDart(operation.ReturnSchema), DartDecode: schemaDartDecode(operation.ReturnSchema)}
		item.TSPath, item.DartPath = clientPath(operation.Path, operation.Parameters)
		item.HasBody = len(operation.BodySchema) > 0
		if item.HasBody {
			item.BodyName = "body"
		}
		for _, p := range operation.Parameters {
			arg := sdkParameterData{Name: p.Name, WireName: p.WireName, In: p.In, Required: p.Required, TSWireName: strconv.Quote(p.WireName), DartWireName: strconv.Quote(p.WireName)}
			item.Parameters = append(item.Parameters, arg)
			sepTS := ", "
			if item.TSArgs == "" {
				sepTS = ""
			}
			item.TSArgs += sepTS + p.Name
			if !p.Required {
				item.TSArgs += "?"
			}
			item.TSArgs += ": string"
			sepDart := ", "
			if item.DartArgs == "" {
				sepDart = ""
			}
			item.DartArgs += sepDart
			if p.Required {
				item.DartArgs += "required "
			}
			item.DartArgs += "String"
			if !p.Required {
				item.DartArgs += "?"
			}
			item.DartArgs += " " + p.Name
			if p.In == "query" {
				item.QueryParameters = append(item.QueryParameters, arg)
			}
			if p.In == "path" {
				// `clientPath` has already encoded path-template variables.
			}
		}
		if item.HasBody {
			if item.TSArgs != "" {
				item.TSArgs += ", "
			}
			item.TSArgs += "body"
			if !operation.BodyRequired {
				item.TSArgs += "?"
			}
			item.TSArgs += ": " + schemaTypeScript(operation.BodySchema)
			if item.DartArgs != "" {
				item.DartArgs += ", "
			}
			if operation.BodyRequired {
				item.DartArgs += "required "
			}
			item.DartArgs += "Object? body"
		}
		data.Operations = append(data.Operations, item)
	}
	return data
}

func dartFields(raw json.RawMessage) []sdkDartField {
	var schema struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if json.Unmarshal(raw, &schema) != nil || schema.Type != "object" {
		return nil
	}
	keys := make([]string, 0, len(schema.Properties))
	for key := range schema.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]sdkDartField, 0, len(keys))
	required := make(map[string]bool, len(schema.Required))
	for _, key := range schema.Required {
		required[key] = true
	}
	for _, key := range keys {
		typeName := strings.TrimSuffix(dartType(schema.Properties[key]), "?")
		if !required[key] {
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
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return ""
	}
	keys := make([]string, 0, len(schema.Properties))
	for key := range schema.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, safeIdentifier(key, "field_")+": "+dartDecodeValue(schema.Properties[key], "json["+strconv.Quote(key)+"]"))
	}
	return strings.Join(values, ", ")
}

func dartDecodeValue(raw json.RawMessage, value string) string {
	var schema struct {
		Type  string          `json:"type"`
		Ref   string          `json:"$ref"`
		Items json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return value
	}
	if schema.Ref != "" {
		return typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/")) + ".fromJson(Map<String, Object?>.from(" + value + " as Map))"
	}
	switch schema.Type {
	case "string":
		return value + " as String?"
	case "integer":
		return "(" + value + " as num?)?.toInt()"
	case "number":
		return "(" + value + " as num?)?.toDouble()"
	case "boolean":
		return value + " as bool?"
	case "array":
		return "(" + value + " as List?)?.map((item) => " + dartDecodeValue(schema.Items, "item") + ").toList()"
	case "object":
		return "Map<String, Object?>.from(" + value + " as Map)"
	default:
		return value
	}
}

func schemaTypeScript(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "JsonValue"
	}
	return typescriptSchema(raw)
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

func clientPath(path string, params []sdkParameter) (string, string) {
	byWire := make(map[string]string, len(params))
	for _, p := range params {
		if p.In == "path" {
			byWire[p.WireName] = p.Name
		}
	}
	var ts, dart []string
	for len(path) > 0 {
		start := strings.IndexByte(path, '{')
		if start < 0 {
			if path != "" {
				ts = append(ts, strconv.Quote(path))
				dart = append(dart, strconv.Quote(path))
			}
			break
		}
		if start > 0 {
			ts = append(ts, strconv.Quote(path[:start]))
			dart = append(dart, strconv.Quote(path[:start]))
		}
		end := strings.IndexByte(path[start:], '}')
		if end < 0 {
			ts = append(ts, strconv.Quote(path[start:]))
			dart = append(dart, strconv.Quote(path[start:]))
			break
		}
		wire := path[start+1 : start+end]
		name, ok := byWire[wire]
		if !ok {
			ts = append(ts, strconv.Quote(path[start:start+end+1]))
			dart = append(dart, strconv.Quote(path[start:start+end+1]))
		} else {
			ts = append(ts, "encodeURIComponent("+name+")")
			dart = append(dart, "Uri.encodeComponent("+name+")")
		}
		path = path[start+end+1:]
	}
	if len(ts) == 0 {
		return `"/"`, `"/"`
	}
	return strings.Join(ts, " + "), strings.Join(dart, " + ")
}

func dartType(raw json.RawMessage) string {
	var schema struct {
		Type  string          `json:"type"`
		Ref   string          `json:"$ref"`
		Items json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return "Object?"
	}
	if schema.Ref != "" {
		return typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/"))
	}
	switch schema.Type {
	case "string":
		return "String?"
	case "integer":
		return "int?"
	case "number":
		return "double?"
	case "boolean":
		return "bool?"
	case "array":
		return "List<" + strings.TrimSuffix(dartType(schema.Items), "?") + ">?"
	case "object":
		return "Map<String, Object?>?"
	default:
		return "Object?"
	}
}

func executeTemplate(name, source string, data any) ([]byte, error) {
	t, err := template.New(name).Parse(source)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := t.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
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
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Items      json.RawMessage            `json:"items"`
		Enum       []json.RawMessage          `json:"enum"`
		Ref        string                     `json:"$ref"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return "{ [key: string]: JsonValue }"
	}
	if schema.Ref != "" {
		return typeName(strings.TrimPrefix(schema.Ref, "#/components/schemas/"))
	}
	if len(schema.Enum) > 0 {
		values := make([]string, 0, len(schema.Enum))
		for _, item := range schema.Enum {
			values = append(values, string(item))
		}
		return strings.Join(values, " | ")
	}
	switch schema.Type {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return typescriptSchema(schema.Items) + "[]"
	case "object":
		keys := make([]string, 0, len(schema.Properties))
		for key := range schema.Properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		required := map[string]bool{}
		for _, key := range schema.Required {
			required[key] = true
		}
		var b strings.Builder
		b.WriteString("{\n")
		for _, key := range keys {
			fmt.Fprintf(&b, "  %q%s: %s;\n", key, map[bool]string{true: "", false: "?"}[required[key]], typescriptSchema(schema.Properties[key]))
		}
		b.WriteString("}")
		return b.String()
	default:
		return "JsonValue"
	}
}

// Ensure the template output remains valid UTF-8 source and that a destination
// path cannot escape the requested output directory.
func validSDKPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
