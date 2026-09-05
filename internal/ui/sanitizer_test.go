package ui

import (
	"strings"
	"testing"
)

func TestHTMLSanitizerNeutralizesXSS(t *testing.T) {
	sanitizer := NewHTMLSanitizer()

	cases := []struct {
		name     string
		input    string
		disallow []string
		mustKeep string
	}{
		{
			name:     "inline script tag",
			input:    `<p>Hello <script>alert('xss')</script>World</p>`,
			disallow: []string{"<script>", "alert"},
			mustKeep: "<p>Hello World</p>",
		},
		{
			name:     "onerror event handler on image",
			input:    `<img src="invalid.jpg" onerror="fetch('/steal?c='+document.cookie)" />`,
			disallow: []string{"onerror", "fetch", "document.cookie"},
			mustKeep: `<img src="invalid.jpg"`,
		},
		{
			name:     "javascript URI in anchor link",
			input:    `<a href="javascript:alert(1)">Click me</a>`,
			disallow: []string{"javascript:", "alert(1)"},
			mustKeep: `Click me`,
		},
		{
			name:     "nested SVG onload payload",
			input:    `<svg><g onload="alert(1)"></g></svg>`,
			disallow: []string{"<svg>", "onload", "alert(1)"},
			mustKeep: "",
		},
		{
			name:     "safe rich text markup preserved",
			input:    `<h1>Heading</h1><p><strong>Bold</strong> and <em>Italic</em></p><ul><li>Item 1</li></ul>`,
			disallow: []string{"<script>", "onload"},
			mustKeep: `<h1>Heading</h1><p><strong>Bold</strong> and <em>Italic</em></p><ul><li>Item 1</li></ul>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := sanitizer.Sanitize(tc.input)
			for _, bad := range tc.disallow {
				if strings.Contains(strings.ToLower(output), strings.ToLower(bad)) {
					t.Errorf("sanitizer output %q still contains disallowed substring %q", output, bad)
				}
			}
			if tc.mustKeep != "" && !strings.Contains(output, tc.mustKeep) {
				t.Errorf("sanitizer output %q does not contain expected substring %q", output, tc.mustKeep)
			}
		})
	}
}
