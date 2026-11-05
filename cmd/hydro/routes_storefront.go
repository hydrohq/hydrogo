package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/blueprints"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/storefront"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func (a *serveApp) storefrontRoutes() []router.Route {
	routes := []router.Route{
		{Pattern: "GET /{$}", Access: router.Public, Handler: http.HandlerFunc(a.storefrontHandler)},
		{Pattern: "GET /showcase", Access: router.Public, Handler: http.HandlerFunc(a.showcasePageHandler)},
		{Pattern: "GET /demo", Access: router.Public, Handler: http.HandlerFunc(a.showcasePageHandler)},
		{Pattern: "GET /urun/{slug}", Access: router.Public, Handler: http.HandlerFunc(a.storefrontProductHandler)},
		{Pattern: "GET /products/{slug}", Access: router.Public, Handler: http.HandlerFunc(a.storefrontProductHandler)},
		{Pattern: "GET /kategori/{slug}", Access: router.Public, Handler: http.HandlerFunc(a.storefrontCategoryHandler)},
		{Pattern: "GET /collections/{slug}", Access: router.Public, Handler: http.HandlerFunc(a.storefrontCategoryHandler)},
		{Pattern: "GET /hakkimizda", Access: router.Public, Handler: http.HandlerFunc(a.storefrontAboutHandler)},
		{Pattern: "GET /about", Access: router.Public, Handler: http.HandlerFunc(a.storefrontAboutHandler)},
		{Pattern: "GET /iletisim", Access: router.Public, Handler: http.HandlerFunc(a.storefrontContactHandler)},
		{Pattern: "GET /contact", Access: router.Public, Handler: http.HandlerFunc(a.storefrontContactHandler)},
		{Pattern: "GET /siparis-takip", Access: router.Public, Handler: http.HandlerFunc(a.storefrontOrderTrackingHandler)},
		{Pattern: "GET /order-tracking", Access: router.Public, Handler: http.HandlerFunc(a.storefrontOrderTrackingHandler)},
		{Pattern: "GET /robots.txt", Access: router.Public, Handler: http.HandlerFunc(a.storefrontRobotsHandler)},
		{Pattern: "GET /storefront/fonts/{name}", Access: router.Public, Handler: http.HandlerFunc(storefront.ServeFont)},
		{Pattern: "POST /api/v1/storefront/order", Access: router.Public, Handler: http.HandlerFunc(a.storefrontOrderHandler)},
		{Pattern: "POST /api/v1/store/orders", Access: router.Public, Handler: http.HandlerFunc(a.storefrontOrderHandler)},
		{Pattern: "POST /api/v1/storefront/track-order", Access: router.Public, Handler: http.HandlerFunc(a.storefrontTrackOrderAPIHandler)},
		{Pattern: "POST /api/v1/inquire", Access: router.Public, Handler: http.HandlerFunc(a.storefrontInquireHandler)},
		{Pattern: "POST /api/v1/reserve", Access: router.Public, Handler: http.HandlerFunc(a.storefrontReserveHandler)},
		{Pattern: "POST /api/v1/waitlist", Access: router.Public, Handler: http.HandlerFunc(a.storefrontWaitlistHandler)},
	}
	if !a.modules["site"] {
		routes = append(routes, router.Route{Pattern: "GET /sitemap.xml", Access: router.Public, Handler: http.HandlerFunc(a.storefrontSitemapHandler)})
	}
	if !a.modules["pages"] {
		routes = append(routes, router.Route{Pattern: "GET /llms.txt", Access: router.Public, Handler: http.HandlerFunc(a.storefrontLLMsHandler)})
	}
	return routes
}

func (a *serveApp) isStorefrontTenant(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if i := strings.IndexByte(host, ':'); i > 0 {
		host = host[:i]
	}
	consoleHost := strings.ToLower(strings.TrimSpace(a.cfg.Factory.ConsoleHost))
	if consoleHost == "" {
		consoleHost = "console.example.com"
	}
	if host == consoleHost {
		return false
	}
	scope, err := tenant.FromContext(r.Context())
	if err == nil && scope.TenantID != "" && scope.TenantID != "default" && scope.TenantID != "platform" {
		return true
	}
	if a.db != nil {
		if _, ok := tenant.ResolveTenantByDomain(r.Context(), a.db, string(a.cfg.Database.Dialect), host); ok {
			return true
		}
	}
	return false
}

