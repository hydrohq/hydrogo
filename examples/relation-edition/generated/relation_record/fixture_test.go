package generated

import "testing"

func TestRelationRecordFixtureShape(t *testing.T) {
	item := RelationRecord{ID: "fixture-relation_record"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}
