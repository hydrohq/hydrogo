package mailer

import (
	"bytes"
	"strings"
)

func RenderAutoresponderHTML(title, body string) (string, error) {
	paragraphs := strings.Split(strings.TrimSpace(body), "\n")
	var output bytes.Buffer
	if err := emailTemplates.ExecuteTemplate(&output, "autoresponder", struct {
		Title      string
		Paragraphs []string
	}{title, paragraphs}); err != nil {
		return "", err
	}
	return output.String(), nil
}
