package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/storefront"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// Dedicated Product Detail Page (GET /urun/{slug} & GET /products/{slug})
func (a *serveApp) storefrontProductHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	var found *storefront.ProductView
	var related []storefront.ProductView

	for i := range products {
		p := &products[i]
		if strings.EqualFold(p.Slug, slug) || strings.EqualFold(p.ID, slug) {
			found = p
		} else {
			related = append(related, *p)
		}
	}

	if found == nil {
		if len(products) > 0 {
			found = &products[0]
		} else {
			http.NotFound(w, r)
			return
		}
	}

	if len(related) > 4 {
		related = related[:4]
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
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
	}

	bgTone := "#FAF8F5"
	if bg, ok := settings["bg_tone"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	}

	cleanPhone := gov.OwnerPhone
	if phone, ok := settings["whatsapp_number"].(string); ok && strings.TrimSpace(phone) != "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	shopierURL := ""
	if sh, ok := settings["shopier_url"].(string); ok {
		shopierURL = sh
	}

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	prodsBytes, _ := json.Marshal(products)
	data := storefront.ProductDetailData{
		StoreName:       storeName,
		Slogan:          slogan,
		PrimaryColor:    primaryColor,
		BGTone:          bgTone,
		CleanPhone:      cleanPhone,
		ShopierURL:      shopierURL,
		FontPreset:      fontPreset,
		Product:         *found,
		RelatedProducts: related,
		ProductsJSON:    template.JS(prodsBytes),
	}

	_ = storefront.RenderProductDetail(w, r, data)
}

// Category Listing Page (GET /kategori/{slug} & GET /collections/{slug})
func (a *serveApp) storefrontCategoryHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		slug = "tumu"
	}

	// Collect unique categories
	categoryMap := make(map[string]bool)
	var categoryList []string
	for _, p := range products {
		cat := strings.TrimSpace(p.Category)
		if cat != "" && !categoryMap[cat] {
			categoryMap[cat] = true
			categoryList = append(categoryList, cat)
		}
	}
	if len(categoryList) == 0 {
		categoryList = []string{"Atölye Seçkisi", "Özel Tasarım", "Masaüstü & Sofra"}
	}

	var filtered []storefront.ProductView
	if slug == "tumu" || slug == "all" || slug == "koleksiyon" {
		filtered = products
	} else {
		for _, p := range products {
			if strings.EqualFold(p.Category, slug) || strings.EqualFold(generateCleanSlug(p.Category), slug) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			filtered = products
		}
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	}

	slogan := ""
	if s, ok := settings["tagline"].(string); ok {
		slogan = s
	}

	primaryColor := "#8C6D46"
	if c, ok := settings["primary_color"].(string); ok && strings.TrimSpace(c) != "" {
		primaryColor = strings.TrimSpace(c)
	}

	bgTone := "#FAF8F5"
	if bg, ok := settings["bg_tone"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	}

	cleanPhone := gov.OwnerPhone
	if phone, ok := settings["whatsapp_number"].(string); ok && strings.TrimSpace(phone) != "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	prodsBytes, _ := json.Marshal(products)
	data := storefront.CategoryData{
		StoreName:       storeName,
		Slogan:          slogan,
		PrimaryColor:    primaryColor,
		BGTone:          bgTone,
		CleanPhone:      cleanPhone,
		FontPreset:      fontPreset,
		CurrentCategory: slug,
		Categories:      categoryList,
		Products:        filtered,
		TotalCount:      len(products),
		ProductsJSON:    template.JS(prodsBytes),
	}

	_ = storefront.RenderCategory(w, r, data)
}

