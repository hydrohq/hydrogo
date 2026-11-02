package sitegen

// ThemeConfig defines visual tokens for the generated boutique storefront.
type ThemeConfig struct {
	Preset       string `json:"preset"`
	FontPreset   string `json:"font_preset"`
	BGColor      string `json:"bg_color,omitempty"`
	TextColor    string `json:"text_color,omitempty"`
	AccentColor  string `json:"accent_color,omitempty"`
	BorderRadius string `json:"border_radius,omitempty"`
}

// LayoutSection defines a section and its props for store_settings.layout_json.
type LayoutSection struct {
	Type  string         `json:"type"`
	Props map[string]any `json:"props"`
}

// ProductItem represents a generated catalog product.
type ProductItem struct {
	Title          string  `json:"title"`
	Slug           string  `json:"slug"`
	Price          float64 `json:"price"`
	CompareAtPrice float64 `json:"compare_at_price,omitempty"`
	Category       string  `json:"category,omitempty"`
	Description    string  `json:"description,omitempty"`
	Badge          string  `json:"badge,omitempty"`
	ImageURL       string  `json:"image_url,omitempty"`
}

// SitePlan is the complete, validated output produced by the Site Generation Engine.
type SitePlan struct {
	Brief         string          `json:"brief"`
	Theme         ThemeConfig     `json:"theme"`
	Layout        []LayoutSection `json:"layout"`
	StoreSettings map[string]any  `json:"store_settings"`
	Products      []ProductItem   `json:"products"`
	PromptID      string          `json:"prompt_id,omitempty"`
	PromptVersion string          `json:"prompt_version,omitempty"`
	PromptHash    string          `json:"prompt_hash,omitempty"`
}
