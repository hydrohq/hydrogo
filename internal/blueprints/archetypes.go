package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/tenant"
)

const (
	ArchetypeBoutique   = "boutique"
	ArchetypeCorporate  = "corporate"
	ArchetypeRestaurant = "restaurant"
	ArchetypeSaaS       = "saas"
	ArchetypePortfolio  = "portfolio"
	ArchetypeNews       = "news"
)

// NormalizeArchetype standardizes an incoming archetype/site_type string.
func NormalizeArchetype(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch lower {
	case "corporate", "b2b", "agency", "consulting":
		return ArchetypeCorporate
	case "restaurant", "cafe", "menu", "gastro", "bistro":
		return ArchetypeRestaurant
	case "saas", "software", "app", "startup":
		return ArchetypeSaaS
	case "portfolio", "studio", "architect", "artist", "photographer":
		return ArchetypePortfolio
	case "news", "portal", "media", "magazine", "newspaper", "press":
		return ArchetypeNews
	case "boutique", "ecommerce", "e-commerce", "store", "shop":
		return ArchetypeBoutique
	}
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(lower, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("corp", "b2b", "agency", "consult"):
		return ArchetypeCorporate
	case has("restaurant", "menu", "cafe", "bistro"):
		return ArchetypeRestaurant
	case has("saas", "software", "startup"):
		return ArchetypeSaaS
	case has("portfolio", "architect", "artist", "photograph", "studio"):
		return ArchetypePortfolio
	case has("news", "magazine", "portal", "press"):
		return ArchetypeNews
	}
	return ArchetypeBoutique
}

// ArchetypeDefinitions returns the content type definitions for the specified archetype.
func ArchetypeDefinitions(archetype string) []contenttypes.Definition {
	switch NormalizeArchetype(archetype) {
	case ArchetypeCorporate:
		return CorporateDefinitions()
	case ArchetypeRestaurant:
		return RestaurantDefinitions()
	case ArchetypeSaaS:
		return SaaSDefinitions()
	case ArchetypePortfolio:
		return PortfolioDefinitions()
	case ArchetypeNews:
		return NewsDefinitions()
	default:
		return BoutiqueDefinitions()
	}
}

// EnsureArchetypeSchemas provisions or updates the schema collections for any archetype.
func EnsureArchetypeSchemas(ctx context.Context, colSvc collections.Service, scope tenant.Scope, archetype string) error {
	defs := ArchetypeDefinitions(archetype)
	schemas, err := contenttypes.Schemas(defs)
	if err != nil {
		return fmt.Errorf("compile %s schemas: %w", archetype, err)
	}

	tenantCtx := tenant.WithScope(ctx, scope)
	if err := colSvc.SyncSchemas(tenantCtx, schemas); err != nil {
		return fmt.Errorf("sync %s collections: %w", archetype, err)
	}
	return nil
}
