package v1

import (
	"errors"
	"fmt"
	"regexp"
)

const (
	MaxRelationDepth = 3
	MaxRelationEdges = 16
	MaxRelationItems = 100
)

var relationIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

type RelationCardinality string

const (
	RelationOne  RelationCardinality = "one"
	RelationMany RelationCardinality = "many"
)

// RelationEdge is a provider-neutral lookup edge. It describes what may be
// traversed; it does not grant authorization or create a database join.
type RelationEdge struct {
	Name        string
	Target      string
	Cardinality RelationCardinality
	Depth       int
	MaxItems    int
}

// ValidateRelationPlan rejects unbounded relation graphs before a resolver or
// admin picker can execute them.
func ValidateRelationPlan(edges []RelationEdge) error {
	if len(edges) > MaxRelationEdges {
		return errors.New("relation edge budget exceeded")
	}
	seen := make(map[string]struct{}, len(edges))
	for _, edge := range edges {
		if !relationIdentifier.MatchString(edge.Name) || !relationIdentifier.MatchString(edge.Target) {
			return fmt.Errorf("invalid relation identifier")
		}
		if edge.Cardinality != RelationOne && edge.Cardinality != RelationMany {
			return fmt.Errorf("invalid relation cardinality for %q", edge.Name)
		}
		if edge.Depth < 1 || edge.Depth > MaxRelationDepth {
			return fmt.Errorf("relation depth for %q is out of bounds", edge.Name)
		}
		maxItems := edge.MaxItems
		if maxItems == 0 {
			maxItems = 25
		}
		if maxItems < 1 || maxItems > MaxRelationItems {
			return fmt.Errorf("relation item budget for %q is out of bounds", edge.Name)
		}
		if _, ok := seen[edge.Name]; ok {
			return fmt.Errorf("duplicate relation %q", edge.Name)
		}
		seen[edge.Name] = struct{}{}
	}
	return nil
}
