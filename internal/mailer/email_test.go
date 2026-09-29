package mailer

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestAutoresponderHTMLIsTemplEscaped(t *testing.T) {
	got, err := RenderAutoresponderHTML(`<Hi>`, `Hello <script>alert(1)</script>
Thanks`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") || !strings.Contains(got, "Thanks") {
		t.Fatalf("unsafe or incomplete autoresponder: %s", got)
	}
}

func TestSMTPFormatsPlainAndHTMLAlternative(t *testing.T) {
	message := formatMessage("sender@example.test", Message{To: "user@example.test", Subject: "Thanks", Body: "Plain", HTMLBody: "<p>HTML</p>"})
	for _, want := range []string{"multipart/alternative", "text/plain; charset=utf-8", "text/html; charset=utf-8", "Plain", "<p>HTML</p>"} {
		if !strings.Contains(message, want) {
			t.Errorf("missing %q from message: %s", want, message)
		}
	}
}

func TestSMTPFormatsBoundedPDFAttachment(t *testing.T) {
	message := Message{To: "user@example.test", Subject: "Quote", Body: "Attached", Attachments: []Attachment{{Filename: "quote-1.pdf", ContentType: "application/pdf", Data: []byte("%PDF-test")}}}
	if err := validateMessage(message); err != nil {
		t.Fatal(err)
	}
	formatted := formatMessage("sender@example.test", message)
	parsed, err := mail.ReadMessage(strings.NewReader(formatted))
	if err != nil {
		t.Fatal(err)
	}
	mediaType, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/mixed" {
		t.Fatalf("content type = %q, err=%v", mediaType, err)
	}
	reader := multipart.NewReader(parsed.Body, params["boundary"])
	found := false
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if part.FileName() == "quote-1.pdf" {
			data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
			if err != nil || string(data) != "%PDF-test" {
				t.Fatalf("attachment=%q err=%v", data, err)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("PDF attachment missing")
	}
	message.Attachments[0].Filename = "../unsafe.pdf"
	if err := validateMessage(message); err == nil {
		t.Fatal("accepted a path-like attachment filename")
	}
}
