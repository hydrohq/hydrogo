package ui

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestAdminShellInteractiveElementsHaveAccessibleNames(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Erişilebilirlik", "", ""); err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	var unnamed []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "a" || node.Data == "button") {
			if accessibleHTMLName(node) == "" {
				unnamed = append(unnamed, node.Data)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if len(unnamed) != 0 {
		t.Fatalf("interactive elements without an accessible name: %v", unnamed)
	}
}

func TestAdminShellAccessibilityCSSContract(t *testing.T) {
	css := AdminCSS + AdminPolishCSS + AdminInfoTipCSS + AdminShadcnCSS
	for _, required := range []string{
		":focus-visible",
		"prefers-reduced-motion:reduce",
		"outline-offset",
	} {
		if !strings.Contains(css, required) {
			t.Fatalf("accessibility CSS contract missing %q", required)
		}
	}
	if !strings.Contains(AdminSPARouterJS, "keydown") {
		t.Fatal("admin shell must retain keyboard interaction handling")
	}
}

func TestAdminShellDoesNotRenderRuntimeErrorDetails(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Güvenli yönetim", "", ""); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, forbidden := range []string{"err.message", "error.message"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("admin shell exposes runtime error detail %q", forbidden)
		}
	}
}

func accessibleHTMLName(node *html.Node) string {
	for _, attr := range node.Attr {
		if attr.Key == "aria-label" || attr.Key == "title" {
			if value := strings.TrimSpace(attr.Val); value != "" {
				return value
			}
		}
	}
	var text func(*html.Node) string
	text = func(current *html.Node) string {
		if current.Type == html.TextNode {
			return current.Data
		}
		var parts []string
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			parts = append(parts, text(child))
		}
		return strings.Join(parts, " ")
	}
	return strings.TrimSpace(text(node))
}
