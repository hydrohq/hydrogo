package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOpenAPI(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, "openapi.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckAPICompatibilityAllowsAdditiveChanges(t *testing.T) {
	root := t.TempDir()
	base := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}}}}}}}}}}}`
	current := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"},"title":{"type":"string"}}}}}}}}},"/api/v1/news":{"get":{"responses":{"200":{"description":"ok"}}}}}}`
	report, err := CheckAPICompatibility(writeOpenAPI(t, root, base), writeOpenAPI(t, filepath.Join(root, "current"), current))
	if err != nil {
		t.Fatal(err)
	}
	if report.PathsChecked != 1 || report.MethodsChecked != 1 || report.PropertiesChecked != 1 || report.ParametersChecked != 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestCheckAPICompatibilityRejectsRemovedRequestParameter(t *testing.T) {
	root := t.TempDir()
	base := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"get":{"parameters":[{"name":"page","in":"query","required":false}],"responses":{"200":{"description":"ok"}}}}}}`
	current := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"get":{"responses":{"200":{"description":"ok"}}}}}}`
	if _, err := CheckAPICompatibility(writeOpenAPI(t, root, base), writeOpenAPI(t, filepath.Join(root, "current"), current)); err == nil {
		t.Fatal("expected removed request parameter error")
	}
}

func TestCheckAPICompatibilityRejectsNewlyRequiredParameter(t *testing.T) {
	root := t.TempDir()
	base := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"get":{"parameters":[{"name":"locale","in":"query","required":false}],"responses":{"200":{"description":"ok"}}}}}}`
	current := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"get":{"parameters":[{"name":"locale","in":"query","required":true}],"responses":{"200":{"description":"ok"}}}}}}`
	if _, err := CheckAPICompatibility(writeOpenAPI(t, root, base), writeOpenAPI(t, filepath.Join(root, "current"), current)); err == nil {
		t.Fatal("expected newly required parameter error")
	}
}

func TestCheckAPICompatibilityRejectsRemovedOperation(t *testing.T) {
	root := t.TempDir()
	base := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{"post":{"responses":{"201":{"description":"created"}}}}}}`
	current := `{"openapi":"3.1.0","paths":{"/api/v1/pages":{}}}`
	if _, err := CheckAPICompatibility(writeOpenAPI(t, root, base), writeOpenAPI(t, filepath.Join(root, "current"), current)); err == nil {
		t.Fatal("expected removed operation error")
	}
}

func TestCheckAPICompatibilityResolvesLocalResponseSchemaReferences(t *testing.T) {
	root := t.TempDir()
	baseline := writeOpenAPI(t, filepath.Join(root, "baseline"), `{"openapi":"3.1.0","paths":{"/api/v1/devices":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/DeviceList"}}}}}}}},"components":{"schemas":{"DeviceList":{"type":"object","properties":{"devices":{"type":"array"}}}}}}`)
	current := writeOpenAPI(t, filepath.Join(root, "current"), `{"openapi":"3.1.0","paths":{"/api/v1/devices":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/DeviceList"}}}}}}}},"components":{"schemas":{"DeviceList":{"type":"object","properties":{}}}}}`)
	if _, err := CheckAPICompatibility(baseline, current); err == nil || !strings.Contains(err.Error(), "devices") {
		t.Fatalf("CheckAPICompatibility error = %v, want removed referenced response property", err)
	}
}
