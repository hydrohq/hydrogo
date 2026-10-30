package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// FactoryOptions configures a single-command Boutique Store provisioning run.
type FactoryOptions struct {
	Dialect         string
	DSN             string
	TenantID        string
	DisplayName     string
	AdminEmail      string
	AdminPassword   string
	Subdomain       string
	CustomDomain    string
	Plan            string
	OwnerPhone      string
	Prompt          string
	ContentTypesDir string
	TrialDays       int
}

// FactoryResult summarizes the newly provisioned boutique store workspace.
type FactoryResult struct {
	TenantID          string `json:"tenant_id"`
	DisplayName       string `json:"display_name"`
	Subdomain         string `json:"subdomain"`
	CustomDomain      string `json:"custom_domain,omitempty"`
	AdminEmail        string `json:"admin_email"`
	GeneratedPassword string `json:"generated_password,omitempty"`
	ThemePreset       string `json:"theme_preset"`
	CategoriesSeeded  int    `json:"categories_seeded"`
	ProductsSeeded    int    `json:"products_seeded"`
	StoreSettingsID   string `json:"store_settings_id"`
}

func main() {
	var opts FactoryOptions
	flag.StringVar(&opts.Dialect, "dialect", "sqlite", "Database dialect (sqlite, postgres, mysql)")
	flag.StringVar(&opts.DSN, "dsn", "hydro.db", "Database DSN or SQLite file path")
	flag.StringVar(&opts.TenantID, "tenant", "", "Unique workspace tenant ID (e.g. ipek-butik)")
	flag.StringVar(&opts.DisplayName, "name", "", "Store display name (e.g. İpek Butik)")
	flag.StringVar(&opts.AdminEmail, "email", "", "Store owner admin email")
	flag.StringVar(&opts.AdminPassword, "password", "", "Optional initial admin password (auto-generated if empty)")
	flag.StringVar(&opts.Subdomain, "subdomain", "", "Platform subdomain (defaults to <tenant>.butik.local)")
	flag.StringVar(&opts.CustomDomain, "domain", "", "Optional custom domain (e.g. www.ipekbutik.com)")
	flag.StringVar(&opts.Plan, "plan", "pro", "Subscription plan preset (starter, pro, vip)")
	flag.StringVar(&opts.OwnerPhone, "phone", "+905550000000", "WhatsApp order & owner phone (+905...)")
	flag.StringVar(&opts.Prompt, "prompt", "Kadın ipek gömlek ve keten takım satan zarif bir butik", "Natural language store description for theme and catalog seeding")
	flag.StringVar(&opts.ContentTypesDir, "content-types-dir", "content-types/boutique", "Path to boutique content-type YAML schemas")
	flag.IntVar(&opts.TrialDays, "trial-days", 14, "Trial period in days")
	flag.Parse()

	res, err := RunFactory(context.Background(), opts, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory error: %v\n", err)
		os.Exit(1)
	}
	_ = res
}

