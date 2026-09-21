package eventcatalog

import (
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
