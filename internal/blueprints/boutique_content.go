package blueprints

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/ardaozyildirim/hydrogo/internal/blueprints/instructions"
)

// NicheEditorial carries full rich storytelling, craft pillars, testimonials, and FAQs for a boutique.
type NicheEditorial struct {
	HeroBadge      string                 `json:"hero_badge"`
	HeroTitle      string                 `json:"hero_title"`
	HeroSubtitle   string                 `json:"hero_subtitle"`
	HeroImage      string                 `json:"hero_image"`
	StoryBadge     string                 `json:"story_badge"`
	StoryTitle     string                 `json:"story_title"`
	StoryText      string                 `json:"story_text"`
	StoryQuote     string                 `json:"story_quote"`
	StoryImage     string                 `json:"story_image"`
	PillarsTitle   string                 `json:"pillars_title"`
	Pillars        []EditorialPillar      `json:"pillars"`
	Testimonials   []EditorialTestimonial `json:"testimonials"`
	FAQs           []EditorialFAQ         `json:"faqs"`
	AtelierAddress string                 `json:"atelier_address"`
	AtelierHours   string                 `json:"atelier_hours"`
	Instagram      string                 `json:"instagram"`
}

// EditorialPillar represents a single craft/quality guarantee card.
type EditorialPillar struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

// EditorialTestimonial represents authentic customer social proof.
type EditorialTestimonial struct {
	Customer string `json:"customer"`
	Location string `json:"location"`
	Quote    string `json:"quote"`
	Item     string `json:"item"`
}

// EditorialFAQ represents an accordion item.
type EditorialFAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

var (
	editorialOnce sync.Once
	editorialMap  map[string]NicheEditorial
)

func loadEditorialContent() {
	editorialOnce.Do(func() {
		data, err := instructions.FS.ReadFile("editorial.json")
		if err == nil {
			_ = json.Unmarshal(data, &editorialMap)
		}
		if editorialMap == nil {
			editorialMap = map[string]NicheEditorial{}
		}
	})
}

// GetNicheEditorialContent derives editorial storytelling content from instruction files.
func GetNicheEditorialContent(brandName, prompt string) NicheEditorial {
	loadEditorialContent()

	haystack := strings.ToLower(brandName + " " + prompt)

	var key string
	if strings.Contains(haystack, "seramik") || strings.Contains(haystack, "çömlek") ||
		strings.Contains(haystack, "fincan") || strings.Contains(haystack, "tabak") ||
		strings.Contains(haystack, "vazo") || strings.Contains(haystack, "kupa") ||
		strings.Contains(haystack, "clay") || strings.Contains(haystack, "ceramic") {
		key = "ceramic"
	} else if strings.Contains(haystack, "deri") || strings.Contains(haystack, "cüzdan") ||
		strings.Contains(haystack, "çanta") || strings.Contains(haystack, "kemer") ||
		strings.Contains(haystack, "saraç") || strings.Contains(haystack, "leather") {
		key = "leather"
	} else {
		key = "default"
	}

	content, ok := editorialMap[key]
	if !ok {
		content = editorialMap["default"]
	}

	return content
}

// MergeIntoSettings populates editorial fields into the store_settings data map.
func (e NicheEditorial) MergeIntoSettings(settings map[string]any) {
	settings["hero_badge"] = e.HeroBadge
	settings["hero_title"] = e.HeroTitle
	settings["hero_subtitle"] = e.HeroSubtitle
	settings["hero_image"] = e.HeroImage
	settings["story_badge"] = e.StoryBadge
	settings["story_title"] = e.StoryTitle
	settings["story_text"] = e.StoryText
	settings["story_quote"] = e.StoryQuote
	settings["story_image"] = e.StoryImage
	settings["pillars_title"] = e.PillarsTitle
	settings["atelier_address"] = e.AtelierAddress
	settings["atelier_hours"] = e.AtelierHours
	settings["instagram"] = e.Instagram

	if pJSON, err := json.Marshal(e.Pillars); err == nil {
		settings["pillars_json"] = string(pJSON)
	}
	if tJSON, err := json.Marshal(e.Testimonials); err == nil {
		settings["testimonials_json"] = string(tJSON)
	}
	if fJSON, err := json.Marshal(e.FAQs); err == nil {
		settings["faqs_json"] = string(fJSON)
	}
}
