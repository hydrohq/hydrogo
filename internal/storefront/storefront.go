package storefront

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

//go:embed storefront.html suspended.html product_detail.html category_view.html about_view.html contact_view.html fonts/*
var templateFS embed.FS

// ResolveFontFamilies returns CSS font families (heading, body) for a given typography preset.
func ResolveFontFamilies(preset string) (heading, body string) {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "playfair", "editorial_playfair", "playfair_jakarta":
		return "'Playfair Display', Didot, 'Bodoni MT', 'Times New Roman', serif", "'Plus Jakarta Sans', system-ui, -apple-system, sans-serif"
	case "cinzel", "monumental_cinzel", "cinzel_inter":
		return "'Cinzel', 'Trajan Pro', 'Bodoni MT', serif", "'Inter', system-ui, -apple-system, sans-serif"
	case "syne", "avantgarde_syne", "syne_inter":
		return "'Syne', 'Futura', system-ui, sans-serif", "'Inter', system-ui, -apple-system, sans-serif"
	case "modern", "modern_grotesk", "jakarta_inter", "sans":
		return "'Plus Jakarta Sans', system-ui, -apple-system, sans-serif", "'Inter', system-ui, -apple-system, sans-serif"
	case "garamond", "artisan_garamond", "cormorant":
		return "'Cormorant Garamond', Garamond, 'Baskerville', serif", "'Plus Jakarta Sans', system-ui, -apple-system, sans-serif"
	default:
		// Default to Playfair Display (luxury editorial) & Plus Jakarta Sans
		return "'Playfair Display', Didot, 'Bodoni MT', serif", "'Plus Jakarta Sans', system-ui, -apple-system, sans-serif"
	}
}

// ServeFont serves locally embedded font files with immutable caching.
func ServeFont(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	content, err := templateFS.ReadFile("fonts/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(content)
}

var (
	templatesOnce     sync.Once
	storefrontTmpl    *template.Template
	suspendedTmpl     *template.Template
	productDetailTmpl *template.Template
	categoryTmpl      *template.Template
	aboutTmpl         *template.Template
	contactTmpl       *template.Template
	tmplErr           error
)

func initTemplates() {
	templatesOnce.Do(func() {
		funcs := template.FuncMap{
			"add": func(a, b int) int { return a + b },
			"urlquery": func(s string) string { return template.URLQueryEscaper(s) },
		}
		storefrontTmpl, tmplErr = template.New("storefront.html").Funcs(funcs).ParseFS(templateFS, "storefront.html")
		if tmplErr != nil {
			return
		}
		suspendedTmpl, tmplErr = template.New("suspended.html").Funcs(funcs).ParseFS(templateFS, "suspended.html")
		if tmplErr != nil {
			return
		}
		productDetailTmpl, tmplErr = template.New("product_detail.html").Funcs(funcs).ParseFS(templateFS, "product_detail.html")
		if tmplErr != nil {
			return
		}
		categoryTmpl, tmplErr = template.New("category_view.html").Funcs(funcs).ParseFS(templateFS, "category_view.html")
		if tmplErr != nil {
			return
		}
		aboutTmpl, tmplErr = template.New("about_view.html").Funcs(funcs).ParseFS(templateFS, "about_view.html")
		if tmplErr != nil {
			return
		}
		contactTmpl, tmplErr = template.New("contact_view.html").Funcs(funcs).ParseFS(templateFS, "contact_view.html")
	})
}

// ProductView represents a single catalog item for storefront rendering.
type ProductView struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Slug           string  `json:"slug"`
	Price          float64 `json:"price"`
	CompareAtPrice float64 `json:"compare_at_price,omitempty"`
	Description    string  `json:"description,omitempty"`
	ImageURL       string  `json:"image_url,omitempty"`
	Sizes          string  `json:"sizes,omitempty"`
	Colors         string  `json:"colors,omitempty"`
	Badge          string  `json:"badge,omitempty"`
	Category       string  `json:"category,omitempty"`
	PaymentLink    string  `json:"payment_link,omitempty"`
	InStock        bool    `json:"in_stock"`
}

// FeaturePillar represents a single craft pillar card.
type FeaturePillar struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

// Testimonial represents authentic customer feedback.
type Testimonial struct {
	Customer string `json:"customer"`
	Location string `json:"location"`
	Quote    string `json:"quote"`
	Item     string `json:"item"`
}

// FAQItem represents an accordion FAQ.
type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// StorefrontData carries data to render the luxury storefront page.
type StorefrontData struct {
	StoreName       string
	Slogan          string
	PrimaryColor    string
	BGTone          string
	CleanPhone      string
	ShopierURL      string
	AnnouncementBar string
	IsDemo          bool
	LayoutMode      string // "multipage", "single_page", "hybrid"
	FontPreset      string // "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"
	FontHeading     string
	FontBody        string

	// Rich Editorial Sections
	HeroBadge      string
	HeroTitle      string
	HeroSubtitle   string
	HeroImage      string
	StoryBadge     string
	StoryTitle     string
	StoryText      string
	StoryQuote     string
	StoryImage     string
	PillarsTitle   string
	Pillars        []FeaturePillar
	Testimonials   []Testimonial
	FAQs           []FAQItem
	AtelierAddress string
	AtelierHours   string
	Instagram      string

	Products         []ProductView
	FeaturedProducts []ProductView
	ProductsJSON     template.JS
}

// ProductDetailData carries data for dedicated product page.
type ProductDetailData struct {
	StoreName       string
	Slogan          string
	PrimaryColor    string
	BGTone          string
	CleanPhone      string
	ShopierURL      string
	FontPreset      string
	FontHeading     string
	FontBody        string
	Product         ProductView
	RelatedProducts []ProductView
}

