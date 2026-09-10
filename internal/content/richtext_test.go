package content

import "testing"

func TestValidateRichText(t *testing.T) {
	for _, value := range []string{"plain text", `{"format":"quill-delta","format_version":1,"document":[{"insert":"hello\n"}]}`, `{"ops":[{"insert":"hello\n"}]}`} {
		if err := ValidateRichText(value); err != nil {
			t.Errorf("valid rich text rejected: %v", err)
		}
	}
	for _, value := range []string{`{"format":"other","document":[]}`, `{"document":{"insert":"bad"}}`} {
		if err := ValidateRichText(value); err == nil {
			t.Errorf("invalid rich text accepted: %s", value)
		}
	}
}

func TestValidateRichTextBoundsInput(t *testing.T) {
	if err := ValidateRichText(string(make([]byte, MaxRichTextBytes+1))); err == nil {
		t.Fatal("oversized rich text accepted")
	}
}
