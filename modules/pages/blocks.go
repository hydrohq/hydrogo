package pages

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	MaxBlocks       = 64
	MaxBlockPayload = 64 << 10
)

type Block struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

var ErrInvalidBlocks = errors.New("invalid page blocks")

func ValidateBlocks(blocks []Block) error {
	if len(blocks) > MaxBlocks {
		return fmt.Errorf("%w: too many blocks", ErrInvalidBlocks)
	}
	for _, block := range blocks {
		if strings.TrimSpace(block.Type) == "" || len(block.Payload) == 0 || len(block.Payload) > MaxBlockPayload {
			return ErrInvalidBlocks
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(block.Payload, &object); err != nil || object == nil {
			return ErrInvalidBlocks
		}
		if err := validatePreset(object, "bg_color", "default", "muted", "primary_tint", "dark"); err != nil {
			return err
		}
		if err := validatePreset(object, "padding_y", "normal", "compact", "spacious"); err != nil {
			return err
		}
		switch block.Type {
		case "hero", "cta":
			if err := requireString(object, "title"); err != nil {
				return err
			}
			if raw, ok := object["media_asset_id"]; ok {
				var assetID string
				if json.Unmarshal(raw, &assetID) != nil || !safeAssetID(assetID) {
					return ErrInvalidBlocks
				}
			}
		case "pricing_table":
			var items []struct {
				Name       string   `json:"name"`
				PriceCents int64    `json:"price_cents"`
				Currency   string   `json:"currency"`
				Features   []string `json:"features"`
			}
			raw, ok := object["items"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 32 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if strings.TrimSpace(item.Name) == "" || item.PriceCents < 0 || len(strings.TrimSpace(item.Currency)) != 3 || len(item.Features) > 32 {
					return ErrInvalidBlocks
				}
			}
		case "testimonials":
			var items []struct {
				Name  string `json:"name"`
				Quote string `json:"quote"`
			}
			raw, ok := object["items"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 32 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Quote) == "" {
					return ErrInvalidBlocks
				}
			}
		case "faq":
			var items []struct {
				Question string `json:"question"`
				Answer   string `json:"answer"`
			}
			raw, ok := object["items"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 32 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
					return ErrInvalidBlocks
				}
			}
		case "before_after":
			if requireAsset(object, "before_image_asset_id") != nil || requireAsset(object, "after_image_asset_id") != nil {
				return ErrInvalidBlocks
			}
			if raw, ok := object["initial_slider_position"]; ok {
				var position int
				if json.Unmarshal(raw, &position) != nil || position < 0 || position > 100 {
					return ErrInvalidBlocks
				}
			}
		case "stats":
			var items []struct {
				Number float64 `json:"number_value"`
				Prefix string  `json:"prefix"`
				Suffix string  `json:"suffix"`
				Label  string  `json:"label"`
				Badge  string  `json:"icon_badge"`
			}
			raw, ok := object["stats"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 32 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if math.IsNaN(item.Number) || math.IsInf(item.Number, 0) || item.Number < 0 || item.Number > 1e12 || strings.TrimSpace(item.Label) == "" || len(item.Prefix) > 32 || len(item.Suffix) > 32 || len(item.Badge) > 32 {
					return ErrInvalidBlocks
				}
			}
		case "contact_cta":
			phone, whatsapp := rawString(object, "phone"), rawString(object, "whatsapp")
			if phone == "" && whatsapp == "" || phone != "" && !validPhoneNumber(phone) || whatsapp != "" && !validWhatsAppNumber(whatsapp) {
				return ErrInvalidBlocks
			}
		case "callout":
			tone := rawString(object, "tone")
			if title := rawString(object, "title"); title == "" || len(title) > 200 || len(rawString(object, "text")) > 4000 || tone != "" && tone != "info" && tone != "warning" && tone != "success" && tone != "tip" {
				return ErrInvalidBlocks
			}
		case "action_card":
			if requireString(object, "title") != nil || len(rawString(object, "text")) > 4000 || len(rawString(object, "button_text")) > 100 || !optionalSafeURL(rawString(object, "button_url")) {
				return ErrInvalidBlocks
			}
		case "carousel":
			var items []struct {
				Src     string `json:"src"`
				Alt     string `json:"alt"`
				Caption string `json:"caption"`
			}
			raw, ok := object["slides"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 20 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if !safeBlockURL(item.Src) || strings.TrimSpace(item.Alt) == "" || len(item.Alt) > 500 || len(item.Caption) > 500 {
					return ErrInvalidBlocks
				}
			}
		case "columns":
			var payload struct {
				Count   int `json:"count"`
				Columns []struct {
					Title     string `json:"title"`
					Text      string `json:"text"`
					ButtonURL string `json:"button_url"`
				} `json:"columns"`
			}
			if json.Unmarshal(block.Payload, &payload) != nil || payload.Count != 2 && payload.Count != 3 || len(payload.Columns) != payload.Count {
				return ErrInvalidBlocks
			}
			for _, column := range payload.Columns {
				if strings.TrimSpace(column.Title) == "" || len(column.Title) > 200 || len(column.Text) > 4000 || !optionalSafeURL(column.ButtonURL) {
					return ErrInvalidBlocks
				}
			}
		case "feature_list":
			var payload struct {
				Items []struct {
					Title string `json:"title"`
					Text  string `json:"text"`
				} `json:"items"`
			}
			if json.Unmarshal(block.Payload, &payload) != nil || len(payload.Items) == 0 || len(payload.Items) > 32 {
				return ErrInvalidBlocks
			}
			for _, item := range payload.Items {
				if strings.TrimSpace(item.Title) == "" || len(item.Title) > 200 || len(item.Text) > 2000 {
					return ErrInvalidBlocks
				}
			}
		case "video_hero":
			if requireString(object, "title") != nil || len(rawString(object, "subtitle")) > 2000 || requireSafeStringURL(object, "video_url") != nil || !optionalSafeURL(rawString(object, "poster_url")) || !optionalSafeURL(rawString(object, "button_url")) {
				return ErrInvalidBlocks
			}
		case "trust_bar":
			var payload struct {
				Items []struct {
					Title    string `json:"title"`
					Subtitle string `json:"subtitle"`
				} `json:"items"`
			}
			if json.Unmarshal(block.Payload, &payload) != nil || len(payload.Items) == 0 || len(payload.Items) > 16 {
				return ErrInvalidBlocks
			}
			for _, item := range payload.Items {
				if strings.TrimSpace(item.Title) == "" || len(item.Title) > 200 || len(item.Subtitle) > 500 {
					return ErrInvalidBlocks
				}
			}
		case "figure":
			if requireSafeStringURL(object, "src") != nil || strings.TrimSpace(rawString(object, "alt")) == "" || len(rawString(object, "alt")) > 500 || len(rawString(object, "caption")) > 1000 || len(rawString(object, "credit")) > 200 || !optionalSafeURL(rawString(object, "credit_url")) {
				return ErrInvalidBlocks
			}
		case "business_status":
			var open bool
			raw, ok := object["open"]
			if !ok || json.Unmarshal(raw, &open) != nil || len(rawString(object, "open_label")) > 100 || len(rawString(object, "closed_label")) > 100 || len(rawString(object, "hours")) > 256 {
				return ErrInvalidBlocks
			}
		case "countdown":
			var payload struct {
				Target   string `json:"target_datetime"`
				Zone     string `json:"timezone"`
				Headline string `json:"expired_headline"`
				URL      string `json:"expired_cta_url"`
			}
			if json.Unmarshal(block.Payload, &payload) != nil || strings.TrimSpace(payload.Headline) == "" || len(payload.Headline) > 200 || len(payload.Zone) > 64 || len(payload.URL) > 2048 || payload.URL != "" && !safeBlockURL(payload.URL) {
				return ErrInvalidBlocks
			}
			if _, err := time.Parse(time.RFC3339, payload.Target); err != nil {
				return ErrInvalidBlocks
			}
			if payload.Zone != "" {
				if _, err := time.LoadLocation(payload.Zone); err != nil {
					return ErrInvalidBlocks
				}
			}
		case "video":
			if requireSafeStringURL(object, "src") != nil {
				return ErrInvalidBlocks
			}
			if raw, ok := object["type"]; ok {
				var typ string
				if json.Unmarshal(raw, &typ) != nil || typ != "video/mp4" && typ != "video/webm" {
					return ErrInvalidBlocks
				}
			}
			if raw, ok := object["tracks"]; ok {
				var tracks []struct {
					Src      string `json:"src"`
					Language string `json:"language"`
					Label    string `json:"label"`
				}
				if json.Unmarshal(raw, &tracks) != nil || len(tracks) > 16 {
					return ErrInvalidBlocks
				}
				for _, track := range tracks {
					if !safeBlockURL(track.Src) || track.Language == "" || track.Label == "" {
						return ErrInvalidBlocks
					}
				}
			}
		case "masonry":
			var items []struct {
				Image string `json:"image"`
				Alt   string `json:"alt"`
				Link  string `json:"link"`
			}
			raw, ok := object["items"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 64 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if !safeBlockURL(item.Image) || strings.TrimSpace(item.Alt) == "" || item.Link != "" && !safeBlockURL(item.Link) {
					return ErrInvalidBlocks
				}
			}
		case "hotspots":
			if requireSafeStringURL(object, "image") != nil {
				return ErrInvalidBlocks
			}
			var points []struct {
				X     float64 `json:"x"`
				Y     float64 `json:"y"`
				Title string  `json:"title"`
				Link  string  `json:"link"`
			}
			raw, ok := object["points"]
			if !ok || json.Unmarshal(raw, &points) != nil || len(points) == 0 || len(points) > 64 {
				return ErrInvalidBlocks
			}
			for _, point := range points {
				if math.IsNaN(point.X) || math.IsNaN(point.Y) || math.IsInf(point.X, 0) || math.IsInf(point.Y, 0) || point.X < 0 || point.X > 100 || point.Y < 0 || point.Y > 100 || strings.TrimSpace(point.Title) == "" || point.Link != "" && !safeBlockURL(point.Link) {
					return ErrInvalidBlocks
				}
			}
		default:
			return fmt.Errorf("%w: unsupported block type %q", ErrInvalidBlocks, block.Type)
		}
		if raw, ok := object["button_url"]; ok {
			var value string
			if json.Unmarshal(raw, &value) != nil || !safeBlockURL(value) {
				return ErrInvalidBlocks
			}
		}
	}
	return nil
}

