package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"io"
	"strings"

	"github.com/a-h/templ"

	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

// RichTextHTML renders the supported Quill Delta subset. Plain text and
// legacy records remain readable, while unknown embeds are ignored safely.
// The markup comes from the components in richtext.templ.
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
	paragraphs := make([]string, 0)
	for _, paragraph := range strings.Split(value, "\n") {
		if paragraph = strings.TrimSpace(paragraph); paragraph != "" {
			paragraphs = append(paragraphs, paragraph)
		}
	}
	return renderString(richPlainParagraphs(paragraphs))
}

type deltaOp struct {
	Insert interface{}            `json:"insert"`
	Attrs  map[string]interface{} `json:"attributes"`
}

// deltaList is the list currently being filled; consecutive list lines of
// the same kind share one <ul>/<ol>.
type deltaList struct {
	ordered bool
	items   []templ.Component
}

// deltaRenderer turns Delta ops into block components. Inline runs collect in
// line until a newline flushes them into a block.
type deltaRenderer struct {
	blocks []templ.Component
	line   []templ.Component
	list   *deltaList
}

func renderDelta(ops []deltaOp) string {
	r := &deltaRenderer{}
	for _, op := range ops {
		switch ins := op.Insert.(type) {
		case string:
			if link, ok := op.Attrs["link"].(string); ok && strings.TrimSpace(ins) == strings.TrimSpace(link) {
				if source, valid := safeVideoURL(link); valid {
					r.flush(op.Attrs)
					r.block(videoEmbed(source, "Embedded video"))
					continue
				}
			}
			parts := strings.Split(ins, "\n")
			for i, part := range parts {
				r.line = append(r.line, formatDeltaText(part, op.Attrs))
				if i < len(parts)-1 {
					r.flush(op.Attrs)
				}
			}
		case map[string]interface{}:
			// Embedded objects like {"image": "url"}
			if imageSrc, ok := ins["image"].(string); ok {
				if src, safe := richImageSource(imageSrc); safe {
					r.closeList()
					r.line = append(r.line, richImage(src))
				}
			}
			if videoSrc, ok := ins["video"].(string); ok {
				if source, valid := safeVideoURL(videoSrc); valid {
					r.flush(nil)
					r.block(videoEmbed(source, "Embedded video"))
				}
			}
		}
	}
	r.flush(nil)
	r.closeList()
	return renderString(richBlocks(r.blocks))
}

func (r *deltaRenderer) closeList() {
	if r.list != nil {
		r.blocks = append(r.blocks, richList(r.list.ordered, r.list.items))
		r.list = nil
	}
}

// block appends a block-level component, closing any open list first so the
// output never nests blocks inside a list.
func (r *deltaRenderer) block(c templ.Component) {
	r.closeList()
	r.blocks = append(r.blocks, c)
}

// flush ends the current line, turning it into a block according to the
// line attributes Quill puts on the newline.
func (r *deltaRenderer) flush(attrs map[string]interface{}) {
	inline := richInline(r.line)
	r.line = nil
	text := renderString(inline)
	if source, ok := safeVideoURL(html.UnescapeString(strings.TrimSpace(text))); ok {
		r.block(videoEmbed(source, "Embedded video"))
		return
	}
	if text == "" && (attrs == nil || attrs["list"] == nil) {
		return
	}
	if attrs != nil {
		if listType, ok := attrs["list"].(string); ok {
			ordered := listType == "ordered"
			if r.list == nil || r.list.ordered != ordered {
				r.closeList()
				r.list = &deltaList{ordered: ordered}
			}
			r.list.items = append(r.list.items, inline)
			return
		}
		r.closeList()
		if header, ok := attrs["header"].(float64); ok && header >= 1 && header <= 4 {
			r.block(richHeading(int(header), inline))
			return
		}
		if _, ok := attrs["blockquote"]; ok {
			r.block(richBlockquote(inline))
			return
		}
		if _, ok := attrs["code-block"]; ok {
			language, _ := attrs["language"].(string)
			if language == "" {
				language, _ = attrs["code-language"].(string)
			}
			r.block(codeBlock(html.UnescapeString(text), language))
			return
		}
	} else {
		r.closeList()
	}
	if text != "" {
		r.block(richParagraph(inline))
	}
}

