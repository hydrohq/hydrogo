package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/storefront"
)

// storefrontAccountHandler serves dedicated customer membership dashboard, orders and profile.
func (a *serveApp) storefrontAccountHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	storeName := gov.DisplayName
	if name, ok := settings["brand_name"].(string); ok && strings.TrimSpace(name) != "" {
		storeName = strings.TrimSpace(name)
	}

	slogan := "Akdeniz zanaatı ve zamansız tasarım parçaları."
	if s, ok := settings["tagline"].(string); ok && strings.TrimSpace(s) != "" {
		slogan = strings.TrimSpace(s)
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
	if s, ok := settings["shopier_url"].(string); ok && strings.TrimSpace(s) != "" {
		shopierURL = strings.TrimSpace(s)
	}

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	contactEmail := fmt.Sprintf("destek@%s.%s", gov.TenantID, a.getBaseDomain())
	if email, ok := settings["contact_email"].(string); ok && strings.TrimSpace(email) != "" {
		contactEmail = strings.TrimSpace(email)
	}

	prodsBytes, _ := json.Marshal(products)
	data := storefront.AccountData{
		StoreName:    storeName,
		Slogan:       slogan,
		PrimaryColor: primaryColor,
		BGTone:       bgTone,
		CleanPhone:   cleanPhone,
		ShopierURL:   shopierURL,
		ContactEmail: contactEmail,
		FontPreset:   fontPreset,
		ProductsJSON: template.JS(prodsBytes),
	}

	_ = storefront.RenderAccount(w, r, data)
}