// RunFactory provisions a complete tenant boutique workspace, syncs boutique schemas, and seeds theme & products.
func RunFactory(ctx context.Context, opts FactoryOptions, out io.Writer) (FactoryResult, error) {
	opts.TenantID = strings.ToLower(strings.TrimSpace(opts.TenantID))
	if opts.TenantID == "" {
		return FactoryResult{}, errors.New("--tenant is required (e.g. --tenant ipek-butik)")
	}
	if strings.TrimSpace(opts.DisplayName) == "" {
		opts.DisplayName = opts.TenantID
	}
	opts.AdminEmail = strings.ToLower(strings.TrimSpace(opts.AdminEmail))
	if opts.AdminEmail == "" {
		opts.AdminEmail = "owner@" + opts.TenantID + ".local"
	}
	if strings.TrimSpace(opts.Subdomain) == "" {
		opts.Subdomain = opts.TenantID + ".butik.local"
	}
	if strings.TrimSpace(opts.Plan) == "" {
		opts.Plan = tenant.PlanPro
	}
	if strings.TrimSpace(opts.OwnerPhone) == "" {
		opts.OwnerPhone = "+905550000000"
	}
	if strings.TrimSpace(opts.ContentTypesDir) == "" {
		opts.ContentTypesDir = "content-types/boutique"
	}
	dialect := database.Dialect(strings.ToLower(strings.TrimSpace(opts.Dialect)))
	if dialect == "" {
		dialect = database.SQLite
	}

	db, err := database.Open(ctx, dialect, opts.DSN)
	if err != nil {
		return FactoryResult{}, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		return FactoryResult{}, fmt.Errorf("apply migrations: %w", err)
	}

	generatedPassword := ""
	password := strings.TrimSpace(opts.AdminPassword)
	if password == "" {
		var raw [9]byte
		_, _ = rand.Read(raw[:])
		password = "Hyd!" + hex.EncodeToString(raw[:])
		generatedPassword = password
	}
	hash, err := auth.HashPassword(password, auth.DefaultArgon2Params)
	if err != nil {
		return FactoryResult{}, fmt.Errorf("hash password: %w", err)
	}

	if _, err := tenant.Provision(ctx, db, string(dialect), tenant.ProvisionRequest{
		TenantID:          opts.TenantID,
		AdminEmail:        opts.AdminEmail,
		AdminPasswordHash: hash,
		Locale:            "tr",
		Timezone:          "Europe/Istanbul",
	}); err != nil {
		return FactoryResult{}, fmt.Errorf("provision tenant: %w", err)
	}

	gov := tenant.DefaultGovernance(opts.TenantID)
	gov.DisplayName = strings.TrimSpace(opts.DisplayName)
	gov.Subdomain = strings.ToLower(strings.TrimSpace(opts.Subdomain))
	gov.CustomDomain = strings.ToLower(strings.TrimSpace(opts.CustomDomain))
	gov.Blueprint = "boutique"
	tenant.ApplyPlanPreset(&gov, opts.Plan)
	gov.OwnerPhone = strings.TrimSpace(opts.OwnerPhone)
	if opts.TrialDays > 0 {
		gov.TrialEndsAt = time.Now().UTC().Add(time.Duration(opts.TrialDays) * 24 * time.Hour).Format(time.RFC3339)
	}
	if err := tenant.UpsertGovernance(ctx, db, string(dialect), gov); err != nil {
		return FactoryResult{}, fmt.Errorf("upsert governance: %w", err)
	}

	defs, err := contenttypes.LoadDir(opts.ContentTypesDir)
	if err != nil {
		return FactoryResult{}, fmt.Errorf("load boutique content types from %s: %w", opts.ContentTypesDir, err)
	}
	if len(defs) == 0 {
		return FactoryResult{}, fmt.Errorf("no boutique YAML schemas found in %s", opts.ContentTypesDir)
	}
	schemas, err := contenttypes.Schemas(defs)
	if err != nil {
		return FactoryResult{}, fmt.Errorf("compile boutique schemas: %w", err)
	}

	scope, err := tenant.NewScope(opts.TenantID, opts.TenantID)
	if err != nil {
		return FactoryResult{}, err
	}
	tenantCtx := tenant.WithScope(ctx, scope)

	colSvc := collections.Service{
		Repository: collectionstore.Store{
			DB:                 db,
			Dialect:            dialect,
			RequireTenantScope: true,
		},
	}
	if err := colSvc.SyncSchemas(tenantCtx, schemas); err != nil {
		return FactoryResult{}, fmt.Errorf("sync boutique schemas: %w", err)
	}

	themePreset, settingsData, categories, products := buildBoutiqueCatalogFromPrompt(opts.Prompt, opts.DisplayName, opts.OwnerPhone)

	settingsRec, err := colSvc.CreateRecord(tenantCtx, "store_settings", settingsData)
	if err != nil {
		return FactoryResult{}, fmt.Errorf("seed store_settings: %w", err)
	}

	catIDs := make([]string, 0, len(categories))
	for _, cat := range categories {
		createdCat, err := colSvc.CreateRecord(tenantCtx, "boutique_categories", cat)
		if err != nil {
			return FactoryResult{}, fmt.Errorf("seed boutique_categories: %w", err)
		}
		catIDs = append(catIDs, createdCat.ID)
	}

	prodCount := 0
	for i, prod := range products {
		catID := catIDs[0]
		if i < len(catIDs) {
			catID = catIDs[i]
		}
		prod["category"] = catID
		prod["status"] = contenttypes.StatusPublished
		if _, err := colSvc.CreateRecord(tenantCtx, "boutique_products", prod); err != nil {
			return FactoryResult{}, fmt.Errorf("seed boutique_products: %w", err)
		}
		prodCount++
	}

	result := FactoryResult{
		TenantID:          opts.TenantID,
		DisplayName:       gov.DisplayName,
		Subdomain:         gov.Subdomain,
		CustomDomain:      gov.CustomDomain,
		AdminEmail:        opts.AdminEmail,
		GeneratedPassword: generatedPassword,
		ThemePreset:       themePreset,
		CategoriesSeeded:  len(catIDs),
		ProductsSeeded:    prodCount,
		StoreSettingsID:   settingsRec.ID,
	}

	if out != nil {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	}
	return result, nil
}

