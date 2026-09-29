package designsystem

import "strings"

type PublicErrorPageProps struct {
	Status                                              int
	Headline, Message, ImageURL, ActionLabel, ActionURL string
	Links                                               []PublicErrorLink
}

type PublicErrorLink struct{ Label, URL string }

func errorPageLinksText(links []PublicErrorLink) string {
	lines := make([]string, 0, len(links))
	for _, link := range links {
		lines = append(lines, link.Label+" | "+link.URL)
	}
	return strings.Join(lines, "\n")
}

type ErrorPageEditorProps struct {
	CSRF                  string
	NotFound, ServerError PublicErrorPageProps
}
