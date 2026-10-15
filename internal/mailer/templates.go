package mailer

import (
	"bytes"
	"embed"
	"html/template"
)

// E-mail markup lives in templates/mail.html. Clients ignore stylesheets, so
// the styles are inline attributes; PrimaryColor is a validated six-digit hex
// value (EmailBranding.Normalize).
//
//go:embed templates/mail.html
var emailTemplateFS embed.FS

var emailTemplates = template.Must(template.ParseFS(emailTemplateFS, "templates/mail.html"))

type emailHeaderData struct {
	LogoURL, LogoAlt, Name string
	Color                  template.CSS
}

type emailFooterData struct {
	Lines            []string
	PrivacyPolicyURL string
}

// renderEmailTemplate renders a named e-mail fragment. The templates only
// write to a buffer, so an error leaves the fragment empty rather than
// failing a send.
func renderEmailTemplate(name string, data any) string {
	var output bytes.Buffer
	if err := emailTemplates.ExecuteTemplate(&output, name, data); err != nil {
		return ""
	}
	return output.String()
}