// Dedicated About & Atelier Story Page (GET /hakkimizda & GET /about)
func (a *serveApp) storefrontAboutHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	}

	slogan := ""
	if s, ok := settings["tagline"].(string); ok {
		slogan = s
	}

	primaryColor := "#8C6D46"
	if c, ok := settings["primary_color"].(string); ok && strings.TrimSpace(c) != "" {
		primaryColor = strings.TrimSpace(c)
	}

	bgTone := "#FAF8F5"
	if bg, ok := settings["bg_tone"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	}

	cleanPhone := gov.OwnerPhone
	if phone, ok := settings["whatsapp_number"].(string); ok && strings.TrimSpace(phone) != "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	_, _, _, _, sBadge, sTitle, sText, sQuote, sImg, pTitle, pillars, _, _, _, _, _ :=
		buildStorefrontEditorial(settings, storeName, gov.Notes)

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	prodsBytes, _ := json.Marshal(products)
	data := storefront.AboutData{
		StoreName:    storeName,
		Slogan:       slogan,
		PrimaryColor: primaryColor,
		BGTone:       bgTone,
		CleanPhone:   cleanPhone,
		FontPreset:   fontPreset,
		StoryBadge:   sBadge,
		StoryTitle:   sTitle,
		StoryText:    sText,
		StoryQuote:   sQuote,
		StoryImage:   sImg,
		PillarsTitle: pTitle,
		Pillars:      pillars,
		ProductsJSON: template.JS(prodsBytes),
	}

	_ = storefront.RenderAbout(w, r, data)
}

// Dedicated Contact & Showroom Page (GET /iletisim & GET /contact)
func (a *serveApp) storefrontContactHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	}

	slogan := ""
	if s, ok := settings["tagline"].(string); ok {
		slogan = s
	}

	primaryColor := "#8C6D46"
	if c, ok := settings["primary_color"].(string); ok && strings.TrimSpace(c) != "" {
		primaryColor = strings.TrimSpace(c)
	}

	bgTone := "#FAF8F5"
	if bg, ok := settings["bg_tone"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	}

	cleanPhone := gov.OwnerPhone
	if phone, ok := settings["whatsapp_number"].(string); ok && strings.TrimSpace(phone) != "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	_, _, _, _, _, _, _, _, _, _, _, _, faqs, atAddr, atHours, _ :=
		buildStorefrontEditorial(settings, storeName, gov.Notes)

	if atAddr == "" {
		atAddr = fmt.Sprintf("%s Tasarım & Zanaat Atölyesi, Çukurcuma Cad. No: 14/A, Beyoğlu / İstanbul", storeName)
	}
	if atHours == "" {
		atHours = "Salı – Cumartesi: 10:30 – 19:00 (Pazar & Pazartesi randevu ile)"
	}

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	prodsBytes, _ := json.Marshal(products)
	data := storefront.ContactData{
		StoreName:      storeName,
		Slogan:         slogan,
		PrimaryColor:   primaryColor,
		BGTone:         bgTone,
		CleanPhone:     cleanPhone,
		FontPreset:     fontPreset,
		AtelierAddress: atAddr,
		AtelierHours:   atHours,
		FAQs:           faqs,
		ProductsJSON:   template.JS(prodsBytes),
	}

	_ = storefront.RenderContact(w, r, data)
}

