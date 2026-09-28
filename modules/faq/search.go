package faq

import (
	"strings"
	"unicode"
)

func Search(items []Question, query string) []Question {
	query = normalizeSearch(query)
	if len(query) > 256 {
		query = query[:256]
	}
	result := make([]Question, 0, len(items))
	for _, item := range items {
		if item.Status != "published" {
			continue
		}
		if query == "" || strings.Contains(normalizeSearch(item.Category+" "+item.QuestionText+" "+item.Answer), query) {
			result = append(result, item)
			if len(result) == 200 {
				break
			}
		}
	}
	return result
}

func normalizeSearch(value string) string {
	var out strings.Builder
	space := true
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsSpace(r) {
			if !space {
				out.WriteByte(' ')
				space = true
			}
			continue
		}
		out.WriteRune(r)
		space = false
	}
	return strings.TrimSpace(out.String())
}
