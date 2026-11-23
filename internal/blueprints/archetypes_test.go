package blueprints

import (
	"testing"

	"github.com/hydrohq/hydrogo/internal/contenttypes"
)

func TestArchetypeDefinitions(t *testing.T) {
	types := []string{
		ArchetypeBoutique,
		ArchetypeCorporate,
		ArchetypeRestaurant,
		ArchetypeSaaS,
		ArchetypePortfolio,
		ArchetypeNews,
	}

	for _, arch := range types {
		t.Run(arch, func(t *testing.T) {
			defs := ArchetypeDefinitions(arch)
			if len(defs) == 0 {
				t.Fatalf("archetype %s returned 0 definitions", arch)
			}
			if _, err := contenttypes.Schemas(defs); err != nil {
				t.Fatalf("contenttypes.Schemas(%s): %v", arch, err)
			}

			var foundSettingsSingleton bool
			var foundCategoryRelation bool
			for _, def := range defs {
				if err := def.Validate(); err != nil {
					t.Fatalf("def %s.%s Validate(): %v", arch, def.ID, err)
				}
				adminCol := def.AdminCollection()
				if def.ID == "store_settings" || def.ID == "site_settings" {
					if !def.Singleton {
						t.Errorf("%s.%s expected Singleton=true", arch, def.ID)
					}
					if adminCol.Type != "global" {
						t.Errorf("%s.%s expected AdminCollection Type=global, got %q", arch, def.ID, adminCol.Type)
					}
					foundSettingsSingleton = true
				}
				if (arch == ArchetypeNews && def.ID == "news_articles") || (arch == ArchetypeRestaurant && def.ID == "menu_items") {
					for _, f := range adminCol.Fields {
						if f.ID == "category_slug" {
							if f.Widget != "relation" || f.Relation == nil || f.Relation.ValueField != "slug" {
								t.Errorf("%s.%s category_slug expected relation widget with value_field=slug, got widget=%q rel=%+v", arch, def.ID, f.Widget, f.Relation)
							}
							foundCategoryRelation = true
						}
					}
				}
			}
			if !foundSettingsSingleton {
				t.Errorf("archetype %s missing singleton settings collection", arch)
			}
			if (arch == ArchetypeNews || arch == ArchetypeRestaurant) && !foundCategoryRelation {
				t.Errorf("archetype %s missing category_slug relation field", arch)
			}
		})
	}
}

func TestNormalizeArchetype(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"corporate", ArchetypeCorporate},
		{"B2B Agency", ArchetypeCorporate},
		{"restaurant", ArchetypeRestaurant},
		{"Gourmet Cafe", ArchetypeRestaurant},
		{"saas startup", ArchetypeSaaS},
		{"software", ArchetypeSaaS},
		{"architecture portfolio", ArchetypePortfolio},
		{"artist", ArchetypePortfolio},
		{"daily newspaper", ArchetypeNews},
		{"boutique", ArchetypeBoutique},
		{"unknown", ArchetypeBoutique},
	}

	for _, tt := range tests {
		got := NormalizeArchetype(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeArchetype(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