func (a *serveApp) resolveStorefrontContext(w http.ResponseWriter, r *http.Request) (tenant.Scope, tenant.Governance, map[string]any, []storefront.ProductView, bool) {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if i := strings.IndexByte(host, ':'); i > 0 {
		host = host[:i]
	}

	consoleHost := strings.ToLower(strings.TrimSpace(a.cfg.Factory.ConsoleHost))
	if consoleHost == "" {
		consoleHost = "console.example.com"
	}

	// 1. If console host, redirect to admin console
	if host == consoleHost {
		http.Redirect(w, r, "/admin/console", http.StatusTemporaryRedirect)
		return tenant.Scope{}, tenant.Governance{}, nil, nil, false
	}

	// 2. Resolve tenant scope
	scope, err := tenant.FromContext(r.Context())
	tenantFound := err == nil && scope.TenantID != "" && scope.TenantID != "default" && scope.TenantID != "platform"
	if !tenantFound {
		if resolved, resOk := tenant.ResolveTenantByDomain(r.Context(), a.db, string(a.cfg.Database.Dialect), host); resOk {
			scope = resolved
			tenantFound = true
		}
	}

	// If not on a tenant host, show showcase for root or redirect
	if !tenantFound || scope.TenantID == "" {
		if r.URL.Path == "/" || r.URL.Path == "" {
			_ = storefront.RenderShowcase(w, r, storefront.ShowcaseData{
				BaseDomain: a.getBaseDomain(),
			})
			return tenant.Scope{}, tenant.Governance{}, nil, nil, false
		}
		http.Redirect(w, r, "/admin", http.StatusTemporaryRedirect)
		return tenant.Scope{}, tenant.Governance{}, nil, nil, false
	}

	// 3. Load tenant governance
	dialect := string(a.cfg.Database.Dialect)
	gov, err := tenant.GetGovernance(r.Context(), a.db, dialect, scope.TenantID)
	if err != nil {
		gov = tenant.DefaultGovernance(scope.TenantID)
	}

	// 4. Suspended check -> Luxury Maintenance Page
	if gov.LifecycleState == tenant.LifecycleSuspended {
		_ = storefront.RenderSuspended(w, r, storefront.SuspendedData{
			StoreName:  gov.DisplayName,
			OwnerPhone: gov.OwnerPhone,
		})
		return tenant.Scope{}, gov, nil, nil, false
	}

	// 5. Query tenant store_settings and products with in-memory caching for high-traffic scalability
	tenantCtx := tenant.WithScope(r.Context(), scope)

	type cachedStorefrontData struct {
		Settings map[string]any
		Products []storefront.ProductView
	}

	cacheKey := "storefront:ctx:" + scope.TenantID
	if a.memoryStore != nil {
		if cached, ok := a.memoryStore.Get(cacheKey); ok {
			if data, valid := cached.(cachedStorefrontData); valid {
				return scope, gov, data.Settings, data.Products, true
			}
		}
	}

	settings := a.loadTenantStoreSettings(tenantCtx, gov)
	products := a.loadTenantStoreProducts(tenantCtx)

	// If no products seeded yet, generate preview catalog
	if len(products) == 0 {
		_, defaultProds := a.fallbackCatalog(gov)
		products = defaultProds
	}

	if a.memoryStore != nil {
		a.memoryStore.Set(cacheKey, cachedStorefrontData{
			Settings: settings,
			Products: products,
		}, 30*time.Second)
	}

	return scope, gov, settings, products, true
}

