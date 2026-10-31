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

//go:embed storefront.html suspended.html
var templateFS embed.FS

var (
	templatesOnce  sync.Once
	storefrontTmpl *template.Template
	suspendedTmpl  *template.Template
	tmplErr        error
)

func initTemplates() {
	templatesOnce.Do(func() {
		funcs := template.FuncMap{
			"add": func(a, b int) int { return a + b },
		}
		storefrontTmpl, tmplErr = template.New("storefront.html").Funcs(funcs).ParseFS(templateFS, "storefront.html")
		if tmplErr != nil {
			return
		}
		suspendedTmpl, tmplErr = template.New("suspended.html").Funcs(funcs).ParseFS(templateFS, "suspended.html")
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

	Products     []ProductView
	ProductsJSON template.JS
}

// SuspendedData carries data to render the maintenance/suspended page.
type SuspendedData struct {
	StoreName  string
	OwnerPhone string
	CleanPhone string
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
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' data: https://fonts.gstatic.com; img-src 'self' data: blob: https:; connect-src 'self'; frame-ancestors 'none';")
	_, err = buf.WriteTo(w)
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
	data.CleanPhone = cleanPhoneNumber(data.OwnerPhone)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' data: https://fonts.gstatic.com; img-src 'self' data: blob: https:; frame-ancestors 'none';")
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
