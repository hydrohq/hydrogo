package eventcatalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestCatalogAcceptsOnlyPublishedTopicVersions(t *testing.T) {
	if err := Validate("tenant.provisioned", 1); err != nil {
		t.Fatal(err)
	}
	if err := Validate("tenant.provisioned", 2); !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("version = %v", err)
	}
	if err := Validate("made.up", 1); !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("topic = %v", err)
	}
}

func TestExportJSONIsVersionedAndDeterministic(t *testing.T) {
	first, err := ExportJSON()
	if err != nil {
		t.Fatal(err)
	}
	second, err := ExportJSON()
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("catalog export changed: %s / %s, err=%v", first, second, err)
	}
	var document Document
	if err := json.Unmarshal(first, &document); err != nil || document.Version != "v1" || len(document.Events) != len(Definitions()) {
		t.Fatalf("catalog document = %#v, err=%v", document, err)
	}
}

func TestCompareBaselineRequiresPublishedEventsToRemainCompatible(t *testing.T) {
	baseline := []Definition{{Topic: "page.published", PayloadVersion: 1}, {Topic: "user.invited", PayloadVersion: 1, PII: true}}
	current := append(append([]Definition(nil), baseline...), Definition{Topic: "page.published", PayloadVersion: 2})
	if err := CompareBaseline(baseline, current); err != nil {
		t.Fatal(err)
	}
	if err := CompareBaseline(baseline, current[1:]); !errors.Is(err, ErrBreakingCatalog) {
		t.Fatalf("removed event = %v", err)
	}
	changed := append([]Definition(nil), current...)
	changed[1].PII = false
	if err := CompareBaseline(baseline, changed); !errors.Is(err, ErrBreakingCatalog) {
		t.Fatalf("PII classification changed = %v", err)
	}
}