func (a *serveApp) storefrontHandler(w http.ResponseWriter, r *http.Request) {
	scope, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	archetype := blueprints.NormalizeArchetype(gov.Blueprint)
	if at, ok := settings["archetype"].(string); ok && at != "" {
		archetype = blueprints.NormalizeArchetype(at)
	} else if st, ok := settings["site_type"].(string); ok && st != "" {
		archetype = blueprints.NormalizeArchetype(st)
	}

	if archetype != blueprints.ArchetypeBoutique {
		if a.renderArchetypeStorefront(w, r, gov, settings, scope, archetype) {
			return
		}
	}

	cleanPhone := gov.OwnerPhone
	if phone, ok := settings["whatsapp_number"].(string); ok && strings.TrimSpace(phone) != "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	} else if name, ok := settings["store_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	}

	slogan := ""
	if s, ok := settings["tagline"].(string); ok {
		slogan = s
	} else if s, ok := settings["slogan"].(string); ok {
		slogan = s
	}

	primaryColor := "#8C6D46"
	if c, ok := settings["primary_color"].(string); ok && strings.TrimSpace(c) != "" {
		primaryColor = strings.TrimSpace(c)
	} else if c, ok := settings["accent_color"].(string); ok && strings.TrimSpace(c) != "" {
		primaryColor = strings.TrimSpace(c)
	}

	bgTone := "#FAF8F5"
	if bg, ok := settings["bg_tone"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	} else if bg, ok := settings["bg_primary_color"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	}

	announcement := ""
	if ann, ok := settings["announcement_bar"].(string); ok {
		announcement = ann
	}

	shopierURL := ""
	if sh, ok := settings["shopier_url"].(string); ok {
		shopierURL = sh
	} else if sh, ok := settings["payment_link"].(string); ok {
		shopierURL = sh
	}

	layoutMode := "multipage"
	if lm, ok := settings["layout_mode"].(string); ok && strings.TrimSpace(lm) != "" {
		layoutMode = strings.TrimSpace(lm)
	}

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	} else if fp, ok := settings["theme_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	isDemo := gov.LifecycleState == tenant.LifecycleDemo || gov.Status == "DEMO" || gov.Status == "demo"

	// Build rich editorial storytelling, pillars, testimonials and FAQs
	hBadge, hTitle, hSub, hImg, sBadge, sTitle, sText, sQuote, sImg, pTitle, pillars, testimonials, faqs, atAddr, atHours, insta :=
		buildStorefrontEditorial(settings, storeName, gov.Notes)

	var layoutJSON string
	if lj, ok := settings["layout_json"].(string); ok {
		layoutJSON = strings.TrimSpace(lj)
	}

	data := storefront.StorefrontData{
		StoreName:       storeName,
		Slogan:          slogan,
		PrimaryColor:    primaryColor,
		BGTone:          bgTone,
		CleanPhone:      cleanPhone,
		ShopierURL:      shopierURL,
		AnnouncementBar: announcement,
		IsDemo:          isDemo,
		LayoutMode:      layoutMode,
		FontPreset:      fontPreset,
		LayoutJSON:      layoutJSON,
		HeroBadge:       hBadge,
		HeroTitle:       hTitle,
		HeroSubtitle:    hSub,
		HeroImage:       hImg,
		StoryBadge:      sBadge,
		StoryTitle:      sTitle,
		StoryText:       sText,
		StoryQuote:      sQuote,
		StoryImage:      sImg,
		PillarsTitle:    pTitle,
		Pillars:         pillars,
		Testimonials:    testimonials,
		FAQs:            faqs,
		AtelierAddress:  atAddr,
		AtelierHours:    atHours,
		Instagram:       insta,
		Products:        products,
	}

	_ = storefront.Render(w, r, data)
}

