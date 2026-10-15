// Package richtext holds the server-side helpers for stored rich text: the
// HTML sanitizer applied on save and a plain-text projection of the Quill
// Delta format used for search and discovery documents. Rendering rich text
// to HTML is the frontend's job.
package richtext

import (
	"encoding/json"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

type Sanitizer struct {
	policy *bluemonday.Policy
}

func NewSanitizer() *Sanitizer {
	p := bluemonday.UGCPolicy()
	p.AllowStandardURLs()
	p.RequireParseableURLs(true)
	p.RequireNoFollowOnLinks(false)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return &Sanitizer{policy: p}
}

func (s *Sanitizer) Sanitize(dirtyHTML string) string {
	if s.policy == nil {
		s.policy = bluemonday.UGCPolicy()
	}
	return s.policy.Sanitize(dirtyHTML)
}

// PlainText returns the text of a stored rich-text value. Quill Delta
// documents ({"ops":[...]} or {"document":[...]}) are flattened to their text
// inserts; anything else is returned trimmed as-is.
func PlainText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var envelope struct {
		Document json.RawMessage `json:"document"`
		Ops      json.RawMessage `json:"ops"`
	}
	if json.Unmarshal([]byte(value), &envelope) != nil {
		return value
	}
	raw := envelope.Document
	if len(raw) == 0 {
		raw = envelope.Ops
	}
	var ops []struct {
		Insert any `json:"insert"`
	}
	if json.Unmarshal(raw, &ops) != nil || len(ops) == 0 {
		return value
	}
	var b strings.Builder
	for _, op := range ops {
		if text, ok := op.Insert.(string); ok {
			b.WriteString(text)
		}
	}
	return strings.TrimSpace(b.String())
}
