package ui

import (
	"html/template"
	"strconv"
	"strings"
)

// ButtonOptions describes a semantic action control. Label and attributes are
// escaped before being returned as template HTML.
type ButtonOptions struct {
	Label    string
	Type     string
	Class    string
	Name     string
	Value    string
	Disabled bool
}

// Button renders a button with a stable shared selector and safe attributes.
func Button(options ButtonOptions) template.HTML {
	typ := options.Type
	if typ == "" {
		typ = "button"
	}
	attrs := ` type="` + template.HTMLEscapeString(typ) + `" class="button`
	if className := strings.TrimSpace(options.Class); className != "" {
		attrs += " " + template.HTMLEscapeString(className)
	}
	attrs += `"`
	if options.Name != "" {
		attrs += ` name="` + template.HTMLEscapeString(options.Name) + `"`
	}
	if options.Value != "" {
		attrs += ` value="` + template.HTMLEscapeString(options.Value) + `"`
	}
	if options.Disabled {
		attrs += ` disabled aria-disabled="true"`
	}
	return template.HTML("<button" + attrs + ">" + template.HTMLEscapeString(options.Label) + "</button>")
}

// FieldOptions describes a labeled form control. Use Type=textarea for a
// multiline control; all user-provided values are escaped.
type FieldOptions struct {
	ID          string
	Name        string
	Label       string
	Type        string
	Value       string
	Placeholder string
	Required    bool
	Error       string
}

// Field renders a labeled input/textarea with an optional generic error.
func Field(options FieldOptions) template.HTML {
	id := template.HTMLEscapeString(options.ID)
	name := template.HTMLEscapeString(options.Name)
	label := template.HTMLEscapeString(options.Label)
	attrs := ` id="` + id + `" name="` + name + `"`
	if options.Required {
		attrs += ` required aria-required="true"`
	}
	if options.Placeholder != "" {
		attrs += ` placeholder="` + template.HTMLEscapeString(options.Placeholder) + `"`
	}
	var control string
	if strings.EqualFold(options.Type, "textarea") {
		control = "<textarea" + attrs + ">" + template.HTMLEscapeString(options.Value) + "</textarea>"
	} else {
		typ := options.Type
		if typ == "" {
			typ = "text"
		}
		control = `<input type="` + template.HTMLEscapeString(typ) + `"` + attrs + ` value="` + template.HTMLEscapeString(options.Value) + `">`
	}
	error := ""
	if strings.TrimSpace(options.Error) != "" {
		error = `<p class="form-error" role="alert">` + template.HTMLEscapeString(options.Error) + `</p>`
	}
	return template.HTML(`<div class="form-field"><label for="` + id + `">` + label + `</label>` + control + error + `</div>`)
}

// FormError renders a stable, non-sensitive form error message.
func FormError(message string) template.HTML {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	return template.HTML(`<p class="form-error" role="alert">` + template.HTMLEscapeString(message) + `</p>`)
}

func Empty(message string) template.HTML {
	if strings.TrimSpace(message) == "" {
		message = "Gösterilecek kayıt yok."
	}
	return template.HTML(`<div class="ui-empty" role="status">` + template.HTMLEscapeString(message) + `</div>`)
}

func Loading(label string) template.HTML {
	if strings.TrimSpace(label) == "" {
		label = "Yükleniyor"
	}
	return template.HTML(`<div class="ui-loading" role="status" aria-live="polite">` + template.HTMLEscapeString(label) + `</div>`)
}

func Error(message string) template.HTML {
	if strings.TrimSpace(message) == "" {
		message = "İşlem tamamlanamadı. Lütfen tekrar deneyin."
	}
	return template.HTML(`<div class="ui-error" role="alert">` + template.HTMLEscapeString(message) + `</div>`)
}

// Pagination renders bounded navigation controls. page is one-based.
func Pagination(page, pages int, previousURL, nextURL string) template.HTML {
	if pages < 1 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	result := `<nav class="ui-pagination" aria-label="Sayfalama"><span aria-current="page">` + strconv.Itoa(page) + `/` + strconv.Itoa(pages) + `</span>`
	if page > 1 && previousURL != "" {
		result += `<a href="` + template.HTMLEscapeString(previousURL) + `" rel="prev">Önceki</a>`
	}
	if page < pages && nextURL != "" {
		result += `<a href="` + template.HTMLEscapeString(nextURL) + `" rel="next">Sonraki</a>`
	}
	return template.HTML(result + `</nav>`)
}

type TableColumn struct {
	Key   string
	Label string
}

// Table renders a small metadata-driven table. Cells are already rendered
// text and are escaped to keep the primitive safe for untrusted data.
func Table(caption string, columns []TableColumn, rows [][]string) template.HTML {
	var out strings.Builder
	out.WriteString(`<div class="ui-table-wrap"><table class="ui-table"><caption>`)
	out.WriteString(template.HTMLEscapeString(caption))
	out.WriteString(`</caption><thead><tr>`)
	for _, column := range columns {
		out.WriteString(`<th scope="col">` + template.HTMLEscapeString(column.Label) + `</th>`)
	}
	out.WriteString(`</tr></thead><tbody>`)
	for _, row := range rows {
		out.WriteString(`<tr>`)
		for index := range columns {
			value := ""
			if index < len(row) {
				value = row[index]
			}
			out.WriteString(`<td>` + template.HTMLEscapeString(value) + `</td>`)
		}
		out.WriteString(`</tr>`)
	}
	out.WriteString(`</tbody></table></div>`)
	return template.HTML(out.String())
}
