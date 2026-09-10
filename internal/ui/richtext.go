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
	currentList := "" // "ul" or "ol"

	closeList := func() {
		if currentList != "" {
			out.WriteString("</" + currentList + ">")
			currentList = ""
		}
	}

	flush := func(attrs map[string]interface{}) {
		text := line.String()
		line.Reset()
		if text == "" && (attrs == nil || attrs["list"] == nil) {
			return
		}
		if attrs != nil {
			if listType, ok := attrs["list"].(string); ok {
				targetList := "ul"
				if listType == "ordered" {
					targetList = "ol"
				}
				if currentList != targetList {
					closeList()
					out.WriteString("<" + targetList + ">")
					currentList = targetList
				}
				out.WriteString("<li>" + text + "</li>")
				return
			}
			closeList()

			if header, ok := attrs["header"].(float64); ok && header >= 1 && header <= 4 {
				tag := []string{"", "h1", "h2", "h3", "h4"}[int(header)]
				out.WriteString("<" + tag + ">" + text + "</" + tag + ">")
				return
			}
			if _, ok := attrs["blockquote"]; ok {
				out.WriteString("<blockquote>" + text + "</blockquote>")
				return
			}
			if _, ok := attrs["code-block"]; ok {
				out.WriteString("<pre><code>" + text + "</code></pre>")
				return
			}
		} else {
			closeList()
		}
		if text != "" {
			out.WriteString("<p>" + text + "</p>")
		}
	}

	for _, op := range ops {
		switch ins := op.Insert.(type) {
		case string:
			parts := strings.Split(ins, "\n")
			for i, part := range parts {
				line.WriteString(formatDeltaText(part, op.Attrs))
				if i < len(parts)-1 {
					flush(op.Attrs)
				}
			}
		case map[string]interface{}:
			// Embedded objects like {"image": "url"}
			if imageSrc, ok := ins["image"].(string); ok {
				imageSrc = strings.TrimSpace(imageSrc)
				lowerSrc := strings.ToLower(imageSrc)
				isSafeURL := strings.HasPrefix(lowerSrc, "https://") || strings.HasPrefix(lowerSrc, "http://") || strings.HasPrefix(lowerSrc, "/media/")
				isSafeData := strings.HasPrefix(lowerSrc, "data:image/jpeg;") || strings.HasPrefix(lowerSrc, "data:image/png;") || strings.HasPrefix(lowerSrc, "data:image/gif;") || strings.HasPrefix(lowerSrc, "data:image/webp;")
				if isSafeURL || isSafeData {
					closeList()
					line.WriteString(`<img src="` + html.EscapeString(imageSrc) + `" alt="Görsel" loading="lazy" class="richtext-image" />`)
				}
			}
		}
	}
	flush(nil)
	closeList()
	return out.String()
}

func formatDeltaText(text string, attrs map[string]interface{}) string {
	text = html.EscapeString(text)
	if attrs == nil {
		return text
	}
	if link, ok := attrs["link"].(string); ok {
		link = strings.TrimSpace(link)
		lower := strings.ToLower(link)
		if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "mailto:") || (strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "//")) || strings.HasPrefix(link, "#") {
			text = `<a href="` + html.EscapeString(link) + `" target="_blank" rel="noopener noreferrer">` + text + "</a>"
		}
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
	if _, ok := attrs["code"]; ok {
		text = "<code>" + text + "</code>"
	}
	return text
}