func buildStorefrontEditorial(settings map[string]any, storeName, notes string) (
	heroBadge, heroTitle, heroSubtitle, heroImage string,
	storyBadge, storyTitle, storyText, storyQuote, storyImage string,
	pillarsTitle string,
	pillars []storefront.FeaturePillar,
	testimonials []storefront.Testimonial,
	faqs []storefront.FAQItem,
	atelierAddress, atelierHours, instagram string,
) {
	baseline := blueprints.GetNicheEditorialContent(storeName, notes)
	heroBadge = baseline.HeroBadge
	heroTitle = baseline.HeroTitle
	heroSubtitle = baseline.HeroSubtitle
	heroImage = baseline.HeroImage
	storyBadge = baseline.StoryBadge
	storyTitle = baseline.StoryTitle
	storyText = baseline.StoryText
	storyQuote = baseline.StoryQuote
	storyImage = baseline.StoryImage
	pillarsTitle = baseline.PillarsTitle
	atelierAddress = baseline.AtelierAddress
	atelierHours = baseline.AtelierHours
	instagram = baseline.Instagram

	for _, p := range baseline.Pillars {
		pillars = append(pillars, storefront.FeaturePillar{Title: p.Title, Desc: p.Desc})
	}
	for _, t := range baseline.Testimonials {
		testimonials = append(testimonials, storefront.Testimonial{Customer: t.Customer, Location: t.Location, Quote: t.Quote, Item: t.Item})
	}
	for _, f := range baseline.FAQs {
		faqs = append(faqs, storefront.FAQItem{Question: f.Question, Answer: f.Answer})
	}

	if v, ok := settings["hero_badge"].(string); ok && strings.TrimSpace(v) != "" {
		heroBadge = v
	}
	if v, ok := settings["hero_title"].(string); ok && strings.TrimSpace(v) != "" {
		heroTitle = v
	}
	if v, ok := settings["hero_subtitle"].(string); ok && strings.TrimSpace(v) != "" {
		heroSubtitle = v
	}
	if v, ok := settings["hero_image"].(string); ok && strings.TrimSpace(v) != "" {
		heroImage = v
	}
	if v, ok := settings["story_badge"].(string); ok && strings.TrimSpace(v) != "" {
		storyBadge = v
	}
	if v, ok := settings["story_title"].(string); ok && strings.TrimSpace(v) != "" {
		storyTitle = v
	}
	if v, ok := settings["story_text"].(string); ok && strings.TrimSpace(v) != "" {
		storyText = v
	}
	if v, ok := settings["story_quote"].(string); ok && strings.TrimSpace(v) != "" {
		storyQuote = v
	}
	if v, ok := settings["story_image"].(string); ok && strings.TrimSpace(v) != "" {
		storyImage = v
	}
	if v, ok := settings["pillars_title"].(string); ok && strings.TrimSpace(v) != "" {
		pillarsTitle = v
	}
	if v, ok := settings["atelier_address"].(string); ok && strings.TrimSpace(v) != "" {
		atelierAddress = v
	}
	if v, ok := settings["atelier_hours"].(string); ok && strings.TrimSpace(v) != "" {
		atelierHours = v
	}
	if v, ok := settings["instagram"].(string); ok && strings.TrimSpace(v) != "" {
		instagram = v
	}

	if pStr, ok := settings["pillars_json"].(string); ok && strings.TrimSpace(pStr) != "" {
		var parsed []storefront.FeaturePillar
		if err := json.Unmarshal([]byte(pStr), &parsed); err == nil && len(parsed) > 0 {
			pillars = parsed
		}
	}
	if tStr, ok := settings["testimonials_json"].(string); ok && strings.TrimSpace(tStr) != "" {
		var parsed []storefront.Testimonial
		if err := json.Unmarshal([]byte(tStr), &parsed); err == nil && len(parsed) > 0 {
			testimonials = parsed
		}
	}
	if fStr, ok := settings["faqs_json"].(string); ok && strings.TrimSpace(fStr) != "" {
		var parsed []storefront.FAQItem
		if err := json.Unmarshal([]byte(fStr), &parsed); err == nil && len(parsed) > 0 {
			faqs = parsed
		}
	}

	return
}

func (a *serveApp) loadTenantStoreSettings(ctx context.Context, gov tenant.Governance) map[string]any {
	primary := "#8C6D46"
	bg := "#FAF8F5"
	switch gov.Blueprint {
	case blueprints.ArchetypeSaaS:
		primary = "#2563EB"
		bg = "#09090B"
	case blueprints.ArchetypeRestaurant:
		primary = "#B45309"
		bg = "#18181B"
	case blueprints.ArchetypeCorporate:
		primary = "#1E3A8A"
		bg = "#F8FAFC"
	case blueprints.ArchetypePortfolio:
		primary = "#0F172A"
		bg = "#F8FAFC"
	}

	out := map[string]any{
		"brand_name":       gov.DisplayName,
		"whatsapp_number":  gov.OwnerPhone,
		"primary_color":    primary,
		"bg_tone":          bg,
		"announcement_bar": "1500 TL ve Üzeri Siparişlerde Ücretsiz Kargo · WhatsApp ile Hızlı Sipariş",
	}

	// Look up site_settings first (for corporate/restaurant/saas/portfolio), then store_settings (boutique)
	recs, err := a.collectionService.ListRecords(ctx, "site_settings", 1, 0)
	if err != nil || len(recs) == 0 {
		recs, err = a.collectionService.ListRecords(ctx, "store_settings", 1, 0)
	}
	if err == nil && len(recs) > 0 {
		for k, v := range recs[0].Data {
			out[k] = v
		}
	}
	return out
}

