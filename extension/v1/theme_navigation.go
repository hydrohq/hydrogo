package v1

import (
	"errors"
	"html/template"
	"net/url"
	"sort"
	"strings"
)

var ErrInvalidThemeNavigation = errors.New("invalid theme navigation")

// ThemeNavigationItem is the safe, bounded navigation projection supplied to
// a theme layout through its "navigation" slot.
type ThemeNavigationItem struct {
	ID       string
	ParentID string
	Label    string
	Href     string
	Target   string
	Position int
}

// RenderThemeNavigation renders a nested menu as escaped HTML for the
// allowlisted navigation slot. External URLs are rejected at this boundary.
func RenderThemeNavigation(items []ThemeNavigationItem) (template.HTML, error) {
	if len(items) > 500 {
		return "", ErrInvalidThemeNavigation
	}
	byID := make(map[string]ThemeNavigationItem, len(items))
	children := make(map[string][]ThemeNavigationItem)
	for _, item := range items {
		parsed, err := url.Parse(item.Href)
		if item.ID == "" || len(item.ID) > 128 || len(item.ParentID) > 128 || len(item.Label) == 0 || len(item.Label) > 256 || item.Position < 0 || err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(item.Href, "/") || strings.HasPrefix(item.Href, "//") || strings.ContainsAny(item.Href, "\\\r\n\x00") || (item.Target != "" && item.Target != "_self" && item.Target != "_blank") {
			return "", ErrInvalidThemeNavigation
		}
		if _, duplicate := byID[item.ID]; duplicate {
			return "", ErrInvalidThemeNavigation
		}
		byID[item.ID] = item
	}
	for _, item := range items {
		if item.ParentID != "" {
			if _, exists := byID[item.ParentID]; !exists || item.ParentID == item.ID {
				return "", ErrInvalidThemeNavigation
			}
		}
		children[item.ParentID] = append(children[item.ParentID], item)
	}
	for parent := range children {
		sort.Slice(children[parent], func(i, j int) bool {
			if children[parent][i].Position == children[parent][j].Position {
				return children[parent][i].ID < children[parent][j].ID
			}
			return children[parent][i].Position < children[parent][j].Position
		})
	}
	var output strings.Builder
	rendered := 0
	var render func(string, int, map[string]bool) error
	render = func(parent string, depth int, stack map[string]bool) error {
		if depth > 8 {
			return ErrInvalidThemeNavigation
		}
		items := children[parent]
		if len(items) == 0 {
			return nil
		}
		class := "site-navigation-list"
		if depth > 0 {
			class = "site-navigation-child-list"
		}
		output.WriteString(`<ul class="` + class + `">`)
		for _, item := range items {
			if stack[item.ID] {
				return ErrInvalidThemeNavigation
			}
			stack[item.ID] = true
			rendered++
			output.WriteString(`<li class="site-navigation-item"><a class="site-navigation-link" href="` + template.HTMLEscapeString(item.Href) + `"`)
			if item.Target == "_blank" {
				output.WriteString(` target="_blank" rel="noopener noreferrer"`)
			}
			output.WriteString(`>` + template.HTMLEscapeString(item.Label) + `</a>`)
			if err := render(item.ID, depth+1, stack); err != nil {
				return err
			}
			delete(stack, item.ID)
			output.WriteString(`</li>`)
		}
		output.WriteString(`</ul>`)
		return nil
	}
	if err := render("", 0, make(map[string]bool)); err != nil {
		return "", err
	}
	if rendered != len(items) {
		return "", ErrInvalidThemeNavigation
	}
	return template.HTML(output.String()), nil
}
