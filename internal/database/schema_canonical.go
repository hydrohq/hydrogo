package database

import (
	"regexp"
	"sort"
	"strings"
)

var (
	spaceRun    = regexp.MustCompile(`\s+`)
	spaceAround = regexp.MustCompile(`\s*([(),])\s*`)
)

// NormalizeSchemaSQL makes two spellings of the same definition compare equal:
// runs of white space collapse and the space around parentheses and commas goes,
// which is all that ALTER TABLE ... ADD COLUMN changes in the stored text.
func NormalizeSchemaSQL(sqlText string) string {
	text := spaceRun.ReplaceAllString(strings.TrimSpace(sqlText), " ")
	return spaceAround.ReplaceAllString(text, "$1")
}

// CanonicalSchemaSQL is NormalizeSchemaSQL plus, for a table, an order-free body:
// a baseline puts a module's columns in one place while the chain appends them
// as migrations run, and nothing in the product addresses columns by position.
func CanonicalSchemaSQL(kind, sqlText string) string {
	if kind != "table" {
		return NormalizeSchemaSQL(sqlText)
	}
	prefix, pieces, suffix, ok := SplitTableBody(sqlText)
	if !ok {
		return NormalizeSchemaSQL(sqlText)
	}
	normalized := make([]string, len(pieces))
	for index, piece := range pieces {
		normalized[index] = NormalizeSchemaSQL(piece)
	}
	sort.Strings(normalized)
	return NormalizeSchemaSQL(prefix) + "(" + strings.Join(normalized, ",") + ")" + NormalizeSchemaSQL(suffix)
}

// SplitTableBody splits "CREATE TABLE name (a, b, c) options" into the text before
// the body, the top-level comma separated pieces of the body, and the text after
// it. ok is false when there is no parenthesised body.
func SplitTableBody(sqlText string) (prefix string, pieces []string, suffix string, ok bool) {
	open := strings.Index(sqlText, "(")
	if open < 0 {
		return "", nil, "", false
	}
	depth, quote := 0, byte(0)
	closeAt := -1
scan:
	for index := open; index < len(sqlText); index++ {
		character := sqlText[index]
		switch {
		case quote != 0:
			if character == quote {
				quote = 0
			}
		case character == '\'' || character == '"' || character == '`':
			quote = character
		case character == '(':
			depth++
		case character == ')':
			depth--
			if depth == 0 {
				closeAt = index
				break scan
			}
		}
	}
	if closeAt < 0 {
		return "", nil, "", false
	}
	body := sqlText[open+1 : closeAt]
	depth, quote = 0, 0
	start := 0
	for index := 0; index < len(body); index++ {
		character := body[index]
		switch {
		case quote != 0:
			if character == quote {
				quote = 0
			}
		case character == '\'' || character == '"' || character == '`':
			quote = character
		case character == '(':
			depth++
		case character == ')':
			depth--
		case character == ',' && depth == 0:
			pieces = append(pieces, strings.TrimSpace(body[start:index]))
			start = index + 1
		}
	}
	pieces = append(pieces, strings.TrimSpace(body[start:]))
	return sqlText[:open], pieces, sqlText[closeAt+1:], true
}