func (a *serveApp) loadTenantStoreProducts(ctx context.Context) []storefront.ProductView {
	recs, err := a.collectionService.ListRecords(ctx, "products", 50, 0)
	if err != nil || len(recs) == 0 {
		recs, err = a.collectionService.ListRecords(ctx, "boutique_products", 50, 0)
	}
	if err != nil || len(recs) == 0 {
		return nil
	}

	var prods []storefront.ProductView
	for _, r := range recs {
		pv := mapRecordToProductView(r)
		if pv.Title != "" {
			prods = append(prods, pv)
		}
	}
	return prods
}

func parseOrSynthesizeVariants(d map[string]any, basePrice float64, sizesStr, colorsStr, defaultImg string) ([]storefront.OptionDef, []storefront.VariantDef, string) {
	var options []storefront.OptionDef
	var variants []storefront.VariantDef
	secImg := ""

	if s, ok := d["secondary_image"].(string); ok && s != "" {
		secImg = s
	}

	// 1. Try options_json / options
	if optRaw, ok := d["options_json"].(string); ok && strings.TrimSpace(optRaw) != "" {
		_ = json.Unmarshal([]byte(optRaw), &options)
	} else if optList, ok := d["options"].([]any); ok {
		if b, err := json.Marshal(optList); err == nil {
			_ = json.Unmarshal(b, &options)
		}
	}

	// 2. Try variants_json / variants
	if varRaw, ok := d["variants_json"].(string); ok && strings.TrimSpace(varRaw) != "" {
		_ = json.Unmarshal([]byte(varRaw), &variants)
	} else if varList, ok := d["variants"].([]any); ok {
		if b, err := json.Marshal(varList); err == nil {
			_ = json.Unmarshal(b, &variants)
		}
	}

	// 3. If neither parsed, synthesize options and variants from colors & sizes
	if len(options) == 0 && len(variants) == 0 {
		var colors, sizes []string
		if strings.TrimSpace(colorsStr) != "" {
			for _, c := range strings.Split(colorsStr, ",") {
				if trimmed := strings.TrimSpace(c); trimmed != "" {
					colors = append(colors, trimmed)
				}
			}
		}
		if strings.TrimSpace(sizesStr) != "" {
			for _, s := range strings.Split(sizesStr, ",") {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					sizes = append(sizes, trimmed)
				}
			}
		}

		if len(colors) == 0 {
			colors = []string{"Taba", "Siyah"}
		}
		if len(sizes) == 0 {
			sizes = []string{"Standart", "Büyük"}
		}

		options = append(options, storefront.OptionDef{Name: "Renk", Values: colors})
		options = append(options, storefront.OptionDef{Name: "Beden", Values: sizes})

		for i, c := range colors {
			for j, s := range sizes {
				skuPrefix := "TB"
				if len(c) >= 2 {
					skuPrefix = strings.ToUpper(c[:2])
				}
				skuSuffix := "STD"
				if len(s) >= 2 {
					skuSuffix = strings.ToUpper(s[:2])
				}
				sku := fmt.Sprintf("%s-%s", skuPrefix, skuSuffix)
				title := fmt.Sprintf("%s / %s", c, s)
				priceDiff := 0.0
				if j > 0 {
					priceDiff = 150.0 // Slight premium on secondary sizing
				}
				stock := 5
				// Demo out-of-stock on the last variant combination
				if i == len(colors)-1 && j == len(sizes)-1 {
					stock = 0
				}
				variants = append(variants, storefront.VariantDef{
					SKU:       sku,
					Title:     title,
					PriceDiff: priceDiff,
					Stock:     stock,
					Options:   []string{c, s},
				})
			}
		}
	}

	if secImg == "" && defaultImg != "" {
		secImg = defaultImg
	}

	return options, variants, secImg
}

