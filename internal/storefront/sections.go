package storefront

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"strings"
)

// SectionConfig defines a single layout section in store_settings.layout_json.
type SectionConfig struct {
	Type  string         `json:"type"`
	Props map[string]any `json:"props"`
}

// ParseAndRenderSections parses a layout_json string and renders each valid section.
// Unknown section types are skipped without error to ensure resilience.
func ParseAndRenderSections(layoutJSON string, data StorefrontData) ([]template.HTML, error) {
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

		out := renderSingleSection(t, props, data)
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

func renderSingleSection(secType string, props map[string]any, data StorefrontData) template.HTML {
	switch secType {
	case "hero":
		badge := propString(props, "badge", data.HeroBadge)
		title := propString(props, "title", data.HeroTitle)
		subtitle := propString(props, "subtitle", data.HeroSubtitle)
		image := propString(props, "image_url", data.HeroImage)
		ctaText := propString(props, "cta_text", "Koleksiyonu Keşfet &rarr;")
		ctaLink := propString(props, "cta_link", "/kategori/tumu")
		secCtaText := propString(props, "secondary_cta_text", "Atölye Hikayemiz &rarr;")
		secCtaLink := propString(props, "secondary_cta_link", "/hakkimizda")

		badgeHTML := ""
		if badge != "" {
			badgeHTML = fmt.Sprintf(`<span class="hero-badge">%s</span>`, html.EscapeString(badge))
		}
		if image == "" {
			image = "https://images.unsplash.com/photo-1565193566173-7a0ee3dbe261?auto=format&fit=crop&w=1200&q=80"
		}

		return template.HTML(fmt.Sprintf(`
<section class="hero-section">
  <div class="container">
    <div class="hero-grid">
      <div>
        %s
        <h2 class="hero-title">%s</h2>
        <p class="hero-subtitle">%s</p>
        <div class="hero-actions">
          <a href="%s" class="btn-primary">%s</a>
          <a href="%s" class="btn-secondary">%s</a>
        </div>
      </div>
      <div>
        <div class="hero-visual">
          <img src="%s" alt="%s" loading="eager">
        </div>
      </div>
    </div>
  </div>
</section>`,
			badgeHTML,
			html.EscapeString(title),
			html.EscapeString(subtitle),
			html.EscapeString(ctaLink),
			html.EscapeString(ctaText),
			html.EscapeString(secCtaLink),
			html.EscapeString(secCtaText),
			html.EscapeString(image),
			html.EscapeString(data.StoreName),
		))

	case "editorial", "story":
		badge := propString(props, "badge", data.StoryBadge)
		title := propString(props, "title", data.StoryTitle)
		text := propString(props, "text", data.StoryText)
		quote := propString(props, "quote", data.StoryQuote)
		image := propString(props, "image_url", data.StoryImage)

		badgeHTML := ""
		if badge != "" {
			badgeHTML = fmt.Sprintf(`<span class="hero-badge">%s</span>`, html.EscapeString(badge))
		}
		quoteHTML := ""
		if quote != "" {
			quoteHTML = fmt.Sprintf(`<div class="story-quote">%s</div>`, html.EscapeString(quote))
		}
		if image == "" {
			image = "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=1200&q=80"
		}

		return template.HTML(fmt.Sprintf(`
<section class="story-section" id="hikaye">
  <div class="container">
    <div class="story-grid">
      <div class="story-visual">
        <img src="%s" alt="%s" loading="lazy">
      </div>
      <div class="story-content">
        %s
        <h3 class="story-title">%s</h3>
        <p class="story-text">%s</p>
        %s
      </div>
    </div>
  </div>
</section>`,
			html.EscapeString(image),
			html.EscapeString(data.StoreName),
			badgeHTML,
			html.EscapeString(title),
			html.EscapeString(text),
			quoteHTML,
		))

	case "featured_grid", "collection":
		title := propString(props, "title", "Seçkin Koleksiyon")
		subtitle := propString(props, "subtitle", "Öne Çıkan Seçkiler")

		var prodCards strings.Builder
		for i, p := range data.Products {
			if i >= 8 {
				break
			}
			badgeSpan := ""
			if p.Badge != "" {
				badgeSpan = fmt.Sprintf(`<span class="product-badge">%s</span>`, html.EscapeString(p.Badge))
			}
			imgSrc := p.ImageURL
			if imgSrc == "" {
				imgSrc = "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=600&q=80"
			}

			prodCards.WriteString(fmt.Sprintf(`
<div class="product-card" onclick="openDrawer('%s')">
  <div class="product-visual">
    <img src="%s" alt="%s" loading="lazy">
    %s
  </div>
  <div class="product-info">
    <h4 class="product-title">%s</h4>
    <div class="product-price">%.0f TL</div>
  </div>
</div>`,
				html.EscapeString(p.ID),
				html.EscapeString(imgSrc),
				html.EscapeString(p.Title),
				badgeSpan,
				html.EscapeString(p.Title),
				p.Price,
			))
		}

		return template.HTML(fmt.Sprintf(`
<section id="koleksiyon" style="padding: 70px 0;">
  <div class="container">
    <div class="catalog-header">
      <div>
        <p class="section-eyebrow">%s</p>
        <h3 class="catalog-title">%s</h3>
      </div>
      <a href="/kategori/tumu" class="btn-secondary" style="font-size: 13px; text-decoration: none;">
        Tüm Koleksiyonu Gör (%d Parça) &rarr;
      </a>
    </div>
    <div class="catalog-grid">
      %s
    </div>
  </div>
</section>`,
			html.EscapeString(strings.ToUpper(subtitle)),
			html.EscapeString(title),
			len(data.Products),
			prodCards.String(),
		))

	case "pillars", "features":
		title := propString(props, "title", "Zanaat İlkelerimiz")
		var pillarsBuf bytes.Buffer

		// Check if items supplied in props
		if items, ok := props["items"].([]any); ok && len(items) > 0 {
			for _, it := range items {
				if m, ok := it.(map[string]any); ok {
					pTitle := propString(m, "title", "")
					pDesc := propString(m, "desc", "")
					pillarsBuf.WriteString(fmt.Sprintf(`
<div class="pillar-card">
  <h4 class="pillar-title">%s</h4>
  <p class="pillar-desc">%s</p>
</div>`, html.EscapeString(pTitle), html.EscapeString(pDesc)))
				}
			}
		} else {
			for _, p := range data.Pillars {
				pillarsBuf.WriteString(fmt.Sprintf(`
<div class="pillar-card">
  <h4 class="pillar-title">%s</h4>
  <p class="pillar-desc">%s</p>
</div>`, html.EscapeString(p.Title), html.EscapeString(p.Desc)))
			}
		}

		return template.HTML(fmt.Sprintf(`
<section class="pillars-section" id="ilkeler">
  <div class="container">
    <h3 class="section-title text-center" style="margin-bottom: 40px;">%s</h3>
    <div class="pillars-grid">
      %s
    </div>
  </div>
</section>`, html.EscapeString(title), pillarsBuf.String()))

	case "newsletter":
		title := propString(props, "title", "Özel Koleksiyon Davetleri")
		subtitle := propString(props, "subtitle", "Yeni çıkan sınırlı üretim serilerden ilk siz haberdar olun.")
		btnText := propString(props, "button_text", "Kayıt Ol")

		return template.HTML(fmt.Sprintf(`
<section style="padding: 60px 0; border-top: 1px solid var(--border); background: var(--card-bg);">
  <div class="container text-center" style="max-width: 600px; margin: 0 auto;">
    <h3 class="section-title">%s</h3>
    <p class="section-subtitle" style="margin-bottom: 24px;">%s</p>
    <div style="display: flex; gap: 8px; justify-content: center;">
      <input type="email" placeholder="E-posta adresiniz" class="form-input" style="max-width: 320px;">
      <button class="btn-primary">%s</button>
    </div>
  </div>
</section>`, html.EscapeString(title), html.EscapeString(subtitle), html.EscapeString(btnText)))

	case "contact_bar":
		addr := propString(props, "address", data.AtelierAddress)
		hours := propString(props, "hours", data.AtelierHours)
		phone := propString(props, "phone", data.CleanPhone)

		return template.HTML(fmt.Sprintf(`
<section style="padding: 40px 0; border-top: 1px solid var(--border); background: var(--bg-tone);">
  <div class="container" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 20px;">
    <div>
      <div style="font-weight: 600; margin-bottom: 4px;">Atölye ve Randevu</div>
      <div style="font-size: 13px; color: var(--text-muted);">%s · %s</div>
    </div>
    <div>
      <a href="https://wa.me/%s" target="_blank" rel="noopener noreferrer" class="btn-secondary" style="font-size: 13px;">
        WhatsApp ile İletişime Geç &rarr;
      </a>
    </div>
  </div>
</section>`, html.EscapeString(addr), html.EscapeString(hours), html.EscapeString(phone)))

	default:
		// Unknown section types are safely skipped!
		return ""
	}
}
