package search

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/hydrohq/hydrogo/internal/database"
)

func tokenize(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsPunct(r) })
}

func snippet(body string, terms []string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len([]rune(body)) > 180 {
		return string([]rune(body)[:180]) + "…"
	}
	return body
}

func sqlPlaceholder(dialect database.Dialect, position int) string {
	if dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