func mapRecordToProductView(r collections.Record) storefront.ProductView {
	pv := storefront.ProductView{
		ID:      r.ID,
		InStock: true,
	}
	d := r.Data
	if t, ok := d["title"].(string); ok {
		pv.Title = t
	}
	if s, ok := d["slug"].(string); ok {
		pv.Slug = s
	}
	if p, ok := d["price"].(float64); ok {
		pv.Price = p
	}
	if cp, ok := d["compare_at_price"].(float64); ok && cp > 0 {
		pv.CompareAtPrice = cp
	}
	if desc, ok := d["description"].(string); ok {
		pv.Description = desc
	}
	if img, ok := d["image_url"].(string); ok {
		pv.ImageURL = img
	}
	if s, ok := d["sizes"].(string); ok {
		pv.Sizes = s
	}
	if c, ok := d["colors"].(string); ok {
		pv.Colors = c
	}
	if b, ok := d["badge"].(string); ok {
		pv.Badge = b
	}
	if pl, ok := d["payment_link"].(string); ok {
		pv.PaymentLink = pl
	}
	if is, ok := d["in_stock"].(bool); ok {
		pv.InStock = is
	}

	opts, vars, sec := parseOrSynthesizeVariants(d, pv.Price, pv.Sizes, pv.Colors, pv.ImageURL)
	pv.Options = opts
	pv.Variants = vars
	pv.SecondaryImage = sec

	if b, err := json.Marshal(pv.Options); err == nil {
		pv.OptionsJSON = template.JS(b)
	}
	if b, err := json.Marshal(pv.Variants); err == nil {
		pv.VariantsJSON = template.JS(b)
	}

	return pv
}

func (a *serveApp) fallbackCatalog(gov tenant.Governance) (map[string]any, []storefront.ProductView) {
	settings, rawProds := a.defaultCatalogGen(gov.DisplayName, gov.OwnerPhone, gov.Notes)
	var views []storefront.ProductView
	for i, p := range rawProds {
		title, _ := p["title"].(string)
		slug, _ := p["slug"].(string)
		price, _ := p["price"].(float64)
		desc, _ := p["description"].(string)
		img, _ := p["image_url"].(string)
		badge, _ := p["badge"].(string)
		sizes, _ := p["sizes"].(string)
		colors, _ := p["colors"].(string)

		pv := storefront.ProductView{
			ID:          fmt.Sprintf("mock-%d", i+1),
			Title:       title,
			Slug:        slug,
			Price:       price,
			Description: desc,
			ImageURL:    img,
			Badge:       badge,
			Sizes:       sizes,
			Colors:      colors,
			InStock:     true,
		}

		opts, vars, sec := parseOrSynthesizeVariants(p, price, sizes, colors, img)
		pv.Options = opts
		pv.Variants = vars
		pv.SecondaryImage = sec

		if b, err := json.Marshal(pv.Options); err == nil {
			pv.OptionsJSON = template.JS(b)
		}
		if b, err := json.Marshal(pv.Variants); err == nil {
			pv.VariantsJSON = template.JS(b)
		}

		views = append(views, pv)
	}
	return settings, views
}

func (a *serveApp) defaultCatalogGen(name, phone, prompt string) (map[string]any, []map[string]any) {
	return blueprints.GenerateLuxuryCatalog(name, phone, prompt)
}

