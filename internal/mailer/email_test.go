package mailer

import (
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
