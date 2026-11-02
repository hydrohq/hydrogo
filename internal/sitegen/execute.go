package sitegen

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// RecordWriter captures the minimal collection creation capability needed by the generator.
type RecordWriter interface {
	CreateRecord(ctx context.Context, collectionID string, data map[string]any) (collections.Record, error)
}

// Execute writes the generated SitePlan into the tenant's store_settings and products collections.
func Execute(ctx context.Context, colSvc RecordWriter, scope tenant.Scope, plan SitePlan) error {
	if err := Validate(&plan); err != nil {
		return fmt.Errorf("validate site plan: %w", err)
	}

	tenantCtx := tenant.WithScope(ctx, scope)

	// 1. Prepare store_settings data
	settings := make(map[string]any, len(plan.StoreSettings)+15)
	for k, v := range plan.StoreSettings {
		settings[k] = v
	}

	storeName, _ := settings["store_name"].(string)
	if storeName == "" {
		storeName, _ = settings["brand_name"].(string)
	}
	if storeName == "" {
		storeName = "Vera Atelier"
	}
	settings["store_name"] = storeName
	settings["brand_name"] = storeName

	whatsapp, _ := settings["whatsapp_number"].(string)
	if whatsapp == "" {
		whatsapp = "+905550000000"
	}
	settings["whatsapp_number"] = whatsapp

	tagline, _ := settings["tagline"].(string)
	if tagline == "" {
		tagline, _ = settings["slogan"].(string)
	}
	if tagline != "" {
		settings["tagline"] = tagline
		settings["slogan"] = tagline
	}

	if plan.Theme.Preset != "" {
		settings["theme_preset"] = plan.Theme.Preset
	}
	if plan.Theme.FontPreset != "" {
		settings["font_preset"] = plan.Theme.FontPreset
	}
	if plan.Theme.AccentColor != "" {
		settings["primary_color"] = plan.Theme.AccentColor
		settings["accent_color"] = plan.Theme.AccentColor
	}
	if plan.Theme.BGColor != "" {
		settings["bg_tone"] = plan.Theme.BGColor
		settings["bg_primary_color"] = plan.Theme.BGColor
	}
	if plan.Theme.BorderRadius != "" {
		settings["border_radius_style"] = plan.Theme.BorderRadius
	}

	if len(plan.Layout) > 0 {
		rawLayout, err := json.Marshal(plan.Layout)
		if err == nil {
			settings["layout_json"] = string(rawLayout)
		}
	}

	if _, err := colSvc.CreateRecord(tenantCtx, "store_settings", settings); err != nil {
		return fmt.Errorf("save store_settings: %w", err)
	}

	// 2. Seed products
	for _, p := range plan.Products {
		prodMap := map[string]any{
			"title":       p.Title,
			"slug":        p.Slug,
			"price":       p.Price,
			"description": p.Description,
			"badge":       p.Badge,
			"image_url":   p.ImageURL,
			"in_stock":    true,
			"status":      contenttypes.StatusPublished,
		}
		if p.CompareAtPrice > 0 {
			prodMap["compare_at_price"] = p.CompareAtPrice
		}
		if _, err := colSvc.CreateRecord(tenantCtx, "products", prodMap); err != nil {
			return fmt.Errorf("create product %q: %w", p.Title, err)
		}
	}

	return nil
}
