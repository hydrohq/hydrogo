package blueprints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
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
	case "corporate", "kurumsal", "b2b", "agency", "ajans":
		return ArchetypeCorporate
	case "restaurant", "restoran", "cafe", "kafe", "menu", "gastro":
		return ArchetypeRestaurant
	case "saas", "software", "yazilim", "app", "startup":
		return ArchetypeSaaS
	case "portfolio", "portfolyo", "studio", "stüdyo", "architect", "artist", "sanatci":
		return ArchetypePortfolio
	case "news", "haber", "portal", "medya", "gazete", "dergi", "press":
		return ArchetypeNews
	case "boutique", "eticaret", "e-ticaret", "ecommerce", "store", "magaza":
		return ArchetypeBoutique
	default:
		if strings.Contains(lower, "kurumsal") || strings.Contains(lower, "ajans") || strings.Contains(lower, "corp") || strings.Contains(lower, "b2b") {
			return ArchetypeCorporate
		}
		if strings.Contains(lower, "restoran") || strings.Contains(lower, "restaurant") || strings.Contains(lower, "menu") || strings.Contains(lower, "kafe") || strings.Contains(lower, "cafe") || strings.Contains(lower, "bistro") {
			return ArchetypeRestaurant
		}
		if strings.Contains(lower, "saas") || strings.Contains(lower, "yazilim") || strings.Contains(lower, "software") || strings.Contains(lower, "startup") {
			return ArchetypeSaaS
		}
		if strings.Contains(lower, "portfolyo") || strings.Contains(lower, "portfolio") || strings.Contains(lower, "sanat") || strings.Contains(lower, "mimar") || strings.Contains(lower, "studio") {
			return ArchetypePortfolio
		}
		if strings.Contains(lower, "haber") || strings.Contains(lower, "gazete") || strings.Contains(lower, "news") || strings.Contains(lower, "portal") || strings.Contains(lower, "medya") {
			return ArchetypeNews
		}
		return ArchetypeBoutique
	}
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

// SeedArchetypeSite provisions demo data tailored to the chosen archetype.
func SeedArchetypeSite(ctx context.Context, colSvc collections.Service, scope tenant.Scope, archetype, name, phone, prompt string) error {
	arch := NormalizeArchetype(archetype)
	switch arch {
	case ArchetypeCorporate:
		return SeedCorporateStore(ctx, colSvc, scope, name, phone, prompt)
	case ArchetypeRestaurant:
		return SeedRestaurantStore(ctx, colSvc, scope, name, phone, prompt)
	case ArchetypeSaaS:
		return SeedSaaSStore(ctx, colSvc, scope, name, phone, prompt)
	case ArchetypePortfolio:
		return SeedPortfolioStore(ctx, colSvc, scope, name, phone, prompt)
	case ArchetypeNews:
		return SeedNewsStore(ctx, colSvc, scope, name, phone, prompt)
	default:
		settings, products := GenerateLuxuryCatalog(name, phone, prompt)
		return SeedBoutiqueStore(ctx, colSvc, scope, settings, products)
	}
}
