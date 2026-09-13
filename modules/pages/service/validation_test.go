package service

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestValidatePageRejectsOversizedFields(t *testing.T) {
	base := pages.Page{ID: "page-1", Locale: "tr", Slug: "hakkinda", Title: "Hakkında", Status: pages.StatusDraft}
	for name, mutate := range map[string]func(*pages.Page){
		"id":    func(p *pages.Page) { p.ID = strings.Repeat("x", 129) },
		"slug":  func(p *pages.Page) { p.Slug = strings.Repeat("x", 257) },
		"title": func(p *pages.Page) { p.Title = strings.Repeat("x", 257) },
		"body":  func(p *pages.Page) { p.Body = strings.Repeat("x", 1<<20+1) },
	} {
		t.Run(name, func(t *testing.T) {
			page := base
			mutate(&page)
			if err := validatePage(&page); err == nil {
				t.Fatal("expected oversized page field to be rejected")
			}
		})
	}
}
