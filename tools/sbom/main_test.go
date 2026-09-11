package main

import (
	"encoding/json"
	"testing"
)

func TestBuildDocumentIsDeterministicAndSPDXShaped(t *testing.T) {
	modules := []module{
		{Path: "example/z", Version: "v1.0.0"},
		{Path: "example/a", Version: "v2.0.0"},
	}
	first, err := json.Marshal(buildDocument(modules))
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(buildDocument(modules))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("SBOM document is not deterministic")
	}
	doc := buildDocument(modules)
	if doc.SPDXVersion != "SPDX-2.3" || len(doc.Packages) != 2 || len(doc.Relationships) != 2 {
		t.Fatalf("unexpected SPDX document shape: %#v", doc)
	}
}
