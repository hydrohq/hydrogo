package adminstudio

import (
	"reflect"
	"testing"
)

func TestValidateRelationQueryBoundsAndDefaults(t *testing.T) {
	got, err := ValidateRelationQuery(RelationQuery{Collection: " products ", Query: " shoe "})
	if err != nil {
		t.Fatal(err)
	}
	want := RelationQuery{Collection: "products", Query: "shoe", Limit: DefaultRelationLimit}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("query = %#v, want %#v", got, want)
	}
}

func TestValidateRelationQueryRejectsUnsafeAndUnboundedInput(t *testing.T) {
	if _, err := ValidateRelationQuery(RelationQuery{Collection: "products/x"}); err != ErrInvalidRelationQuery {
		t.Fatalf("unsafe collection error = %v", err)
	}
	if _, err := ValidateRelationQuery(RelationQuery{Collection: "products", Limit: MaxRelationLimit + 1}); err != ErrInvalidRelationQuery {
		t.Fatalf("limit error = %v", err)
	}
}

func TestNormalizeRelationOptionsIsDeterministicAndRejectsDuplicates(t *testing.T) {
	got, err := NormalizeRelationOptions([]RelationOption{{ID: "b", Label: " B "}, {ID: "a", Label: "A"}}, 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []RelationOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("options = %#v, want %#v", got, want)
	}
	if _, err := NormalizeRelationOptions([]RelationOption{{ID: "a", Label: "A"}, {ID: "a", Label: "Again"}}, 10); err != ErrInvalidRelationQuery {
		t.Fatalf("duplicate error = %v", err)
	}
}
