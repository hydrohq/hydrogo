package designsystem

import (
	"context"
	"strings"
	"testing"
)

func TestCodeBlockHighlightsAndEscapesSource(t *testing.T) {
	code := "func main() {\n\tprintln(\"Merhaba <HydroGo>\")\n}"
	var out strings.Builder
	if err := CodeBlock(CodeBlockProps{Code: code, Language: "go"}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{`class="code-token code-token-keyword">func</span>`, `data-copy-code`, `role="status"`, `Merhaba &lt;HydroGo&gt;`} {
		if !strings.Contains(got, want) {
			t.Fatalf("code block missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "<HydroGo>") {
		t.Fatalf("source was not escaped: %s", got)
	}
}

func TestCodeTokensPreserveExactUnicodeAndNormalizeLanguage(t *testing.T) {
	code := "// İzmir\nconst şehir = \"açık\""
	var got strings.Builder
	for _, token := range CodeTokens(code, "typescript") {
		got.WriteString(token.Text)
	}
	if got.String() != code {
		t.Fatalf("tokenized source = %q, want %q", got.String(), code)
	}
	if tokens := CodeTokens(code, "typescript"); len(tokens) < 4 || tokens[2].Kind != "keyword" {
		t.Fatalf("typescript keywords were not highlighted: %#v", tokens)
	}
}
