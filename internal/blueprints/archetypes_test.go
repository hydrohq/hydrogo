package blueprints

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hydrohq/hydrogo/internal/collections"
	collectionstore "github.com/hydrohq/hydrogo/internal/collections/sqlstore"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/migrations"
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

func TestSeedEveryArchetypeIntoSQLite(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "blueprints.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("ApplyMigrations: %v", err)
	}

	colSvc := collections.Service{
		Repository: collectionstore.Store{
			DB:                 db,
			Dialect:            database.SQLite,
			RequireTenantScope: true,
		},
	}

	arches := []string{
		ArchetypeBoutique,
		ArchetypeCorporate,
		ArchetypeRestaurant,
		ArchetypeSaaS,
		ArchetypePortfolio,
		ArchetypeNews,
	}

	for _, arch := range arches {
		t.Run(arch, func(t *testing.T) {
			scope, err := tenant.NewScope("t-"+arch, "t-"+arch)
			if err != nil {
				t.Fatal(err)
			}
			if err := EnsureArchetypeSchemas(ctx, colSvc, scope, arch); err != nil {
				t.Fatalf("EnsureArchetypeSchemas(%s): %v", arch, err)
			}
			if err := SeedArchetypeSite(ctx, colSvc, scope, arch, "Demo "+arch, "+905551112233", "test prompt"); err != nil {
				t.Fatalf("SeedArchetypeSite(%s): %v", arch, err)
			}
		})
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
