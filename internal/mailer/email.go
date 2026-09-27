package mailer

import (
	"bytes"
	"context"
	"strings"
)

func RenderAutoresponderHTML(title, body string) (string, error) {
	paragraphs := strings.Split(strings.TrimSpace(body), "\n")
	var output bytes.Buffer
	if err := AutoresponderEmail(title, paragraphs).Render(context.Background(), &output); err != nil {
		return "", err
	}
	return output.String(), nil
}
