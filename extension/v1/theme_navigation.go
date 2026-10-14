package v1

import (
	"context"
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
	rendered := 0
	var build func(string, int, map[string]bool) ([]themeNavigationNode, error)
	build = func(parent string, depth int, stack map[string]bool) ([]themeNavigationNode, error) {
		if depth > 8 {
			return nil, ErrInvalidThemeNavigation
		}
		nodes := make([]themeNavigationNode, 0, len(children[parent]))
		for _, item := range children[parent] {
			if stack[item.ID] {
				return nil, ErrInvalidThemeNavigation
			}
			stack[item.ID] = true
			rendered++
			nested, err := build(item.ID, depth+1, stack)
			if err != nil {
				return nil, err
			}
			delete(stack, item.ID)
			nodes = append(nodes, themeNavigationNode{Item: item, Children: nested})
		}
		return nodes, nil
	}
	roots, err := build("", 0, make(map[string]bool))
	if err != nil {
		return "", err
	}
	if rendered != len(items) {
		return "", ErrInvalidThemeNavigation
	}
	if len(roots) == 0 {
		return "", nil
	}
	var output strings.Builder
	if err := themeNavigationList(roots, 0).Render(context.Background(), &output); err != nil {
		return "", err
	}
	return template.HTML(output.String()), nil
}

// themeNavigationNode is a validated menu entry with its sorted children.
type themeNavigationNode struct {
	Item     ThemeNavigationItem
	Children []themeNavigationNode
}

func themeNavigationListClass(depth int) string {
	if depth > 0 {
		return "site-navigation-child-list"
	}
	return "site-navigation-list"
}
