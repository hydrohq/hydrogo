package sdkdist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOpenAPISDKCreatesBothClientsDeterministically(t *testing.T) {
	root := t.TempDir()
	spec := filepath.Join(root, "openapi.json")
	data := `{"openapi":"3.1.0","info":{"title":"API","version":"1"},"paths":{"/items/{item-id}":{"parameters":[{"name":"item-id","in":"path","required":true}],"get":{"operationId":"getItem","parameters":[{"name":"locale","in":"query"}],"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}}}},"post":{"operationId":"createItem","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}},"responses":{"201":{"description":"created","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}}}}}},"components":{"schemas":{"Item":{"type":"object","required":["id"],"properties":{"id":{"type":"string"},"active":{"type":"boolean"}}}}}}`
	if err := os.WriteFile(spec, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "sdk")
	files, err := GenerateOpenAPISDK(spec, out, SDKAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("files=%v", files)
	}
	ts, err := os.ReadFile(filepath.Join(out, "typescript/client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"export type Item", `"id": string`, `"active"?: boolean`, "getItem(item_id: string, locale?: string): Promise<Item>", "createItem(item_id: string, body: Item): Promise<Item>", "encodeURIComponent(item_id)", `params.set("locale", locale)`, "JSON.stringify(body)"} {
		if !strings.Contains(string(ts), want) {
			t.Errorf("TypeScript missing %q:\n%s", want, ts)
		}
	}
	dart, err := os.ReadFile(filepath.Join(out, "dart/lib/hydro_client.dart"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"abstract interface class HydroTransport", "Future<Item> getItem({", "required this.id", "factory Item.fromJson", "Uri.encodeComponent(item_id)", "required Object? body"} {
		if !strings.Contains(string(dart), want) {
			t.Errorf("Dart missing %q:\n%s", want, dart)
		}
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
