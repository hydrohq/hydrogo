// Package shell renders the Admin Studio frame: document head, sidebar,
// topbar, page heading and the shared dialogs at the end of the page.
//
// Everything the frame shows comes from a typed View. Navigation is data
// (see nav.go) filtered in Go for the signed-in user before rendering, so the
// markup never needs rewriting afterwards.
package shell

import "strings"

// Text is a label in the two admin locales.
type Text struct {
	TR string
	EN string
}

// In returns the label for locale, falling back to Turkish.
func (t Text) In(locale string) string {
	if locale == "en" && t.EN != "" {
		return t.EN
	}
	return t.TR
}

// User is the signed-in staff account the frame is rendered for.
type User struct {
	Email string
	// Role is the auth role name: admin, editor or viewer.
	Role string
	// Permissions are the effective admin permissions of the account.
	Permissions []string
	// Scope is an opaque per-user key for client-side draft recovery.
	Scope string
}

// Can reports whether the user holds permission.
func (u User) Can(permission string) bool {
	for _, allowed := range u.Permissions {
		if allowed == permission {
			return true
		}
	}
	return false
}

// Name is the short account name shown in the topbar.
func (u User) Name() string {
	if local, _, _ := strings.Cut(u.Email, "@"); local != "" {
		return local
	}
	return "Admin"
}

// ContentType is one YAML content type listed under "Content types".
type ContentType struct {
	ID    string
	Label string
}

// Action is the primary call to action next to the page title.
type Action struct {
	Href  string
	Label Text
}

// Support is the agency support card at the bottom of the sidebar.
type Support struct {
	AgencyName  string
	SLABadge    string
	Email       string
	Phone       string
	WhatsAppURL string
	GuideURL    string
}

// StaffNotice is a dismissible broadcast shown above the frame.
type StaffNotice struct {
	Message    string
	Severity   string
	DismissKey string
}

// Banners are the system-state warnings shown above the frame.
type Banners struct {
	Maintenance      bool
	BrokenLinksJobID string
	SafeMode         bool
	StaffNotice      *StaffNotice
}

// View is everything the shell needs to render one admin page frame.
type View struct {
	Title       string
	Eyebrow     string
	Description string
	// Locale is "tr" or "en".
	Locale string
	// Active is the navigation key of the current page ("pages", "news", …).
	Active string
	// Path is the request path, used to highlight content-type links.
	Path string
	User User
	// Modules is the set of enabled modules. Nil means unknown: no item is
	// hidden for module reasons.
	Modules      map[string]bool
	ContentTypes []ContentType
	// Trash adds the recycle-bin link (only on editions that mount it).
	Trash   bool
	Action  *Action
	Support *Support
	Banners Banners
}

// Lang is the document language.
func (v View) Lang() string {
	if v.Locale == "en" {
		return "en"
	}
	return "tr"
}

// T picks the label for the view's locale.
func (v View) T(text Text) string {
	return text.In(v.Lang())
}

func (v View) moduleEnabled(module string) bool {
	if module == "" || v.Modules == nil {
		return true
	}
	return v.Modules[module]
}
