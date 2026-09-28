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
	Name     string          `json:"name"`
	In       string          `json:"in"`
	Required bool            `json:"required"`
	Schema   json.RawMessage `json:"schema"`
}
type openAPIPathItem struct {
	Parameters []openAPIParameter `json:"parameters"`
	Get        openAPIOperation   `json:"get"`
	Post       openAPIOperation   `json:"post"`
	Put        openAPIOperation   `json:"put"`
	Patch      openAPIOperation   `json:"patch"`
	Delete     openAPIOperation   `json:"delete"`
	Head       openAPIOperation   `json:"head"`
	Options    openAPIOperation   `json:"options"`
	Trace      openAPIOperation   `json:"trace"`
}

type sdkOperation struct {
	Name         string
	Method       string
	Path         string
	Parameters   []sdkParameter
	BodySchema   json.RawMessage
	BodyRequired bool
	ReturnSchema json.RawMessage
	NoContent    bool
}

type sdkParameter struct {
	Name     string
	WireName string
	In       string
	Required bool
	Schema   json.RawMessage
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
	if language == SDKAll || language == SDKTypeScript {
		packageJSON, _ := json.MarshalIndent(struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Type    string `json:"type"`
			Exports string `json:"exports"`
		}{Name: "hydro-sdk-client", Version: sdkPackageVersion(doc.Info.Version), Type: "module", Exports: "./client.ts"}, "", "  ")
		files["typescript/package.json"] = append(packageJSON, '\n')
		files["typescript/tsconfig.json"] = []byte("{\n  \"compilerOptions\": {\n    \"target\": \"ES2022\",\n    \"module\": \"NodeNext\",\n    \"moduleResolution\": \"NodeNext\",\n    \"strict\": true,\n    \"lib\": [\"ES2022\", \"DOM\"]\n  },\n  \"include\": [\"client.ts\"]\n}\n")
	}
	if language == SDKAll || language == SDKDart {
		files["dart/pubspec.yaml"] = []byte("name: hydro_sdk_client\nversion: " + sdkPackageVersion(doc.Info.Version) + "\nenvironment:\n  sdk: '>=3.2.0 <4.0.0'\n\n# Uses dart:io and the Dart standard library; no package dependencies.\n")
	}
	files["README.md"] = []byte("# HydroGo generated SDKs\n\nClients generated from an OpenAPI 3.1 JSON document. TypeScript uses the Fetch API; Dart includes a standard-library `dart:io` transport for Flutter native and Dart server applications. Both allow a custom transport.\n\n- TypeScript: `typescript/client.ts`\n- Dart: `dart/lib/hydro_client.dart`\n- Source contract: supplied with `--openapi`\n\nGenerated files are deterministic for a given contract and generator version.\n")
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

var sdkSemver = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

