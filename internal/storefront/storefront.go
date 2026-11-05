package storefront

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

//go:embed *.html sections/*.html fonts/*
var templateFS embed.FS

var (
	preloadedFonts     map[string][]byte
	preloadedFontsOnce sync.Once
)

func initFonts() {
	preloadedFontsOnce.Do(func() {
		preloadedFonts = make(map[string][]byte)
		entries, err := templateFS.ReadDir("fonts")
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := templateFS.ReadFile("fonts/" + entry.Name())
			if err == nil {
				preloadedFonts[entry.Name()] = data
			}
		}
	})
}

func init() {
	initFonts()
}

// ResolveFontFamilies returns CSS font families (heading, body) for a given typography preset.
func ResolveFontFamilies(preset string) (heading, body template.CSS) {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "playfair", "editorial_playfair", "playfair_jakarta":
		return "'Playfair Display', Didot, 'Bodoni MT', 'Times New Roman', serif", "'Geist', 'Plus Jakarta Sans', system-ui, sans-serif"
	case "cinzel", "monumental_cinzel", "cinzel_inter":
		return "'Cinzel', 'Trajan Pro', 'Bodoni MT', serif", "'Geist', 'Inter', system-ui, sans-serif"
	case "syne", "avantgarde_syne", "syne_inter":
		return "'Syne', 'Futura', system-ui, sans-serif", "'Geist', 'Inter', system-ui, sans-serif"
	case "modern", "modern_grotesk", "jakarta_inter":
		return "'Plus Jakarta Sans', system-ui, -apple-system, sans-serif", "'Geist', 'Inter', system-ui, sans-serif"
	case "garamond", "artisan_garamond", "cormorant":
		return "'Cormorant Garamond', Garamond, 'Baskerville', serif", "'Geist', 'Plus Jakarta Sans', system-ui, sans-serif"
	case "geist", "v0", "v0_geist", "modern_geist":
		fallthrough
	default:
		// Default to v0 Geist: ultra-clean, modern, Swiss/Vercel aesthetic!
		return "'Geist', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", "'Geist', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
	}
}

// ServeFont serves locally embedded font files from preloaded memory with immutable caching.
func ServeFont(w http.ResponseWriter, r *http.Request) {
	initFonts()
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	content, ok := preloadedFonts[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(content)
}

func writeHTMLWithConditionalCache(w http.ResponseWriter, r *http.Request, body []byte) error {
	sum := sha256.Sum256(body)
	etag := `W/"` + hex.EncodeToString(sum[:16]) + `"`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")

	if r != nil {
		if match := r.Header.Get("If-None-Match"); match != "" {
			if match == etag || match == "*" || strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return nil
			}
		}
	}

	_, err := w.Write(body)
	return err
}

var (
	templatesOnce     sync.Once
	storefrontTmpl    *template.Template
	suspendedTmpl     *template.Template
	productDetailTmpl *template.Template
	categoryTmpl      *template.Template
	aboutTmpl         *template.Template
	contactTmpl       *template.Template
	orderTrackingTmpl *template.Template
	tmplErr           error
)

func initTemplates() {
	templatesOnce.Do(func() {
		funcs := template.FuncMap{
			"add":      func(a, b int) int { return a + b },
			"urlquery": func(s string) string { return template.URLQueryEscaper(s) },
		}
		storefrontTmpl, tmplErr = template.New("storefront.html").Funcs(funcs).ParseFS(templateFS, "storefront.html", "sections/*.html")
		if tmplErr != nil {
			return
		}
		suspendedTmpl, tmplErr = template.New("suspended.html").Funcs(funcs).ParseFS(templateFS, "suspended.html")
		if tmplErr != nil {
			return
		}
		productDetailTmpl, tmplErr = template.New("product_detail.html").Funcs(funcs).ParseFS(templateFS, "product_detail.html", "sections/*.html")
		if tmplErr != nil {
			return
		}
		categoryTmpl, tmplErr = template.New("category_view.html").Funcs(funcs).ParseFS(templateFS, "category_view.html", "sections/*.html")
		if tmplErr != nil {
			return
		}
		aboutTmpl, tmplErr = template.New("about_view.html").Funcs(funcs).ParseFS(templateFS, "about_view.html", "sections/*.html")
		if tmplErr != nil {
			return
		}
		contactTmpl, tmplErr = template.New("contact_view.html").Funcs(funcs).ParseFS(templateFS, "contact_view.html", "sections/*.html")
		if tmplErr != nil {
			return
		}
		orderTrackingTmpl, tmplErr = template.New("order_tracking.html").Funcs(funcs).ParseFS(templateFS, "order_tracking.html", "sections/*.html")
	})
}

