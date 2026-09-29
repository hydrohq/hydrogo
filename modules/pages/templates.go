package pages

import (
	"encoding/json"
	"errors"
	"regexp"
)

var ErrInvalidLayoutTemplate = errors.New("invalid page layout template")

type LayoutTemplate struct {
	ID, Name, Description string
	Blocks                []Block
}

var layoutTemplateID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func BuiltinLayoutTemplates() []LayoutTemplate {
	block := func(kind, payload string) Block { return Block{Type: kind, Payload: json.RawMessage(payload)} }
	return []LayoutTemplate{
		{ID: "blank", Name: "Blank canvas", Description: "Start with an empty page."},
		{ID: "service_showcase", Name: "Service showcase", Description: "Introduce a service and its benefits.", Blocks: []Block{
			block("hero", `{"title":"Your service","subtitle":"A clear introduction to what you offer.","button_text":"Get in touch","button_url":"/contact"}`),
			block("feature_list", `{"items":[{"title":"Benefit one","text":"Describe a practical customer benefit."},{"title":"Benefit two","text":"Explain what makes this service useful."}]}`),
		}},
		{ID: "landing", Name: "Landing page", Description: "A focused campaign page with proof and a call to action.", Blocks: []Block{
			block("hero", `{"title":"A better way to work","subtitle":"Share the promise of your campaign.","button_text":"Get started","button_url":"/contact"}`),
			block("trust_bar", `{"items":[{"title":"Trusted","subtitle":"Built for your team"},{"title":"Supported","subtitle":"Here when you need us"}]}`),
			block("action_card", `{"title":"Ready to take the next step?","text":"Tell visitors how to get started.","button_text":"Contact us","button_url":"/contact"}`),
		}},
		{ID: "case_study", Name: "Case study", Description: "Describe a challenge, outcome, and customer proof.", Blocks: []Block{
			block("hero", `{"title":"Customer success story","subtitle":"Introduce the customer and the result."}`),
			block("callout", `{"title":"The outcome","text":"Summarize the measurable change for the customer.","tone":"success"}`),
			block("testimonials", `{"items":[{"name":"Customer name","quote":"Share a short customer quote."}]}`),
		}},
	}
}

func ValidateLayoutTemplates(templates []LayoutTemplate) error {
	seen := make(map[string]struct{}, len(templates))
	for _, item := range templates {
		if !layoutTemplateID.MatchString(item.ID) || item.Name == "" || len(item.Name) > 100 || len(item.Description) > 500 || len(item.Blocks) > MaxBlocks {
			return ErrInvalidLayoutTemplate
		}
		if _, exists := seen[item.ID]; exists {
			return ErrInvalidLayoutTemplate
		}
		seen[item.ID] = struct{}{}
		if len(item.Blocks) > 0 && ValidateBlocks(item.Blocks) != nil {
			return ErrInvalidLayoutTemplate
		}
	}
	return nil
}

func ResolveLayoutTemplate(id string, custom []LayoutTemplate) ([]Block, error) {
	registry := append(BuiltinLayoutTemplates(), custom...)
	if ValidateLayoutTemplates(registry) != nil {
		return nil, ErrInvalidLayoutTemplate
	}
	for _, item := range registry {
		if item.ID == id {
			return append([]Block(nil), item.Blocks...), nil
		}
	}
	return nil, ErrInvalidLayoutTemplate
}