func sdkPackageVersion(value string) string {
	if sdkSemver.MatchString(value) {
		return value
	}
	return "0.0.0"
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
		methods := map[string]openAPIOperation{"get": pathItem.Get, "post": pathItem.Post, "put": pathItem.Put, "patch": pathItem.Patch, "delete": pathItem.Delete, "head": pathItem.Head, "options": pathItem.Options, "trace": pathItem.Trace}
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
					params = append(params, sdkParameter{Name: safeIdentifier(p.Name, "arg_"), WireName: p.Name, In: p.In, Required: p.Required || p.In == "path", Schema: p.Schema})
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
			item.NoContent = operationHasOnlyNoContentResponses(operation.Responses)
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

func operationHasOnlyNoContentResponses(responses map[string]struct {
	Content map[string]struct {
		Schema json.RawMessage `json:"schema"`
	} `json:"content"`
}) bool {
	hasSuccess := false
	for code, response := range responses {
		if len(code) != 3 || code[0] != '2' {
			continue
		}
		hasSuccess = true
		if len(response.Content) > 0 {
			return false
		}
	}
	return hasSuccess
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
    const text = await response.text();
    if (!text) return undefined as T;
    return (response.headers.get("content-type")?.includes("json") ? JSON.parse(text) : text) as T;
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
{{range .QueryParameters}}    if ({{.Name}} !== undefined) params.set({{.TSWireName}}, String({{.Name}}));
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
import 'dart:convert';
import 'dart:io';

abstract interface class HydroTransport {
  Future<T> request<T>(String path, {String method = 'GET', Object? body, required T Function(Object?) decode});
}
class HydroIOTransport implements HydroTransport {
  HydroIOTransport(this.baseUrl, {this.token, HttpClient? client}) : client = client ?? HttpClient();
  final String baseUrl;
  final String? token;
  final HttpClient client;
  @override
  Future<T> request<T>(String path, {String method = 'GET', Object? body, required T Function(Object?) decode}) async {
    final request = await client.openUrl(method, Uri.parse(baseUrl).resolve(path));
    request.headers.set(HttpHeaders.acceptHeader, 'application/json');
    if (token != null && token!.isNotEmpty) request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
    if (body != null) {
      request.headers.contentType = ContentType.json;
      request.write(jsonEncode(body));
    }
    final response = await request.close();
    final text = await utf8.decoder.bind(response).join();
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw HttpException('Hydro API request failed: ${response.statusCode}', uri: response.request.uri);
    }
    if (text.isEmpty) return decode(null);
    final isJson = response.headers.contentType?.mimeType.toLowerCase().contains('json') ?? false;
    return decode(isJson ? jsonDecode(text) : text);
  }
}
{{range .Schemas}}
class {{.Name}} {
  const {{.Name}}({
    {{.DartConstructor}}
  });
{{range .DartFields}}  final {{.Type}} {{.Name}};
{{end}}  Map<String, Object?> toJson() => { {{.DartEncodeFields}} };
  factory {{.Name}}.fromJson(Map<String, Object?> json) => {{.Name}}({{.DartDecodeFields}});
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
{{range .QueryParameters}}    {{if and .Required (not .Nullable)}}query[{{.DartWireName}}] = {{.Name}}.toString();
{{else}}if ({{.Name}} != null) query[{{.DartWireName}}] = {{.Name}}.toString();
{{end}}{{end}}    final suffix = query.isEmpty ? '' : '?' + Uri(queryParameters: query).query;
    return transport.request<{{.DartReturnType}}>({{.DartPath}} + suffix, method: '{{.Method}}'{{if .HasBody}}, body: body{{end}}, decode: {{.DartDecode}});
  }
{{end}}
}
`
	return executeTemplate("dart", source, buildSDKTemplateData(doc, operations))
}

type sdkSchema struct {
	Name, Body, DartConstructor, DartDecodeFields, DartEncodeFields string
	DartFields                                                      []sdkDartField
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
	DartBodyType                                                         string
	Parameters, QueryParameters                                          []sdkParameterData
	HasBody                                                              bool
	NoContent                                                            bool
}
type sdkParameterData struct {
	Name, WireName, In       string
	TSWireName, DartWireName string
	TSType, DartType         string
	Required, Nullable       bool
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
		data.Schemas = append(data.Schemas, sdkSchema{Name: schemaName, Body: typescriptSchema(doc.Components.Schemas[name]), DartConstructor: dartConstructor(doc.Components.Schemas[name]), DartFields: dartFields(doc.Components.Schemas[name]), DartDecodeFields: dartDecodeFields(doc.Components.Schemas[name]), DartEncodeFields: dartEncodeFields(doc.Components.Schemas[name])})
	}
	for _, operation := range operations {
		item := sdkOperationData{Name: operation.Name, Method: operation.Method, TSReturnType: schemaTypeScript(operation.ReturnSchema), DartReturnType: schemaDart(operation.ReturnSchema), DartDecode: schemaDartDecode(operation.ReturnSchema), NoContent: operation.NoContent}
		if operation.NoContent {
			item.TSReturnType = "void"
			item.DartReturnType = "void"
			item.DartDecode = "(value) => null"
		}
		item.TSPath, item.DartPath = clientPath(operation.Path, operation.Parameters)
		item.HasBody = len(operation.BodySchema) > 0
		if item.HasBody {
			item.BodyName = "body"
			item.DartBodyType = schemaDart(operation.BodySchema)
			if !operation.BodyRequired && !schemaAllowsNull(operation.BodySchema) {
				item.DartBodyType = strings.TrimSuffix(item.DartBodyType, "?") + "?"
			}
		}
		requiredTSArgs := make([]string, 0, len(operation.Parameters)+1)
		optionalTSArgs := make([]string, 0, len(operation.Parameters)+1)
		for _, p := range operation.Parameters {
			tsType := "string"
			dartParamType := "String"
			if len(p.Schema) > 0 {
				tsType = typescriptSchema(p.Schema)
				dartParamType = strings.TrimSuffix(dartType(p.Schema), "?")
				if !p.Required || schemaAllowsNull(p.Schema) {
					dartParamType += "?"
				}
			} else if !p.Required {
				dartParamType += "?"
			}
			arg := sdkParameterData{Name: p.Name, WireName: p.WireName, In: p.In, Required: p.Required, Nullable: schemaAllowsNull(p.Schema), TSWireName: strconv.Quote(p.WireName), DartWireName: strconv.Quote(p.WireName), TSType: tsType, DartType: dartParamType}
			item.Parameters = append(item.Parameters, arg)
			tsArg := p.Name
			if !p.Required {
				tsArg += "?"
			}
			tsArg += ": " + tsType
			if p.Required {
				requiredTSArgs = append(requiredTSArgs, tsArg)
			} else {
				optionalTSArgs = append(optionalTSArgs, tsArg)
			}
			sepDart := ", "
			if item.DartArgs == "" {
				sepDart = ""
			}
			item.DartArgs += sepDart
			if p.Required {
				item.DartArgs += "required "
			}
			item.DartArgs += dartParamType
			item.DartArgs += " " + p.Name
			if p.In == "query" {
				item.QueryParameters = append(item.QueryParameters, arg)
			}
			if p.In == "path" {
				// `clientPath` has already encoded path-template variables.
			}
		}
		if item.HasBody {
			bodyArg := "body"
			if !operation.BodyRequired {
				bodyArg += "?"
			}
			bodyArg += ": " + schemaTypeScript(operation.BodySchema)
			if operation.BodyRequired {
				requiredTSArgs = append(requiredTSArgs, bodyArg)
			} else {
				optionalTSArgs = append(optionalTSArgs, bodyArg)
			}
			if item.DartArgs != "" {
				item.DartArgs += ", "
			}
			if operation.BodyRequired {
				item.DartArgs += "required "
			}
			item.DartArgs += item.DartBodyType + " body"
		}
		item.TSArgs = strings.Join(append(requiredTSArgs, optionalTSArgs...), ", ")
		data.Operations = append(data.Operations, item)
	}
	return data
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

// Ensure the template output remains valid UTF-8 source and that a destination
// path cannot escape the requested output directory.
func validSDKPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
