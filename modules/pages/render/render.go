package render

import (
	"encoding/json"
	"errors"
	"html/template"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

var ErrInvalidBlock = errors.New("cannot render invalid block")

// Blocks returns HTML generated only from validated, allowlisted block data.
// The returned string is safe to pass to a trusted template component; it is
// never copied from a client-provided HTML field.
func Blocks(blocks []pages.Block) (string, error) {
	if err := pages.ValidateBlocks(blocks); err != nil {
		return "", ErrInvalidBlock
	}
	var out strings.Builder
	for _, block := range blocks {
		html, err := blockHTML(block)
		if err != nil {
			return "", err
		}
		out.WriteString(html)
	}
	return out.String(), nil
}

func blockHTML(block pages.Block) (string, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(block.Payload, &data); err != nil {
		return "", ErrInvalidBlock
	}
	text := func(key string) (string, error) {
		var value string
		if err := json.Unmarshal(data[key], &value); err != nil {
			return "", ErrInvalidBlock
		}
		return template.HTMLEscapeString(value), nil
	}
	switch block.Type {
	case "hero":
		title, err := text("title")
		if err != nil {
			return "", err
		}
		subtitle, _ := textOptional(data, "subtitle")
		return `<section data-block="hero"><h1>` + title + `</h1><p>` + subtitle + `</p></section>`, nil
	case "cta":
		title, err := text("title")
		if err != nil {
			return "", err
		}
		url, _ := textOptional(data, "button_url")
		label, _ := textOptional(data, "button_text")
		return `<section data-block="cta"><h2>` + title + `</h2><a href="` + url + `">` + label + `</a></section>`, nil
	case "faq":
		var items []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		if err := json.Unmarshal(data["items"], &items); err != nil {
			return "", ErrInvalidBlock
		}
		var out strings.Builder
		out.WriteString(`<section data-block="faq">`)
		for _, item := range items {
			out.WriteString(`<details><summary>` + template.HTMLEscapeString(item.Question) + `</summary><p>` + template.HTMLEscapeString(item.Answer) + `</p></details>`)
		}
		out.WriteString(`</section>`)
		return out.String(), nil
	case "pricing_table":
		var items []struct {
			Name       string   `json:"name"`
			PriceCents int64    `json:"price_cents"`
			Currency   string   `json:"currency"`
			Features   []string `json:"features"`
		}
		if err := json.Unmarshal(data["items"], &items); err != nil {
			return "", ErrInvalidBlock
		}
		var out strings.Builder
		out.WriteString(`<section data-block="pricing_table"><div class="pricing-grid">`)
		for _, item := range items {
			out.WriteString(`<article><h2>` + template.HTMLEscapeString(item.Name) + `</h2><p>` + template.HTMLEscapeString(strconv.FormatInt(item.PriceCents, 10)) + ` ` + template.HTMLEscapeString(item.Currency) + `</p><ul>`)
			for _, feature := range item.Features {
				out.WriteString(`<li>` + template.HTMLEscapeString(feature) + `</li>`)
			}
			out.WriteString(`</ul></article>`)
		}
		out.WriteString(`</div></section>`)
		return out.String(), nil
	case "testimonials":
		var items []struct {
			Name  string `json:"name"`
			Quote string `json:"quote"`
		}
		if err := json.Unmarshal(data["items"], &items); err != nil {
			return "", ErrInvalidBlock
		}
		var out strings.Builder
		out.WriteString(`<section data-block="testimonials">`)
		for _, item := range items {
			out.WriteString(`<blockquote><p>` + template.HTMLEscapeString(item.Quote) + `</p><cite>` + template.HTMLEscapeString(item.Name) + `</cite></blockquote>`)
		}
		out.WriteString(`</section>`)
		return out.String(), nil
	default:
		return "", ErrInvalidBlock
	}
}

func textOptional(data map[string]json.RawMessage, key string) (string, error) {
	raw, ok := data[key]
	if !ok {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", ErrInvalidBlock
	}
	return template.HTMLEscapeString(value), nil
}
