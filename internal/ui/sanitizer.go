package ui

import (
	"github.com/microcosm-cc/bluemonday"
)

type HTMLSanitizer struct {
	policy *bluemonday.Policy
}

func NewHTMLSanitizer() *HTMLSanitizer {
	p := bluemonday.UGCPolicy()
	// Disallow scripts, flash, plugins, frames, and event handlers
	p.AllowStandardURLs()
	p.RequireParseableURLs(true)
	p.RequireNoFollowOnLinks(false)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return &HTMLSanitizer{policy: p}
}

func (s *HTMLSanitizer) Sanitize(dirtyHTML string) string {
	if s.policy == nil {
		s.policy = bluemonday.UGCPolicy()
	}
	return s.policy.Sanitize(dirtyHTML)
}
