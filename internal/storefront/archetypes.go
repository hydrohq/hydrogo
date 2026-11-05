package storefront

import (
	"bytes"
	"html/template"
	"net/http"
	"sync"
)

var (
	archetypesOnce sync.Once
	corporateTmpl  *template.Template
	restaurantTmpl *template.Template
	saasTmpl       *template.Template
	portfolioTmpl  *template.Template
	showcaseTmpl   *template.Template
	archetypeErr   error
)

func initArchetypeTemplates() {
	archetypesOnce.Do(func() {
		funcs := template.FuncMap{
			"add":      func(a, b int) int { return a + b },
			"urlquery": func(s string) string { return template.URLQueryEscaper(s) },
		}
		corporateTmpl, archetypeErr = template.New("corporate.html").Funcs(funcs).ParseFS(templateFS, "corporate.html", "sections/*.html")
		if archetypeErr != nil {
			return
		}
		restaurantTmpl, archetypeErr = template.New("restaurant.html").Funcs(funcs).ParseFS(templateFS, "restaurant.html", "sections/*.html")
		if archetypeErr != nil {
			return
		}
		saasTmpl, archetypeErr = template.New("saas.html").Funcs(funcs).ParseFS(templateFS, "saas.html", "sections/*.html")
		if archetypeErr != nil {
			return
		}
		portfolioTmpl, archetypeErr = template.New("portfolio.html").Funcs(funcs).ParseFS(templateFS, "portfolio.html", "sections/*.html")
		if archetypeErr != nil {
			return
		}
		showcaseTmpl, archetypeErr = template.New("showcase.html").Funcs(funcs).ParseFS(templateFS, "showcase.html", "sections/*.html")
	})
}

// CorporateService represents a single service offering.
type CorporateService struct {
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	Icon             string `json:"icon"`
	ShortDescription string `json:"short_description"`
	Featured         bool   `json:"featured"`
}

// CorporateProject represents a case study or project.
type CorporateProject struct {
	Title      string `json:"title"`
	Slug       string `json:"slug"`
	ClientName string `json:"client_name"`
	Category   string `json:"category"`
	ImageURL   string `json:"image_url"`
	Summary    string `json:"summary"`
	Metrics    string `json:"metrics"`
	Featured   bool   `json:"featured"`
}

// CorporateTestimonial represents a client endorsement.
type CorporateTestimonial struct {
	ClientName string `json:"client_name"`
	Company    string `json:"company"`
	Role       string `json:"role"`
	Quote      string `json:"quote"`
	AvatarURL  string `json:"avatar_url"`
}

// CorporateTeamMember represents an executive or team member.
type CorporateTeamMember struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	LinkedIn  string `json:"linkedin"`
}

// CorporateData carries data for modern corporate agency sites.
type CorporateData struct {
	BrandName    string
	Tagline      string
	PrimaryColor string
	BGTone       string
	FontPreset   string
	FontHeading  template.CSS
	FontBody     template.CSS
	Phone        string
	Email        string
	Address      string
	HeroBadge    string
	HeroTitle    string
	HeroSubtitle string
	AboutTitle   string
	AboutText    string
	Stat1Val     string
	Stat1Lbl     string
	Stat2Val     string
	Stat2Lbl     string
	Stat3Val     string
	Stat3Lbl     string
	Services     []CorporateService
	Projects     []CorporateProject
	Testimonials []CorporateTestimonial
	Team         []CorporateTeamMember
}

// RestaurantCategory represents a menu category.
type RestaurantCategory struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
}

// RestaurantItem represents a dish or beverage.
type RestaurantItem struct {
	Name          string  `json:"name"`
	CategorySlug  string  `json:"category_slug"`
	Price         float64 `json:"price"`
	Description   string  `json:"description"`
	ImageURL      string  `json:"image_url"`
	IsChefSpecial bool    `json:"is_chef_special"`
	IsVegetarian  bool    `json:"is_vegetarian"`
	IsGlutenFree  bool    `json:"is_gluten_free"`
	IsAvailable   bool    `json:"is_available"`
}

// RestaurantData carries data for restaurant & QR menu sites.
type RestaurantData struct {
	BrandName      string
	Tagline        string
	PrimaryColor   string
	BGTone         string
	FontPreset     string
	FontHeading    template.CSS
	FontBody       template.CSS
	Phone          string
	CleanPhone     string
	WhatsAppNumber string
	Address        string
	OpeningHours   string
	HeroBadge      string
	HeroTitle      string
	HeroSubtitle   string
	HeroImage      string
	Categories     []RestaurantCategory
	Items          []RestaurantItem
}