func validatePreset(object map[string]json.RawMessage, key string, allowed ...string) error {
	raw, ok := object[key]
	if !ok {
		return nil
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ErrInvalidBlocks
	}
	for _, option := range allowed {
		if value == option {
			return nil
		}
	}
	return ErrInvalidBlocks
}

// ValidateBlocksForRender retains old unknown block payloads for safe fallback
// display. Write paths continue to use ValidateBlocks and reject unknown types.
func ValidateBlocksForRender(blocks []Block) error {
	if len(blocks) > MaxBlocks {
		return ErrInvalidBlocks
	}
	for _, block := range blocks {
		if strings.TrimSpace(block.Type) == "" || len(block.Payload) == 0 || len(block.Payload) > MaxBlockPayload {
			return ErrInvalidBlocks
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(block.Payload, &object); err != nil || object == nil {
			return ErrInvalidBlocks
		}
		if !knownBlockType(block.Type) {
			continue
		}
		if err := ValidateBlocks([]Block{block}); err != nil {
			return err
		}
	}
	return nil
}

func knownBlockType(blockType string) bool {
	switch blockType {
	case "hero", "cta", "pricing_table", "testimonials", "faq", "before_after", "stats", "contact_cta", "video", "masonry", "hotspots", "callout", "action_card", "carousel", "columns", "feature_list", "video_hero", "trust_bar", "figure", "business_status":
		return true
	}
	return false
}

func optionalSafeURL(value string) bool { return value == "" || safeBlockURL(value) }

func requireAsset(object map[string]json.RawMessage, key string) error {
	var id string
	raw, ok := object[key]
	if !ok || json.Unmarshal(raw, &id) != nil || !safeAssetID(id) {
		return ErrInvalidBlocks
	}
	return nil
}
func rawString(object map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(object[key], &value)
	return strings.TrimSpace(value)
}
func requireSafeStringURL(object map[string]json.RawMessage, key string) error {
	value := rawString(object, key)
	if value == "" || !safeBlockURL(value) {
		return ErrInvalidBlocks
	}
	return nil
}
func validPhoneNumber(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for i, r := range value {
		if r == '+' && i == 0 {
			continue
		}
		if r >= '0' && r <= '9' || r == ' ' || r == '(' || r == ')' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
	return len(digits) >= 7 && len(digits) <= 15
}
func validWhatsAppNumber(value string) bool {
	return validPhoneNumber(value) && !strings.HasPrefix(strings.TrimPrefix(strings.TrimSpace(value), "+"), "0")
}

func requireString(object map[string]json.RawMessage, key string) error {
	raw, ok := object[key]
	var value string
	if !ok || json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" || len(strings.TrimSpace(value)) > 512 {
		return ErrInvalidBlocks
	}
	return nil
}

func safeBlockURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || strings.EqualFold(u.Scheme, "javascript") || strings.EqualFold(u.Scheme, "data") {
		return false
	}
	return u.Scheme == "" || strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")
}

func safeAssetID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func EncodeBlocksForStore(blocks []Block) (string, error) {
	if err := ValidateBlocks(blocks); err != nil {
		return "", err
	}
	if blocks == nil {
		return "[]", nil
	}
	b, err := json.Marshal(blocks)
	return string(b), err
}

func DecodeBlocksForStore(value any) ([]Block, error) {
	if value == nil {
		return nil, nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, ErrInvalidBlocks
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var blocks []Block
	if err := json.Unmarshal(data, &blocks); err != nil || ValidateBlocksForRender(blocks) != nil {
		return nil, ErrInvalidBlocks
	}
	return blocks, nil
}
