package components

// SidebarItem is a safe, presentation-only navigation entry supplied by an
// active module manifest. Icons are identifiers, never raw SVG or HTML.
type SidebarItem struct {
	Route  string
	Label  string
	Icon   string
	Group  string
	Active bool
}
