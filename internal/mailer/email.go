package mailer

import (
	"bytes"
	"strings"
)

// EmailNotification represents a transactional or autoresponder notification
// queued for fenced asynchronous delivery via the mail spool.
type EmailNotification struct {
	EventKey string
	To       string
	Subject  string
	Body     string
	HTMLBody string
}

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
