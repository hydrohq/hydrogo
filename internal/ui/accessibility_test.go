package ui

import (
	"bytes"
	"math"
	"strconv"
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
	css := AdminCSS + AdminPolishCSS + AdminInfoTipCSS
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

func TestLightThemeAlertTextMeetsWCAGAA(t *testing.T) {
	if !strings.Contains(AdminCSSOverrides, `html[data-theme=light] .studio-alert{color:#991b1b}`) {
		t.Fatal("light-theme alerts must use the accessible dark-red text token")
	}
	if !strings.Contains(AdminCSSOverrides, `.studio-alert a{color:inherit;text-decoration:underline;text-underline-offset:2px}`) {
		t.Fatal("alert links must inherit the accessible alert color and remain visibly identifiable")
	}
	palettes := []struct {
		name, css, background string
	}{
		{"hydrogo", AdminHydroGoCSS, "#fdecee"},
		{"base", AdminCSS, "#fff1f2"},
	}
	for _, palette := range palettes {
		t.Run(palette.name, func(t *testing.T) {
			if !strings.Contains(palette.css, "--red-subtle:"+palette.background) {
				t.Fatalf("light alert background %s changed without updating this contrast check", palette.background)
			}
			if ratio := wcagContrast("#991b1b", palette.background); ratio < 4.5 {
				t.Fatalf("alert text contrast = %.2f:1, want at least 4.5:1", ratio)
			}
		})
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

func wcagContrast(foreground, background string) float64 {
	fg, bg := relativeLuminance(foreground), relativeLuminance(background)
	if fg < bg {
		fg, bg = bg, fg
	}
	return (fg + 0.05) / (bg + 0.05)
}

func relativeLuminance(value string) float64 {
	value = strings.TrimPrefix(value, "#")
	var channels [3]float64
	for i := range channels {
		channel, err := strconv.ParseUint(value[i*2:i*2+2], 16, 8)
		if err != nil {
			panic(err)
		}
		normalized := float64(channel) / 255
		if normalized <= 0.04045 {
			channels[i] = normalized / 12.92
		} else {
			channels[i] = math.Pow((normalized+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}