// SaaSFeature represents a core feature card in Bento grid.
type SaaSFeature struct {
	Title       string `json:"title"`
	Badge       string `json:"badge"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
}

// SaaSPricingTier represents a subscription plan.
type SaaSPricingTier struct {
	Name          string   `json:"name"`
	PriceMonthly  float64  `json:"price_monthly"`
	PriceAnnual   float64  `json:"price_annual"`
	Description   string   `json:"description"`
	FeaturesList  string   `json:"features_list"`
	Features      []string `json:"-"`
	IsHighlighted bool     `json:"is_highlighted"`
	ButtonLabel   string   `json:"button_label"`
	ButtonLink    string   `json:"button_link"`
}

// SaaSFAQ represents a frequently asked question.
type SaaSFAQ struct {
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	Category  string `json:"category"`
	SortOrder int    `json:"sort_order"`
}

// SaaSData carries data for modern SaaS product landing pages.
type SaaSData struct {
	BrandName    string
	Tagline      string
	PrimaryColor string
	BGTone       string
	FontPreset   string
	FontHeading  template.CSS
	FontBody     template.CSS
	ContactEmail string
	HeroBadge    string
	HeroTitle    string
	HeroSubtitle string
	HeroImage    string
	CTALabel     string
	CTALink      string
	Features     []SaaSFeature
	PricingTiers []SaaSPricingTier
	FAQs         []SaaSFAQ
}

// PortfolioWork represents a project or artwork in portfolio.
type PortfolioWork struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Category    string `json:"category"`
	Year        string `json:"year"`
	Client      string `json:"client"`
	CoverImage  string `json:"cover_image"`
	Description string `json:"description"`
	Featured    bool   `json:"featured"`
}

// PortfolioAward represents an award or publication.
type PortfolioAward struct {
	Title        string `json:"title"`
	Organization string `json:"organization"`
	Year         string `json:"year"`
	ProjectName  string `json:"project_name"`
}

// PortfolioData carries data for architect/artist studio portfolio.
type PortfolioData struct {
	ArtistName   string
	Tagline      string
	PrimaryColor string
	BGTone       string
	FontPreset   string
	FontHeading  template.CSS
	FontBody     template.CSS
	ContactEmail string
	Phone        string
	Location     string
	Instagram    string
	Bio          string
	HeroTitle    string
	HeroSubtitle string
	Works        []PortfolioWork
	Awards       []PortfolioAward
}

// ShowcaseSite represents an active demo site shown in the showcase gallery.
type ShowcaseSite struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Archetype   string `json:"archetype"`
	URL         string `json:"url"`
	Subdomain   string `json:"subdomain"`
	Badge       string `json:"badge"`
	Description string `json:"description"`
	CoverImage  string `json:"cover_image"`
}

// ShowcaseData carries data for the official HydroGo showcase and landing page.
type ShowcaseData struct {
	BaseDomain    string
	TotalSites    int
	ShowcaseSites []ShowcaseSite
}

// RenderCorporate writes the modern corporate agency HTML to w.
func RenderCorporate(w http.ResponseWriter, r *http.Request, data CorporateData) error {
	initArchetypeTemplates()
	if archetypeErr != nil {
		return archetypeErr
	}
	if data.BrandName == "" {
		data.BrandName = "Corporate Enterprise"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#1E3A8A"
	}
	if data.BGTone == "" {
		data.BGTone = "#F8FAFC"
	}
	if data.FontPreset == "" {
		data.FontPreset = "v0_geist"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}

	var buf bytes.Buffer
	if err := corporateTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}

// RenderRestaurant writes the gourmet restaurant & QR menu HTML to w.
func RenderRestaurant(w http.ResponseWriter, r *http.Request, data RestaurantData) error {
	initArchetypeTemplates()
	if archetypeErr != nil {
		return archetypeErr
	}
	if data.BrandName == "" {
		data.BrandName = "Bistro & Lounge"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#B45309"
	}
	if data.BGTone == "" {
		data.BGTone = "#18181B"
	}
	if data.FontPreset == "" {
		data.FontPreset = "artisan_garamond"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	data.CleanPhone = cleanPhoneNumber(data.Phone)

	var buf bytes.Buffer
	if err := restaurantTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}

// RenderSaaS writes the high-conversion SaaS landing page HTML to w.
func RenderSaaS(w http.ResponseWriter, r *http.Request, data SaaSData) error {
	initArchetypeTemplates()
	if archetypeErr != nil {
		return archetypeErr
	}
	if data.BrandName == "" {
		data.BrandName = "NovaFlow AI"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#2563EB"
	}
	if data.BGTone == "" {
		data.BGTone = "#09090B"
	}
	if data.FontPreset == "" {
		data.FontPreset = "v0_geist"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}

	var buf bytes.Buffer
	if err := saasTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}

// RenderPortfolio writes the creative studio portfolio HTML to w.
func RenderPortfolio(w http.ResponseWriter, r *http.Request, data PortfolioData) error {
	initArchetypeTemplates()
	if archetypeErr != nil {
		return archetypeErr
	}
	if data.ArtistName == "" {
		data.ArtistName = "Apex Studio"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#0F172A"
	}
	if data.BGTone == "" {
		data.BGTone = "#F8FAFC"
	}
	if data.FontPreset == "" {
		data.FontPreset = "monumental_cinzel"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}

	var buf bytes.Buffer
	if err := portfolioTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}

// RenderShowcase writes the official platform showcase and onboarding landing HTML to w.
func RenderShowcase(w http.ResponseWriter, r *http.Request, data ShowcaseData) error {
	initArchetypeTemplates()
	if archetypeErr != nil {
		return archetypeErr
	}
	if data.BaseDomain == "" {
		data.BaseDomain = "ardalabs.dev"
	}

	var buf bytes.Buffer
	if err := showcaseTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}
