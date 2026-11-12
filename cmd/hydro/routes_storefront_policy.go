package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/hydrohq/hydrogo/internal/blueprints"
	"github.com/hydrohq/hydrogo/internal/storefront"
)

// storefrontPolicyHandler serves dedicated compliance, privacy, cookies, terms and return policies.
func (a *serveApp) storefrontPolicyHandler(w http.ResponseWriter, r *http.Request) {
	_, gov, settings, products, ok := a.resolveStorefrontContext(w, r)
	if !ok {
		return
	}

	archetype := blueprints.NormalizeArchetype(gov.Blueprint)
	if at, ok := settings["archetype"].(string); ok && at != "" {
		archetype = blueprints.NormalizeArchetype(at)
	} else if st, ok := settings["site_type"].(string); ok && st != "" {
		archetype = blueprints.NormalizeArchetype(st)
	}

	// For news archetype, route to news portal layout
	if archetype == blueprints.ArchetypeNews {
		a.storefrontHandler(w, r)
		return
	}

	p := strings.ToLower(strings.Trim(r.URL.Path, "/"))

	policyType := "kvkk"
	policyTitle := "6698 Sayılı KVKK Aydınlatma Metni"
	policySubtitle := "Kişisel Verilerin Korunması ve İşlenmesi Politikası"

	switch {
	case strings.Contains(p, "gizlilik") || strings.Contains(p, "privacy"):
		policyType = "gizlilik"
		policyTitle = "Gizlilik ve Güvenlik Politikası"
		policySubtitle = "256-bit SSL Güvencesi ve Veri Gizliliği Standartları"
	case strings.Contains(p, "cerez") || strings.Contains(p, "cookies"):
		policyType = "cerez"
		policyTitle = "Çerez (Cookie) Aydınlatma Metni"
		policySubtitle = "Kullanıcı Deneyimi ve Çerez Yönetim İlkeleri"
	case strings.Contains(p, "mesafe") || strings.Contains(p, "satis") || strings.Contains(p, "terms"):
		policyType = "mesafe"
		policyTitle = "Mesafeli Satış Sözleşmesi"
		policySubtitle = "6502 Sayılı Tüketicinin Korunması Kanunu Kapsamında Ön Bilgilendirme"
	case strings.Contains(p, "teslimat") || strings.Contains(p, "iade") || strings.Contains(p, "returns"):
		policyType = "iade"
		policyTitle = "Teslimat, Cayma Hakkı ve İade Şartları"
		policySubtitle = "14 Gün Koşulsuz İade ve Beden Değişim Güvencesi"
	case strings.Contains(p, "kunye") || strings.Contains(p, "imprint"):
		policyType = "kunye"
		policyTitle = "Şirket Künyesi ve İletişim Bilgileri"
		policySubtitle = "Resmi Ticari Sicil ve Müşteri Destek Kanalları"
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

	fontPreset := "v0_geist"
	if fp, ok := settings["font_preset"].(string); ok && strings.TrimSpace(fp) != "" {
		fontPreset = strings.TrimSpace(fp)
	}

	contactEmail := fmt.Sprintf("destek@%s.%s", gov.TenantID, a.getBaseDomain())
	if email, ok := settings["contact_email"].(string); ok && strings.TrimSpace(email) != "" {
		contactEmail = strings.TrimSpace(email)
	}

	shopierURL := ""
	if s, ok := settings["shopier_url"].(string); ok && strings.TrimSpace(s) != "" {
		shopierURL = strings.TrimSpace(s)
	}

	prodsBytes, _ := json.Marshal(products)
	data := storefront.PolicyData{
		StoreName:      storeName,
		Slogan:         slogan,
		PrimaryColor:   primaryColor,
		BGTone:         bgTone,
		CleanPhone:     cleanPhone,
		ShopierURL:     shopierURL,
		ContactEmail:   contactEmail,
		FontPreset:     fontPreset,
		PolicyType:     policyType,
		PolicyTitle:    policyTitle,
		PolicySubtitle: policySubtitle,
		LastUpdated:    "08 Kasım 2026",
		ProductsJSON:   template.JS(prodsBytes),
	}

	_ = storefront.RenderPolicy(w, r, data)
}