func (a *serveApp) storefrontOrderHandler(w http.ResponseWriter, r *http.Request) {
	type cartItemIn struct {
		ID    string  `json:"id"`
		Title string  `json:"title"`
		Price float64 `json:"price"`
		Qty   int     `json:"qty"`
		Size  string  `json:"size"`
		Color string  `json:"color"`
	}

	var in struct {
		ProductID       string       `json:"product_id"`
		ProductTitle    string       `json:"product_title"`
		Price           float64      `json:"price"`
		Size            string       `json:"size"`
		Color           string       `json:"color"`
		SKU             string       `json:"sku"`
		VariantTitle    string       `json:"variant_title"`
		Items           []cartItemIn `json:"items"`
		Discount        float64      `json:"discount"`
		PromoCode       string       `json:"promo_code"`
		CustomerName    string       `json:"customer_name"`
		CustomerPhone   string       `json:"customer_phone"`
		CustomerAddress string       `json:"customer_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid json payload", nil)
		return
	}

	// 1. Resolve tenant scope
	scope, err := tenant.FromContext(r.Context())
	tenantFound := err == nil && scope.TenantID != ""
	if !tenantFound {
		host := strings.ToLower(strings.TrimSpace(r.Host))
		if i := strings.IndexByte(host, ':'); i > 0 {
			host = host[:i]
		}
		if res, resOk := tenant.ResolveTenantByDomain(r.Context(), a.db, string(a.cfg.Database.Dialect), host); resOk {
			scope = res
			tenantFound = true
		}
	}

	var codeBytes [2]byte
	_, _ = rand.Read(codeBytes[:])
	codeNum := 1000 + (int(codeBytes[0])<<8|int(codeBytes[1]))%9000
	orderCode := fmt.Sprintf("#HYD-%04d", codeNum)

	var totalPrice float64 = in.Price
	var itemsSummary string

	if len(in.Items) > 0 {
		var sb strings.Builder
		totalPrice = 0
		for i, item := range in.Items {
			qty := item.Qty
			if qty <= 0 {
				qty = 1
			}
			totalPrice += item.Price * float64(qty)
			if i > 0 {
				sb.WriteString(", ")
			}
			detail := item.Title
			if item.Size != "" && item.Size != "Standart" {
				detail += " (" + item.Size + ")"
			}
			if item.Color != "" {
				detail += " (" + item.Color + ")"
			}
			sb.WriteString(fmt.Sprintf("%dx %s", qty, detail))
		}
		if in.Discount > 0 {
			totalPrice -= in.Discount
			if totalPrice < 0 {
				totalPrice = 0
			}
			sb.WriteString(fmt.Sprintf(" [Kupon: %s -%.2f TL]", in.PromoCode, in.Discount))
		}
		itemsSummary = sb.String()
	} else {
		variantSummary := in.Size
		if in.Color != "" {
			if variantSummary != "" {
				variantSummary = in.Color + " / " + variantSummary
			} else {
				variantSummary = in.Color
			}
		}
		if in.VariantTitle != "" {
			variantSummary = in.VariantTitle
		}
		if variantSummary != "" {
			itemsSummary = fmt.Sprintf("%s (%s)", in.ProductTitle, variantSummary)
		} else {
			itemsSummary = in.ProductTitle
		}
	}

	if tenantFound && scope.TenantID != "" {
		tenantCtx := tenant.WithScope(r.Context(), scope)

		orderData := map[string]any{
			"order_code":       orderCode,
			"order_no":         orderCode,
			"customer_name":    in.CustomerName,
			"customer_phone":   in.CustomerPhone,
			"customer_address": in.CustomerAddress,
			"shipping_address": in.CustomerAddress,
			"items_json":       itemsSummary,
			"items_summary":    itemsSummary,
			"total_price":      totalPrice,
			"total_amount":     totalPrice,
			"status":           "yeni",
			"order_channel":    "WhatsApp",
		}

		// Try orders collection, then boutique_orders
		created, err := a.collectionService.CreateRecord(tenantCtx, "orders", orderData)
		if err != nil {
			fmt.Printf("[storefrontOrderHandler] CreateRecord orders failed: %v\n", err)
			created, err = a.collectionService.CreateRecord(tenantCtx, "boutique_orders", orderData)
			if err != nil {
				fmt.Printf("[storefrontOrderHandler] CreateRecord boutique_orders failed: %v\n", err)
			}
		}
		if codeVal, ok := created.Data["order_code"].(string); ok && codeVal != "" {
			orderCode = codeVal
		}
	}

	api.WriteData(w, http.StatusOK, map[string]any{
		"order_code": orderCode,
		"status":     "PENDING_WHATSAPP",
	}, nil)
}
