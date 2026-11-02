package storefront

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"
)

// SectionConfig defines a single layout section in store_settings.layout_json.
type SectionConfig struct {
	Type  string         `json:"type"`
	Props map[string]any `json:"props"`
}

var (
	sectionTemplatesOnce sync.Once
	sectionTemplates     *template.Template
	sectionTemplateErr   error
)

func initSectionTemplates() {
	sectionTemplatesOnce.Do(func() {
		funcs := template.FuncMap{
			"add": func(a, b int) int { return a + b },
		}
		sectionTemplates, sectionTemplateErr = template.New("sections").Funcs(funcs).ParseFS(templateFS, "sections/*.html")
	})
}

// ParseAndRenderSections parses a layout_json string and renders each valid section using template partials.
// Unknown section types are skipped without error to ensure resilience.
func ParseAndRenderSections(layoutJSON string, data StorefrontData) ([]template.HTML, error) {
	initSectionTemplates()
	if sectionTemplateErr != nil {
		return nil, sectionTemplateErr
	}

	raw := strings.TrimSpace(layoutJSON)
	if raw == "" || raw == "[]" {
		return nil, nil
	}

	var configs []SectionConfig
	if err := json.Unmarshal([]byte(raw), &configs); err != nil {
		return nil, fmt.Errorf("invalid layout_json: %w", err)
	}

	rendered := make([]template.HTML, 0, len(configs))
	for _, sec := range configs {
		t := strings.ToLower(strings.TrimSpace(sec.Type))
		props := sec.Props
		if props == nil {
			props = map[string]any{}
		}

		out := renderTemplateSection(t, props, data)
		if out != "" {
			rendered = append(rendered, out)
		}
	}

	return rendered, nil
}

func propString(props map[string]any, key, fallback string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return fallback
}

func renderTemplateSection(secType string, props map[string]any, data StorefrontData) template.HTML {
	tmplName := secType + ".html"
	if secType == "story" {
		tmplName = "editorial.html"
	} else if secType == "collection" {
		tmplName = "featured_grid.html"
	} else if secType == "features" {
		tmplName = "pillars.html"
	}

	if sectionTemplates.Lookup(tmplName) == nil {
		return ""
	}

	var buf bytes.Buffer
	var err error

	switch tmplName {
	case "hero.html":
		image := propString(props, "image_url", data.HeroImage)
		if image == "" {
			image = "https://images.unsplash.com/photo-1565193566173-7a0ee3dbe261?auto=format&fit=crop&w=1200&q=80"
		}
		err = sectionTemplates.ExecuteTemplate(&buf, tmplName, struct {
			Badge            string
			Title            string
			Subtitle         string
			CTAText          string
			CTALink          string
			SecondaryCTAText string
			SecondaryCTALink string
			ImageURL         string
			StoreName        string
		}{
			Badge:            propString(props, "badge", data.HeroBadge),
			Title:            propString(props, "title", data.HeroTitle),
			Subtitle:         propString(props, "subtitle", data.HeroSubtitle),
			CTAText:          propString(props, "cta_text", "Koleksiyonu Keşfet →"),
			CTALink:          propString(props, "cta_link", "/kategori/tumu"),
			SecondaryCTAText: propString(props, "secondary_cta_text", "Atölye Hikayemiz →"),
			SecondaryCTALink: propString(props, "secondary_cta_link", "/hakkimizda"),
			ImageURL:         image,
			StoreName:        data.StoreName,
		})

	case "editorial.html":
		image := propString(props, "image_url", data.StoryImage)
		if image == "" {
			image = "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=1200&q=80"
		}
		err = sectionTemplates.ExecuteTemplate(&buf, tmplName, struct {
			Badge     string
			Title     string
			Text      string
			Quote     string
			ImageURL  string
			StoreName string
		}{
			Badge:     propString(props, "badge", data.StoryBadge),
			Title:     propString(props, "title", data.StoryTitle),
			Text:      propString(props, "text", data.StoryText),
			Quote:     propString(props, "quote", data.StoryQuote),
			ImageURL:  image,
			StoreName: data.StoreName,
		})

	case "featured_grid.html":
		prods := data.Products
		if len(prods) > 8 {
			prods = prods[:8]
		}
		err = sectionTemplates.ExecuteTemplate(&buf, tmplName, struct {
			Title    string
			Subtitle string
			Products []ProductView
		}{
			Title:    propString(props, "title", "Seçkin Koleksiyon"),
			Subtitle: strings.ToUpper(propString(props, "subtitle", "Öne Çıkan Seçkiler")),
			Products: prods,
		})

	case "pillars.html":
		type item struct {
			Title string
			Desc  string
		}
		var items []item
		if rawItems, ok := props["items"].([]any); ok && len(rawItems) > 0 {
			for _, it := range rawItems {
				if m, ok := it.(map[string]any); ok {
					items = append(items, item{
						Title: propString(m, "title", ""),
						Desc:  propString(m, "desc", ""),
					})
				}
			}
		} else {
			for _, p := range data.Pillars {
				items = append(items, item{Title: p.Title, Desc: p.Desc})
			}
		}
		err = sectionTemplates.ExecuteTemplate(&buf, tmplName, struct {
			Title string
			Items []item
		}{
			Title: propString(props, "title", "Zanaat İlkelerimiz"),
			Items: items,
		})

	case "newsletter.html":
		err = sectionTemplates.ExecuteTemplate(&buf, tmplName, struct {
			Title      string
			Subtitle   string
			ButtonText string
		}{
			Title:      propString(props, "title", "Özel Koleksiyon Davetleri"),
			Subtitle:   propString(props, "subtitle", "Yeni çıkan sınırlı üretim serilerden ilk siz haberdar olun."),
			ButtonText: propString(props, "button_text", "Kayıt Ol"),
		})

	case "contact_bar.html":
		err = sectionTemplates.ExecuteTemplate(&buf, tmplName, struct {
			Address string
			Hours   string
			Phone   string
		}{
			Address: propString(props, "address", data.AtelierAddress),
			Hours:   propString(props, "hours", data.AtelierHours),
			Phone:   propString(props, "phone", data.CleanPhone),
		})
	}

	if err != nil {
		return ""
	}
	return template.HTML(buf.String())
}
