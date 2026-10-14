package main

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"
)

//go:embed docs_portal_en.html
var docsPortalEnglish string

//go:embed docs_portal_tr.html
var docsPortalTurkish string

//go:embed docs_portal.css
var docsPortalCSS string

var docsPortalTemplates = map[string]*template.Template{
	"en": template.Must(template.New("docs-en").Parse(docsPortalEnglish)),
	"tr": template.Must(template.New("docs-tr").Parse(docsPortalTurkish)),
}

func docsPortalHTML(lang string) string {
	tmpl, ok := docsPortalTemplates[lang]
	if !ok {
		tmpl = docsPortalTemplates["en"]
	}
	var output bytes.Buffer
	data := struct {
		CSS template.CSS
	}{CSS: template.CSS(docsPortalCSS)} // CSS is a build-time, embedded asset.
	if err := tmpl.Execute(&output, data); err != nil {
		output.Reset()
		_ = docsUnavailablePage().Render(context.Background(), &output)
	}
	return output.String()
}
