package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/storefront"
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

	fontPreset := "editorial_playfair"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

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

	fontPreset := "editorial_playfair"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

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
	}

	_ = storefront.RenderCategory(w, r, data)
}

// Dedicated About & Atelier Story Page (GET /hakkimizda & GET /about)
func (a *serveApp) storefrontAboutHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, _, ok := a.resolveStorefrontContext(w, r)
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

	fontPreset := "editorial_playfair"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

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
	}

	_ = storefront.RenderAbout(w, r, data)
}

// Dedicated Contact & Showroom Page (GET /iletisim & GET /contact)
func (a *serveApp) storefrontContactHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, _, ok := a.resolveStorefrontContext(w, r)
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

	fontPreset := "editorial_playfair"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

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
	}

	_ = storefront.RenderContact(w, r, data)
}