// CategoryData carries data for category listing page.
type CategoryData struct {
	StoreName       string
	Slogan          string
	PrimaryColor    string
	BGTone          string
	CleanPhone      string
	FontPreset      string
	FontHeading     string
	FontBody        string
	CurrentCategory string
	Categories      []string
	Products        []ProductView
	TotalCount      int
}

// AboutData carries data for dedicated atelier/story page.
type AboutData struct {
	StoreName    string
	Slogan       string
	PrimaryColor string
	BGTone       string
	CleanPhone   string
	FontPreset   string
	FontHeading  string
	FontBody     string
	StoryBadge   string
	StoryTitle   string
	StoryText    string
	StoryQuote   string
	StoryImage   string
	PillarsTitle string
	Pillars      []FeaturePillar
}

// ContactData carries data for dedicated contact page.
type ContactData struct {
	StoreName      string
	Slogan         string
	PrimaryColor   string
	BGTone         string
	CleanPhone     string
	FontPreset     string
	FontHeading    string
	FontBody       string
	AtelierAddress string
	AtelierHours   string
	FAQs           []FAQItem
}

// SuspendedData carries data to render the maintenance/suspended page.
type SuspendedData struct {
	StoreName   string
	OwnerPhone  string
	CleanPhone  string
	FontPreset  string
	FontHeading string
	FontBody    string
}

// Render writes the luxury boutique storefront HTML to w.
func Render(w http.ResponseWriter, r *http.Request, data StorefrontData) error {
	initTemplates()
	if tmplErr != nil {
		return tmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Boutique Atelier"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#8C6D46"
	}
	if data.BGTone == "" {
		data.BGTone = "#FAF8F5"
	}
	if data.LayoutMode == "" {
		data.LayoutMode = "hybrid"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	if len(data.Products) > 4 {
		data.FeaturedProducts = data.Products[:4]
	} else {
		data.FeaturedProducts = data.Products
	}
	data.CleanPhone = cleanPhoneNumber(data.CleanPhone)

	prodsJSON, err := json.Marshal(data.Products)
	if err == nil {
		data.ProductsJSON = template.JS(prodsJSON) // #nosec G203 -- sanitized JSON of product catalog
	} else {
		data.ProductsJSON = template.JS("[]")
	}

	var buf bytes.Buffer
	if err := storefrontTmpl.Execute(&buf, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")
	_, err = buf.WriteTo(w)
	return err
}

// RenderProductDetail renders dedicated product detail view.
func RenderProductDetail(w http.ResponseWriter, r *http.Request, data ProductDetailData) error {
	initTemplates()
	if tmplErr != nil {
		return tmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Boutique Atelier"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#8C6D46"
	}
	if data.BGTone == "" {
		data.BGTone = "#FAF8F5"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	data.CleanPhone = cleanPhoneNumber(data.CleanPhone)

	var buf bytes.Buffer
	if err := productDetailTmpl.Execute(&buf, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")
	_, err := buf.WriteTo(w)
	return err
}

// RenderCategory renders category collections view.
func RenderCategory(w http.ResponseWriter, r *http.Request, data CategoryData) error {
	initTemplates()
	if tmplErr != nil {
		return tmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Boutique Atelier"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#8C6D46"
	}
	if data.BGTone == "" {
		data.BGTone = "#FAF8F5"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	data.CleanPhone = cleanPhoneNumber(data.CleanPhone)

	var buf bytes.Buffer
	if err := categoryTmpl.Execute(&buf, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")
	_, err := buf.WriteTo(w)
	return err
}

// RenderAbout renders dedicated atelier story view.
func RenderAbout(w http.ResponseWriter, r *http.Request, data AboutData) error {
	initTemplates()
	if tmplErr != nil {
		return tmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Boutique Atelier"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#8C6D46"
	}
	if data.BGTone == "" {
		data.BGTone = "#FAF8F5"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	data.CleanPhone = cleanPhoneNumber(data.CleanPhone)

	var buf bytes.Buffer
	if err := aboutTmpl.Execute(&buf, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")
	_, err := buf.WriteTo(w)
	return err
}

// RenderContact renders dedicated contact and showroom view.
func RenderContact(w http.ResponseWriter, r *http.Request, data ContactData) error {
	initTemplates()
	if tmplErr != nil {
		return tmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Boutique Atelier"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#8C6D46"
	}
	if data.BGTone == "" {
		data.BGTone = "#FAF8F5"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	data.CleanPhone = cleanPhoneNumber(data.CleanPhone)

	var buf bytes.Buffer
	if err := contactTmpl.Execute(&buf, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")
	_, err := buf.WriteTo(w)
	return err
}

// RenderSuspended writes the luxury suspended/maintenance HTML to w.
func RenderSuspended(w http.ResponseWriter, r *http.Request, data SuspendedData) error {
	initTemplates()
	if tmplErr != nil {
		return tmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Mağaza"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	data.CleanPhone = cleanPhoneNumber(data.OwnerPhone)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; frame-ancestors 'none';")
	w.WriteHeader(http.StatusServiceUnavailable)
	return suspendedTmpl.Execute(w, data)
}

func cleanPhoneNumber(phone string) string {
	var sb strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	out := sb.String()
	if strings.HasPrefix(out, "0") {
		out = "9" + out
	}
	if !strings.HasPrefix(out, "90") && len(out) == 10 {
		out = "90" + out
	}
	return out
}