func buildBoutiqueCatalogFromPrompt(prompt, displayName, phone string) (string, map[string]any, []map[string]any, []map[string]any) {
	lower := strings.ToLower(prompt)

	themePreset := "minimal_cream"
	bgColor := "#FAF7F2"
	textColor := "#1C1917"
	accentColor := "#8C6D46"
	fontHeading := "Cormorant Garamond"
	radiusStyle := "soft"

	switch {
	case strings.Contains(lower, "siyah") || strings.Contains(lower, "koyu") || strings.Contains(lower, "altın") || strings.Contains(lower, "noir") || strings.Contains(lower, "gece"):
		themePreset = "noir_gold"
		bgColor = "#111111"
		textColor = "#F5F2EB"
		accentColor = "#D4AF37"
		fontHeading = "Bodoni Moda"
		radiusStyle = "sharp"
	case strings.Contains(lower, "pembe") || strings.Contains(lower, "pastel") || strings.Contains(lower, "blush") || strings.Contains(lower, "çiçek"):
		themePreset = "pastel_blush"
		bgColor = "#FFF9F9"
		textColor = "#2D2326"
		accentColor = "#C97A8E"
		fontHeading = "Playfair Display"
		radiusStyle = "pill"
	case strings.Contains(lower, "zümrüt") || strings.Contains(lower, "yeşil") || strings.Contains(lower, "emerald"):
		themePreset = "emerald_silk"
		bgColor = "#F4F8F6"
		textColor = "#102820"
		accentColor = "#1B5E43"
		fontHeading = "Cormorant Garamond"
		radiusStyle = "soft"
	}

	settings := map[string]any{
		"store_name":              displayName,
		"slogan":                  "Zamansız Zarafet & Seçkin Tasarımlar",
		"whatsapp_number":         phone,
		"instagram_handle":        "@" + strings.ReplaceAll(strings.ToLower(displayName), " ", ""),
		"announcement_bar":        "1500 TL ve Üzeri Siparişlerde Ücretsiz Kargo · WhatsApp İle Hızlı Sipariş",
		"free_shipping_threshold": 1500.0,
		"theme_preset":            themePreset,
		"bg_primary_color":        bgColor,
		"text_primary_color":      textColor,
		"accent_color":            accentColor,
		"font_heading":            fontHeading,
		"border_radius_style":     radiusStyle,
		"hero_title":              displayName + " Yeni Sezon Koleksiyonu",
		"hero_cta_text":           "Koleksiyonu Keşfet",
	}

	categories := []map[string]any{
		{
			"name":       "Yeni Sezon",
			"slug":       "yeni-sezon",
			"sort_order": 1.0,
			"featured":   true,
		},
		{
			"name":       "Üst Giyim & Gömlek",
			"slug":       "ust-giyim",
			"sort_order": 2.0,
			"featured":   true,
		},
		{
			"name":       "Elbise & Takım",
			"slug":       "elbise-takim",
			"sort_order": 3.0,
			"featured":   true,
		},
	}

	products := []map[string]any{
		{
			"title":            "Saf İpek Kruvaze Gömlek",
			"slug":             "saf-ipek-kruvaze-gomlek",
			"sku":              "BTK-101",
			"price":            1850.0,
			"compare_at_price": 2200.0,
			"sizes":            "S, M, L",
			"colors":           "Ekru, Siyah, Çağla Yeşili",
			"stock_quantity":   14.0,
			"badge":            "Yeni Sezon",
			"fabric_care":      "%100 Saf İpek dokuma. Elde soğuk suda yıkama veya kuru temizleme önerilir.",
			"description":      "Doğal dokusu ve dökümlü kesimiyle gün boyu zarafet sunan özel tasarım ipek gömlek.",
			"featured":         true,
		},
		{
			"title":          "Yüksek Bel Keten Palazzo Pantolon",
			"slug":           "yuksek-bel-keten-palazzo-pantolon",
			"sku":            "BTK-102",
			"price":          1490.0,
			"sizes":          "S, M, L, XL",
			"colors":         "Bej, Taş Rengi, Lacivert",
			"stock_quantity": 20.0,
			"badge":          "Özel Tasarım",
			"fabric_care":    "%100 Doğal Keten. 30 derecede hassas yıkama.",
			"description":    "Nefes alan keten kumaşı ve rahat kalıbıyla yaz-bahar gardırobunuzun vazgeçilmez parçası.",
			"featured":       true,
		},
		{
			"title":            "Saten Dokulu Midi Boy Abiye Elbise",
			"slug":             "saten-dokulu-midi-boy-abiye-elbise",
			"sku":              "BTK-103",
			"price":            2450.0,
			"compare_at_price": 2950.0,
			"sizes":            "S, M, L",
			"colors":           "Şampanya, Zümrüt, Siyah",
			"stock_quantity":   3.0,
			"badge":            "Son 3 Ürün",
			"fabric_care":      "Yalnızca kuru temizleme.",
			"description":      "Özel davetler için tasarlanmış, zarif sırt detaylı midi boy saten elbise.",
			"featured":         true,
		},
		{
			"title":          "İtalyan Kesim Keten Blazer Ceket",
			"slug":           "italyan-kesim-keten-blazer-ceket",
			"sku":            "BTK-104",
			"price":          2190.0,
			"sizes":          "S, M, L",
			"colors":         "Krem, Kahve",
			"stock_quantity": 9.0,
			"badge":          "Yeni Sezon",
			"fabric_care":    "%85 Keten, %15 Pamuk astar.",
			"description":    "Hem günlük şıklıkta hem ofis kombinlerinde kullanabileceğiniz zamansız keten blazer.",
			"featured":       false,
		},
	}

	return themePreset, settings, categories, products
}
