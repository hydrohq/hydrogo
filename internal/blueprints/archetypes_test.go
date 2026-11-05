package blueprints

import (
	"testing"
)

func TestArchetypeDefinitions(t *testing.T) {
	types := []string{
		ArchetypeBoutique,
		ArchetypeCorporate,
		ArchetypeRestaurant,
		ArchetypeSaaS,
		ArchetypePortfolio,
	}

	for _, arch := range types {
		defs := ArchetypeDefinitions(arch)
		if len(defs) == 0 {
			t.Errorf("archetype %s returned 0 definitions", arch)
		}
	}
}

func TestNormalizeArchetype(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"kurumsal", ArchetypeCorporate},
		{"B2B Ajans", ArchetypeCorporate},
		{"restoran", ArchetypeRestaurant},
		{"Gourmet Cafe", ArchetypeRestaurant},
		{"saas startup", ArchetypeSaaS},
		{"yazilim", ArchetypeSaaS},
		{"mimarlik portfolyo", ArchetypePortfolio},
		{"sanatci", ArchetypePortfolio},
		{"butik", ArchetypeBoutique},
		{"unknown", ArchetypeBoutique},
	}

	for _, tt := range tests {
		got := NormalizeArchetype(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeArchetype(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