// OptionDef represents an option dimension (e.g. Renk, Beden).
type OptionDef struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// VariantDef represents a specific SKU with pricing adjustment and stock.
type VariantDef struct {
	SKU       string   `json:"sku"`
	Title     string   `json:"title"`
	PriceDiff float64  `json:"price_diff"`
	Stock     int      `json:"stock"`
	Options   []string `json:"options,omitempty"`
}

// ProductView represents a single catalog item for storefront rendering.
type ProductView struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	Slug           string       `json:"slug"`
	Price          float64      `json:"price"`
	CompareAtPrice float64      `json:"compare_at_price,omitempty"`
	Description    string       `json:"description,omitempty"`
	ImageURL       string       `json:"image_url,omitempty"`
	SecondaryImage string       `json:"secondary_image,omitempty"`
	Sizes          string       `json:"sizes,omitempty"`
	Colors         string       `json:"colors,omitempty"`
	Badge          string       `json:"badge,omitempty"`
	Category       string       `json:"category,omitempty"`
	PaymentLink    string       `json:"payment_link,omitempty"`
	InStock        bool         `json:"in_stock"`
	Options        []OptionDef  `json:"options,omitempty"`
	Variants       []VariantDef `json:"variants,omitempty"`
	OptionsJSON    template.JS  `json:"-"`
	VariantsJSON   template.JS  `json:"-"`
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
	FontHeading     template.CSS
	FontBody        template.CSS

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

	// Dynamic Sectional Architecture (ADR-0031)
	LayoutJSON       string
	RenderedSections []template.HTML
	HasCustomLayout  bool
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
	FontHeading     template.CSS
	FontBody        template.CSS
	Product         ProductView
	RelatedProducts []ProductView
	ProductsJSON    template.JS
}

// CategoryData carries data for category listing page.
type CategoryData struct {
	StoreName       string
	Slogan          string
	PrimaryColor    string
	BGTone          string
	CleanPhone      string
	ShopierURL      string
	FontPreset      string
	FontHeading     template.CSS
	FontBody        template.CSS
	CurrentCategory string
	Categories      []string
	Products        []ProductView
	TotalCount      int
	ProductsJSON    template.JS
}

// AboutData carries data for dedicated atelier/story page.
type AboutData struct {
	StoreName    string
	Slogan       string
	PrimaryColor string
	BGTone       string
	CleanPhone   string
	ShopierURL   string
	FontPreset   string
	FontHeading  template.CSS
	FontBody     template.CSS
	StoryBadge   string
	StoryTitle   string
	StoryText    string
	StoryQuote   string
	StoryImage   string
	PillarsTitle string
	Pillars      []FeaturePillar
	ProductsJSON template.JS
}

// ContactData carries data for dedicated contact page.
type ContactData struct {
	StoreName      string
	Slogan         string
	PrimaryColor   string
	BGTone         string
	CleanPhone     string
	ShopierURL     string
	FontPreset     string
	FontHeading    template.CSS
	FontBody       template.CSS
	AtelierAddress string
	AtelierHours   string
	FAQs           []FAQItem
	ProductsJSON   template.JS
}

// OrderTrackingData carries data for order status tracking page.
type OrderTrackingData struct {
	StoreName       string
	Slogan          string
	PrimaryColor    string
	BGTone          string
	CleanPhone      string
	ShopierURL      string
	FontPreset      string
	FontHeading     template.CSS
	FontBody        template.CSS
	QueryCode       string
	OrderFound      bool
	OrderCode       string
	OrderStatus     string // yeni, onaylandi, kargolandi, teslim_edildi, iptal
	CustomerName    string
	CustomerPhone   string
	CustomerAddress string
	ItemsSummary    string
	TotalAmount     float64
	TrackingCode    string
	Carrier         string
	OrderDate       string
	ProductsJSON    template.JS
}

// SuspendedData carries data to render the maintenance/suspended page.
type SuspendedData struct {
	StoreName   string
	OwnerPhone  string
	CleanPhone  string
	FontPreset  string
	FontHeading template.CSS
	FontBody    template.CSS
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
		data.LayoutMode = "multipage"
	}
	if data.FontPreset == "" {
		data.FontPreset = "v0_geist"
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

	if strings.TrimSpace(data.LayoutJSON) != "" {
		secs, err := ParseAndRenderSections(data.LayoutJSON, data)
		if err == nil && len(secs) > 0 {
			data.RenderedSections = secs
			data.HasCustomLayout = true
		}
	}

	var buf bytes.Buffer
	if err := storefrontTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
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
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
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
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
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
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
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
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}

// RenderOrderTracking renders dedicated order tracking view.
func RenderOrderTracking(w http.ResponseWriter, r *http.Request, data OrderTrackingData) error {
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
	if err := orderTrackingTmpl.Execute(&buf, data); err != nil {
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