func (a *serveApp) storefrontOrderTrackingHandler(w http.ResponseWriter, r *http.Request) {
	scope, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	}

	slogan := "Özgün Zanaat & Lüks Butik"
	if s, ok := settings["tagline"].(string); ok {
		slogan = s
	} else if s, ok := settings["slogan"].(string); ok {
		slogan = s
	}

	primaryColor := "#8C6D46"
	if c, ok := settings["primary_color"].(string); ok && strings.TrimSpace(c) != "" {
		primaryColor = strings.TrimSpace(c)
	}

	bgTone := "#FAF8F5"
	if bg, ok := settings["bg_tone"].(string); ok && strings.TrimSpace(bg) != "" {
		bgTone = strings.TrimSpace(bg)
	}

	cleanPhone := gov.OwnerPhone
	if phone, ok := settings["whatsapp_number"].(string); ok && strings.TrimSpace(phone) != "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	prodsBytes, _ := json.Marshal(products)

	queryCode := strings.TrimSpace(r.URL.Query().Get("code"))
	if queryCode == "" {
		queryCode = strings.TrimSpace(r.URL.Query().Get("q"))
	}

	trackingData := storefront.OrderTrackingData{
		StoreName:    storeName,
		Slogan:       slogan,
		PrimaryColor: primaryColor,
		BGTone:       bgTone,
		CleanPhone:   cleanPhone,
		FontPreset:   fontPreset,
		QueryCode:    queryCode,
		ProductsJSON: template.JS(prodsBytes),
	}

	if queryCode != "" {
		cleanQ := strings.ToUpper(strings.TrimPrefix(queryCode, "#"))
		cleanPhoneQ := strings.ReplaceAll(queryCode, " ", "")

		tenantCtx := tenant.WithScope(r.Context(), scope)
		recs, err := a.collectionService.ListRecords(tenantCtx, "orders", 100, 0)
		if err != nil || len(recs) == 0 {
			recs, _ = a.collectionService.ListRecords(tenantCtx, "boutique_orders", 100, 0)
		}

		for _, r := range recs {
			d := r.Data
			code, _ := d["order_code"].(string)
			if code == "" {
				code, _ = d["order_no"].(string)
			}
			cleanCode := strings.ToUpper(strings.TrimPrefix(code, "#"))
			phone, _ := d["customer_phone"].(string)
			cleanRecPhone := strings.ReplaceAll(phone, " ", "")

			if strings.EqualFold(cleanCode, cleanQ) || strings.EqualFold(code, cleanQ) || (cleanPhoneQ != "" && strings.Contains(cleanRecPhone, cleanPhoneQ)) {
				trackingData.OrderFound = true
				trackingData.OrderCode = code
				if trackingData.OrderCode == "" {
					trackingData.OrderCode = r.ID
				}
				if st, ok := d["order_status"].(string); ok && st != "" {
					trackingData.OrderStatus = st
				} else if st, ok := d["status"].(string); ok && st != "" {
					trackingData.OrderStatus = st
				} else {
					trackingData.OrderStatus = "yeni"
				}
				trackingData.CustomerName, _ = d["customer_name"].(string)
				trackingData.CustomerPhone, _ = d["customer_phone"].(string)
				trackingData.CustomerAddress, _ = d["customer_address"].(string)
				if trackingData.CustomerAddress == "" {
					trackingData.CustomerAddress, _ = d["shipping_address"].(string)
				}
				trackingData.ItemsSummary, _ = d["items_summary"].(string)
				if trackingData.ItemsSummary == "" {
					trackingData.ItemsSummary, _ = d["items_json"].(string)
				}
				if tot, ok := d["total_amount"].(float64); ok {
					trackingData.TotalAmount = tot
				} else if tot, ok := d["total_price"].(float64); ok {
					trackingData.TotalAmount = tot
				}
				trackingData.TrackingCode, _ = d["tracking_code"].(string)
				trackingData.Carrier, _ = d["carrier"].(string)
				trackingData.OrderDate = r.CreatedAt.Format("02.01.2006 15:04")
				break
			}
		}
	}

	_ = storefront.RenderOrderTracking(w, r, trackingData)
}

