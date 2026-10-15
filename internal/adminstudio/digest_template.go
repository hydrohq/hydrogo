package adminstudio

import (
	_ "embed"
	"html/template"
)

// digestEmail is the HTML body of the weekly and monthly admin digest e-mail
// (digest.html).
//
//go:embed digest.html
var digestEmailSource string

var digestEmail = template.Must(template.New("digest").Parse(digestEmailSource))

type digestEmailData struct {
	Stats           DigestStats
	Title, From, To string
	Monthly         bool
}
