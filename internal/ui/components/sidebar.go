package components

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// SidebarItem is a safe, presentation-only navigation entry supplied by an
// active module manifest. Icons are identifiers, never raw SVG or HTML.
type SidebarItem struct {
	Route  string
	Label  string
	Icon   string
	Group  string
	Active bool
}

func Sidebar(items []SidebarItem) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, `<nav aria-label="Modüller">`); err != nil {
			return err
		}
		lastGroup := ""
		for _, item := range items {
			group := strings.TrimSpace(item.Group)
			if group != lastGroup && group != "" {
				if _, err := io.WriteString(w, `<span class="sidebar-section-label">`+esc(strings.ToUpper(group))+`</span>`); err != nil {
					return err
				}
				lastGroup = group
			}
			className := "nav-item"
			if item.Active {
				className += " is-active"
			}
			markup := `<a class="` + esc(className) + `" href="` + esc(item.Route) + `"><span class="nav-icon" data-icon="` + esc(item.Icon) + `" aria-hidden="true"></span><span>` + esc(item.Label) + `</span></a>`
			if _, err := io.WriteString(w, markup); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</nav>`)
		return err
	})
}
