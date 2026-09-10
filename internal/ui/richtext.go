package ui

import (
	"encoding/json"
	"html"
	"strings"
)

// RichTextHTML renders the supported Quill Delta subset. Plain text and
// legacy records remain readable, while unknown embeds are ignored safely.
func RichTextHTML(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var envelope struct {
		Document json.RawMessage `json:"document"`
		Ops      []deltaOp       `json:"ops"`
	}
	if json.Unmarshal([]byte(value), &envelope) == nil {
		raw := envelope.Document
		if len(raw) == 0 && envelope.Ops != nil {
			b, _ := json.Marshal(envelope.Ops)
			raw = b
		}
		var ops []deltaOp
		if json.Unmarshal(raw, &ops) == nil && len(ops) > 0 {
			return renderDelta(ops)
		}
	}
	var out strings.Builder
	for _, paragraph := range strings.Split(value, "\n") {
		if paragraph = strings.TrimSpace(paragraph); paragraph != "" {
			out.WriteString("<p>")
			out.WriteString(html.EscapeString(paragraph))
			out.WriteString("</p>")
		}
	}
	return out.String()
}

type deltaOp struct {
	Insert interface{}            `json:"insert"`
	Attrs  map[string]interface{} `json:"attributes"`
}

func renderDelta(ops []deltaOp) string {
	var out strings.Builder
	var line strings.Builder
	flush := func(attrs map[string]interface{}) {
		text := line.String()
		line.Reset()
		if text == "" {
			return
		}
		if attrs != nil {
			if header, ok := attrs["header"].(float64); ok && header >= 1 && header <= 3 {
				tag := []string{"", "h1", "h2", "h3"}[int(header)]
				out.WriteString("<" + tag + ">" + text + "</" + tag + ">")
				return
			}
			if _, ok := attrs["blockquote"]; ok {
				out.WriteString("<blockquote>" + text + "</blockquote>")
				return
			}
		}
		out.WriteString("<p>" + text + "</p>")
	}
	for _, op := range ops {
		text, ok := op.Insert.(string)
		if !ok {
			continue
		}
		parts := strings.Split(text, "\n")
		for i, part := range parts {
			line.WriteString(formatDeltaText(part, op.Attrs))
			if i < len(parts)-1 {
				flush(op.Attrs)
			}
		}
	}
	flush(nil)
	return out.String()
}

func formatDeltaText(text string, attrs map[string]interface{}) string {
	text = html.EscapeString(text)
	if attrs == nil {
		return text
	}
	if link, ok := attrs["link"].(string); ok && (strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "mailto:")) {
		text = `<a href="` + html.EscapeString(link) + `" rel="noopener noreferrer">` + text + "</a>"
	}
	if _, ok := attrs["bold"]; ok {
		text = "<strong>" + text + "</strong>"
	}
	if _, ok := attrs["italic"]; ok {
		text = "<em>" + text + "</em>"
	}
	if _, ok := attrs["underline"]; ok {
		text = "<u>" + text + "</u>"
	}
	if _, ok := attrs["strike"]; ok {
		text = "<s>" + text + "</s>"
	}
	return text
}