// codeBlock renders the design-system code block, falling back to a plain
// escaped <pre> if highlighting fails.
func codeBlock(code, language string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var highlighted bytes.Buffer
		if err := designsystem.CodeBlock(designsystem.CodeBlockProps{Code: code, Language: language}).Render(ctx, &highlighted); err != nil {
			return richCodeFallback(code).Render(ctx, w)
		}
		_, err := highlighted.WriteTo(w)
		return err
	})
}

// richImageSource allows http(s), local media and raster data URLs only.
func richImageSource(raw string) (string, bool) {
	src := strings.TrimSpace(raw)
	lower := strings.ToLower(src)
	for _, prefix := range []string{"https://", "http://", "/media/", "data:image/jpeg;", "data:image/png;", "data:image/gif;", "data:image/webp;"} {
		if strings.HasPrefix(lower, prefix) {
			return src, true
		}
	}
	return "", false
}

// richLinkHref allows http(s), mailto, same-origin paths and fragments.
func richLinkHref(raw string) (string, bool) {
	link := strings.TrimSpace(raw)
	lower := strings.ToLower(link)
	ok := strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "mailto:") || (strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "//")) || strings.HasPrefix(link, "#")
	return link, ok
}

// formatDeltaText wraps text in its inline formats, innermost first: link,
// bold, italic, underline, strike, code, then allowlisted inline styles.
func formatDeltaText(text string, attrs map[string]interface{}) templ.Component {
	c := richText(text)
	if attrs == nil {
		return c
	}
	if link, ok := attrs["link"].(string); ok {
		if href, safe := richLinkHref(link); safe {
			c = richLink(href, c)
		}
	}
	for _, format := range []struct {
		key  string
		wrap func(templ.Component) templ.Component
	}{
		{"bold", richStrong},
		{"italic", richEm},
		{"underline", richUnderline},
		{"strike", richStrike},
		{"code", richCode},
	} {
		if _, ok := attrs[format.key]; ok {
			c = format.wrap(c)
		}
	}
	for _, style := range richInlineStyles(attrs) {
		c = richStyled(style, c)
	}
	return c
}

// richInlineStyles returns the inline declarations to apply, innermost
// first. Only fixed sizes, fixed font stacks and hex colours are allowed.
func richInlineStyles(attrs map[string]interface{}) []templ.SafeCSS {
	var styles []templ.SafeCSS
	if size, ok := attrs["size"].(string); ok {
		sizes := map[string]string{"small": ".75em", "large": "1.5em", "huge": "2.5em"}
		if cssSize, valid := sizes[size]; valid {
			styles = append(styles, templ.SafeCSS("font-size:"+cssSize))
		}
	}
	if font, ok := attrs["font"].(string); ok {
		fonts := map[string]string{"serif": "Georgia,Times New Roman,serif", "monospace": "Monaco,Courier New,monospace"}
		if cssFont, valid := fonts[font]; valid {
			styles = append(styles, templ.SafeCSS("font-family:"+cssFont))
		}
	}
	if color, ok := attrs["color"].(string); ok && safeHexColor(color) {
		styles = append(styles, templ.SafeCSS("color:"+color))
	}
	if background, ok := attrs["background"].(string); ok && safeHexColor(background) {
		styles = append(styles, templ.SafeCSS("background-color:"+background))
	}
	return styles
}

func safeHexColor(value string) bool {
	if (len(value) != 4 && len(value) != 7) || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// renderString renders a component that cannot fail (or whose failure is
// best rendered as nothing) to a string.
func renderString(c templ.Component) string {
	var out strings.Builder
	if err := c.Render(context.Background(), &out); err != nil {
		return ""
	}
	return out.String()
}
