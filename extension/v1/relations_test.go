package v1

import "testing"

func TestValidateRelationPlanBoundsAndCardinality(t *testing.T) {
	if err := ValidateRelationPlan([]RelationEdge{{Name: "author", Target: "users", Cardinality: RelationOne, Depth: 1}}); err != nil {
		t.Fatal(err)
	}
	cases := []RelationEdge{
		{Name: "author", Target: "users", Cardinality: "many_to_many", Depth: 1},
		{Name: "author", Target: "users", Cardinality: RelationOne, Depth: MaxRelationDepth + 1},
		{Name: "author", Target: "users", Cardinality: RelationMany, Depth: 1, MaxItems: MaxRelationItems + 1},
	}
	for _, edge := range cases {
		if err := ValidateRelationPlan([]RelationEdge{edge}); err == nil {
			t.Fatalf("edge %#v was accepted", edge)
		}
	}
	if err := ValidateRelationPlan([]RelationEdge{{Name: "author", Target: "users", Cardinality: RelationOne, Depth: 1}, {Name: "author", Target: "teams", Cardinality: RelationOne, Depth: 1}}); err == nil {
		t.Fatal("duplicate relation was accepted")
	}
}
