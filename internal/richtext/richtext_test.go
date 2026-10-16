package richtext

import "testing"

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