func (a *serveApp) storefrontTrackOrderAPIHandler(w http.ResponseWriter, r *http.Request) {
	scope, _, _, _, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "storefront not found", nil)
		return
	}

	var in struct {
		Code  string `json:"code"`
		Phone string `json:"phone"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)

	query := strings.TrimSpace(in.Code)
	if query == "" {
		query = strings.TrimSpace(in.Phone)
	}
	if query == "" {
		api.WriteError(w, r, http.StatusBadRequest, "code or phone is required", nil)
		return
	}

	cleanQ := strings.ToUpper(strings.TrimPrefix(query, "#"))
	cleanPhoneQ := strings.ReplaceAll(query, " ", "")

	tenantCtx := tenant.WithScope(r.Context(), scope)
	recs, err := a.collectionService.ListRecords(tenantCtx, "orders", 100, 0)
	if err != nil || len(recs) == 0 {
		recs, _ = a.collectionService.ListRecords(tenantCtx, "boutique_orders", 100, 0)
	}

	for _, rec := range recs {
		d := rec.Data
		code, _ := d["order_code"].(string)
		if code == "" {
			code, _ = d["order_no"].(string)
		}
		cleanCode := strings.ToUpper(strings.TrimPrefix(code, "#"))
		phone, _ := d["customer_phone"].(string)
		cleanRecPhone := strings.ReplaceAll(phone, " ", "")

		if strings.EqualFold(cleanCode, cleanQ) || strings.EqualFold(code, cleanQ) || (cleanPhoneQ != "" && strings.Contains(cleanRecPhone, cleanPhoneQ)) {
			status, _ := d["order_status"].(string)
			if status == "" {
				status, _ = d["status"].(string)
			}
			trackingCode, _ := d["tracking_code"].(string)
			itemsSummary, _ := d["items_summary"].(string)
			tot, _ := d["total_amount"].(float64)

			api.WriteData(w, http.StatusOK, map[string]any{
				"found":         true,
				"order_code":    code,
				"status":        status,
				"customer_name": d["customer_name"],
				"items_summary": itemsSummary,
				"total_amount":  tot,
				"tracking_code": trackingCode,
				"created_at":    rec.CreatedAt.Format(time.RFC3339),
			}, nil)
			return
		}
	}

	api.WriteData(w, http.StatusOK, map[string]any{
		"found":   false,
		"message": "Sipariş bulunamadı",
	}, nil)
}

func (a *serveApp) storefrontSitemapHandler(w http.ResponseWriter, r *http.Request) {
	_, _, _, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	scheme := "https"
	if r.TLS == nil && (strings.HasPrefix(r.Host, "127.0.0.1") || strings.HasPrefix(r.Host, "localhost")) {
		scheme = "http"
	}
	base := fmt.Sprintf("%s://%s", scheme, r.Host)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	staticPaths := []string{"", "kategori/tumu", "hakkimizda", "iletisim", "siparis-takip"}
	for _, p := range staticPaths {
		loc := base + "/" + p
		if p == "" {
			loc = base + "/"
		}
		sb.WriteString(fmt.Sprintf("  <url>\n    <loc>%s</loc>\n    <changefreq>daily</changefreq>\n    <priority>%.1f</priority>\n  </url>\n",
			loc, func() float64 {
				if p == "" {
					return 1.0
				}
				return 0.8
			}()))
	}

	for _, prod := range products {
		if prod.Slug != "" {
			loc := fmt.Sprintf("%s/urun/%s", base, prod.Slug)
			sb.WriteString(fmt.Sprintf("  <url>\n    <loc>%s</loc>\n    <changefreq>weekly</changefreq>\n    <priority>0.9</priority>\n  </url>\n", loc))
		}
	}

	sb.WriteString(`</urlset>`)
	_, _ = w.Write([]byte(sb.String()))
}

func (a *serveApp) storefrontRobotsHandler(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if r.TLS == nil && (strings.HasPrefix(r.Host, "127.0.0.1") || strings.HasPrefix(r.Host, "localhost")) {
		scheme = "http"
	}
	base := fmt.Sprintf("%s://%s", scheme, r.Host)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", base)
}

func (a *serveApp) storefrontLLMsHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	name := gov.DisplayName
	if n, ok := settings["brand_name"].(string); ok && n != "" {
		name = n
	}
	tagline := gov.Notes
	if t, ok := settings["tagline"].(string); ok && t != "" {
		tagline = t
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n> %s\n\n## Hakkında\nBu web sitesi HydroGo Cloud AI üzerinde çalışan modern bir işletmedir.\n\n## Ürünler ve Hizmetler\n", name, tagline))
	for _, p := range products {
		if p.Title != "" {
			sb.WriteString(fmt.Sprintf("- **%s**: %.2f TL\n", p.Title, p.Price))
		}
	}
	sb.WriteString(fmt.Sprintf("\n## İletişim\nTelefon: %s\n", gov.OwnerPhone))
	_, _ = w.Write([]byte(sb.String()))
}
