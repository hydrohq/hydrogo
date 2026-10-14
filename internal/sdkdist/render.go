package sdkdist

import (
	"bytes"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

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
