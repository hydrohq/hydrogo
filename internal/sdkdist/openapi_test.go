package sdkdist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateOpenAPISDKCreatesBothClientsDeterministically(t *testing.T) {
	root := t.TempDir()
	spec := filepath.Join(root, "openapi.json")
	data := `{"openapi":"3.1.0","info":{"title":"API","version":"1"},"paths":{"/items/{item-id}":{"parameters":[{"name":"item-id","in":"path","required":true}],"get":{"operationId":"getItem","parameters":[{"name":"locale","in":"query","schema":{"type":"integer"}}],"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}}}},"post":{"operationId":"createItem","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}},"responses":{"201":{"description":"created","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}}}}}},"components":{"schemas":{"Item":{"type":"object","required":["id"],"properties":{"id":{"type":"string"},"active":{"type":"boolean"},"nickname":{"type":["string","null"]}}}}}}`
	if err := os.WriteFile(spec, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "sdk")
	files, err := GenerateOpenAPISDK(spec, out, SDKAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 6 {
		t.Fatalf("generated SDK package files = %v, want 6", files)
	}
	ts, err := os.ReadFile(filepath.Join(out, "typescript/client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"export type Item", `"id": string`, `"active"?: boolean`, `"nickname"?: string | null`, "getItem(item_id: string, locale?: number): Promise<Item>", "createItem(item_id: string, body: Item): Promise<Item>", "encodeURIComponent(item_id)", `params.set("locale", String(locale))`, "JSON.stringify(body)"} {
		if !strings.Contains(string(ts), want) {
			t.Errorf("TypeScript missing %q:\n%s", want, ts)
		}
	}
	dart, err := os.ReadFile(filepath.Join(out, "dart/lib/hydro_client.dart"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"abstract interface class HydroTransport", "class HydroIOTransport implements HydroTransport", "Future<Item> getItem({", "required String item_id", "int? locale", "required this.id", `json["id"] as String`, `String? nickname`, `nickname: json["nickname"] as String?`, `Map<String, Object?> toJson()`, `"nickname": nickname`, "factory Item.fromJson", "Uri.encodeComponent(item_id)", "required Item body"} {
		if !strings.Contains(string(dart), want) {
			t.Errorf("Dart missing %q:\n%s", want, dart)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "typescript/package.json")); err != nil {
		t.Fatalf("TypeScript package metadata missing: %v", err)
	}
	tsconfig, err := os.ReadFile(filepath.Join(out, "typescript/tsconfig.json"))
	if err != nil {
		t.Fatalf("TypeScript compiler config missing: %v", err)
	}
	for _, option := range []string{`"strict": true`, `"exactOptionalPropertyTypes": true`, `"noUncheckedIndexedAccess": true`, `"noEmit": true`} {
		if !strings.Contains(string(tsconfig), option) {
			t.Errorf("TypeScript compiler config missing %q: %s", option, tsconfig)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "dart/pubspec.yaml")); err != nil {
		t.Fatalf("Dart package metadata missing: %v", err)
	}
}

func TestRepositoryPublicOpenAPIContractGeneratesTypedResponses(t *testing.T) {
	spec := filepath.Join("..", "..", "docs", "api", "public-content-v1.openapi.json")
	data, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	var doc openAPIDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse public OpenAPI contract: %v", err)
	}
	for path, item := range doc.Paths {
		for _, operation := range []openAPIOperation{item.Get, item.Post, item.Put, item.Patch, item.Delete} {
			if operation.OperationID == "" {
				continue
			}
			if len(operation.Responses) == 0 {
				t.Errorf("%s operation %s has no responses", path, operation.OperationID)
			}
			for _, param := range operation.Parameters {
				if len(param.Schema) == 0 {
					t.Errorf("%s parameter %s has no schema", operation.OperationID, param.Name)
				}
			}
		}
	}
	out := t.TempDir()
	if _, err := GenerateOpenAPISDK(spec, out, SDKAll); err != nil {
		t.Fatal(err)
	}
	ts, err := os.ReadFile(filepath.Join(out, "typescript", "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"listPublishedNews(locale: string, limit?: number): Promise<NewsItem[]>", "getPublishedPageBySlug(locale: string, slug: string): Promise<Page>", `"payload": { [key: string]: JsonValue }`} {
		if !strings.Contains(string(ts), want) {
			t.Errorf("generated public SDK missing %q:\n%s", want, ts)
		}
	}
	if strings.Contains(string(ts), "Promise<void>") {
		t.Fatal("public JSON endpoints were incorrectly emitted as no-content operations")
	}
}

func TestRepositoryMobileOpenAPIContractsGenerateTypedClients(t *testing.T) {
	for _, name := range []string{"mobile-v1.openapi.json", "member-mobile-v1.openapi.json"} {
		t.Run(name, func(t *testing.T) {
			spec := filepath.Join("..", "..", "docs", "api", name)
			out := t.TempDir()
			if _, err := GenerateOpenAPISDK(spec, out, SDKAll); err != nil {
				t.Fatal(err)
			}
			for _, generated := range []string{"typescript/client.ts", "dart/lib/hydro_client.dart"} {
				generatedPath := filepath.Join(out, generated)
				if info, err := os.Stat(generatedPath); err != nil || info.Size() == 0 {
					t.Errorf("generated %s missing or empty: %v", generated, err)
					continue
				}
				code, err := os.ReadFile(generatedPath)
				if err != nil {
					t.Fatal(err)
				}
				if strings.HasSuffix(generated, ".ts") && regexp.MustCompile(`\bany\b`).Match(code) {
					t.Error("generated TypeScript contains any")
				}
				if strings.HasSuffix(generated, ".dart") && regexp.MustCompile(`\bdynamic\b`).Match(code) {
					t.Error("generated Dart contains dynamic")
				}
			}
		})
	}
}

func TestOpenAPIDocumentRejectsUnresolvedAndExternalReferences(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{name: "unresolved", spec: `{"openapi":"3.1.0","info":{"title":"API","version":"1"},"paths":{"/items":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Missing"}}}}}}}},"components":{"schemas":{}}}`},
		{name: "external", spec: `{"openapi":"3.1.0","info":{"title":"API","version":"1"},"paths":{"/items":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"https://example.test/schema.json"}}}}}}}}}`},
		{name: "missing-info", spec: `{"openapi":"3.1.0","paths":{"/items":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "openapi.json")
			if err := os.WriteFile(path, []byte(test.spec), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := GenerateOpenAPISDK(path, filepath.Join(t.TempDir(), "sdk"), SDKAll); err == nil {
				t.Fatal("invalid OpenAPI document accepted")
			}
		})
	}
}

func TestSDKOperationsRecognizesNoContentResponses(t *testing.T) {
	responses := map[string]openAPIResponse{"204": {}}
	operations, err := sdkOperations(openAPIDocument{Paths: map[string]openAPIPathItem{"/items/{id}": {Delete: openAPIOperation{OperationID: "deleteItem", Responses: responses}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || !operations[0].NoContent {
		t.Fatalf("no-content operation = %+v", operations)
	}
	data := buildSDKTemplateData(openAPIDocument{}, operations)
	if data.Operations[0].TSReturnType != "void" || data.Operations[0].DartReturnType != "void" {
		t.Fatalf("no-content generated types = %+v", data.Operations[0])
	}
}

func TestSDKOperationsResolvesReusableRequestBodiesAndResponses(t *testing.T) {
	doc := openAPIDocument{
		Info: struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		}{Title: "API", Version: "1.0.0"},
		Paths: map[string]openAPIPathItem{"/items": {Post: openAPIOperation{
			OperationID: "createItem",
			RequestBody: openAPIRequestBody{Ref: "#/components/requestBodies/CreateItem"},
			Responses:   map[string]openAPIResponse{"201": {Ref: "#/components/responses/CreatedItem"}},
		}}},
	}
	doc.Components.Schemas = map[string]json.RawMessage{"Item": json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}}}`)}
	doc.Components.RequestBodies = map[string]openAPIRequestBody{"CreateItem": {Required: true, Content: map[string]openAPIMediaType{"application/json": {Schema: json.RawMessage(`{"$ref":"#/components/schemas/Item"}`)}}}}
	doc.Components.Responses = map[string]openAPIResponse{"CreatedItem": {Content: map[string]openAPIMediaType{"application/json": {Schema: json.RawMessage(`{"$ref":"#/components/schemas/Item"}`)}}}}
	operations, err := sdkOperations(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || !operations[0].BodyRequired || string(operations[0].BodySchema) != `{"$ref":"#/components/schemas/Item"}` || string(operations[0].ReturnSchema) != string(operations[0].BodySchema) {
		t.Fatalf("resolved operation = %+v", operations)
	}
}

func TestSDKRequiredBodyPrecedesOptionalTypeScriptParameters(t *testing.T) {
	operations := []sdkOperation{{
		Name: "createItem", Method: "POST", Path: "/items/{id}", BodySchema: json.RawMessage(`{"$ref":"#/components/schemas/Item"}`), BodyRequired: true,
		Parameters: []sdkParameter{{Name: "id", WireName: "id", In: "path", Required: true}, {Name: "locale", WireName: "locale", In: "query"}},
	}}
	data := buildSDKTemplateData(openAPIDocument{}, operations)
	if got, want := data.Operations[0].TSArgs, "id: string, body: Item, locale?: string"; got != want {
		t.Fatalf("TypeScript args = %q, want %q", got, want)
	}
}

func TestSchemaTypeScriptSupportsOpenAPI31NullableAndCompositions(t *testing.T) {
	if got := typescriptSchema(json.RawMessage(`{"type":["string","null"]}`)); got != "string | null" {
		t.Fatalf("nullable type = %q", got)
	}
	if got := typescriptSchema(json.RawMessage(`{"anyOf":[{"type":"string"},{"type":"integer"}]}`)); got != "string | number" {
		t.Fatalf("union type = %q", got)
	}
	if got := typescriptSchema(json.RawMessage(`{"allOf":[{"type":"object","properties":{"id":{"type":"string"}}},{"type":"object","properties":{"name":{"type":"string"}}}]}`)); !strings.Contains(got, " & ") {
		t.Fatalf("intersection type = %q", got)
	}
	fields := dartFields(json.RawMessage(`{"allOf":[{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}},{"type":"object","properties":{"name":{"type":"string"}}}]}`))
	if len(fields) != 2 || fields[0].Name != "id" || fields[0].Type != "String" || fields[1].Name != "name" || fields[1].Type != "String?" {
		t.Fatalf("Dart allOf fields = %+v", fields)
	}
}

func TestGenerateOpenAPISDKRejectsUnsupportedAndMalformedSpecs(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bad.json")
	if err := os.WriteFile(path, []byte(`{"openapi":"3.0.3","paths":{"/x":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateOpenAPISDK(path, filepath.Join(root, "out"), SDKAll); err == nil {
		t.Fatal("accepted OpenAPI 3.0")
	}
	if _, err := GenerateOpenAPISDK(path, filepath.Join(root, "out"), "rust"); err == nil {
		t.Fatal("accepted unsupported language")
	}
}
