package v1

import "html/template"

// The types below are the data contract between core modules and theme page
// views. Modules fill them; themes own every tag and class name. Fields typed
// template.HTML hold markup that core already rendered and sanitized (article
// bodies, design-system widgets); every other field is plain text that the
// template escapes.

// ThemeSiteView is shared by every page view.
type ThemeSiteView struct {
	Locale      string
	Direction   string
	Title       string
	Description string
	// Today is the current date, formatted for the site's locale.
	Today string
	// Navigation is the site menu rendered by RenderThemeNavigation. It is
	// empty when no menu is configured; themes then fall back to Categories.
	Navigation template.HTML
	// Categories are derived from published content, for themes that build
	// a section menu when the site has no navigation.
	Categories []ThemeCategoryLink
	// Stylesheets and Scripts are same-origin assets the module needs on the
	// page (design-system widgets, reading tools, form behaviour). Themes
	// must emit them as <link rel="stylesheet"> and <script defer>.
	Stylesheets []string
	Scripts     []string
	// Widgets is trailing body markup: consent notice, social links and the
	// floating contact button. Themes print it just before </body>.
	Widgets template.HTML
	// AppearanceCSS is the admin's brand tokens (modules/site Appearance),
	// already validated and compiled to a `:root{--brand:...}` rule. Empty
	// when the site module is disabled or no Appearance has been saved.
	// Themes print it as <style>{{.AppearanceCSS}}</style> in <head>; it is
	// typed template.CSS (not template.HTML) so html/template's contextual
	// escaper trusts it inside a <style> element instead of replacing it.
	AppearanceCSS template.CSS
	// NewsURL is the localized news index; PagesURL is the localized prefix
	// for CMS pages (append "<slug>/html").
	NewsURL  string
	PagesURL string
}

type ThemeCategoryLink struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

// NewsCardView is one story in a list, rail or related block.
type NewsCardView struct {
	Title       string
	Slug        string
	URL         string
	Excerpt     string
	Category    string
	CategoryKey string
	Author      string
	Date        string
	// CoverURL is empty when the story has no cover image.
	// CoverFallbackURL is the original upload, for a failed CDN variant.
	CoverURL         string
	CoverFallbackURL string
	ReadingMinutes   int
	Featured         bool
}

type NewsListView struct {
	Site ThemeSiteView
	// Items is every published story, newest first.
	Items []NewsCardView
	// Lead is the featured story (or the newest one); nil when Items is empty.
	Lead *NewsCardView
	// Slides starts with Lead and holds at most ten stories for a carousel.
	Slides []NewsCardView
	// Headlines are the two stories after Lead in Slides.
	Headlines []NewsCardView
	// Popular are the first five Slides.
	Popular []NewsCardView
	// ActiveCategory is the ?category= key the page was opened with.
	ActiveCategory string
	Total          int
}

type NewsArticleView struct {
	NewsCardView
	Body            template.HTML
	TableOfContents template.HTML
	ReadingTools    template.HTML
	Feedback        template.HTML
	KeyTakeaways    []string
	Tags            []string
	CategoryURL     string
	ShareWhatsApp   string
	ShareX          string
}

type NewsDetailView struct {
	Site    ThemeSiteView
	Article NewsArticleView
	// Related excludes the article itself; Trending holds at most four.
	Related  []NewsCardView
	Trending []NewsCardView
	Preview  bool
}

// MemberAuthView drives the login and registration pages. The form must post
// to Action with the hidden _csrf and return_to fields and carry the
// data-member-form attribute; ScriptURL adds the JSON submit behaviour.
type MemberAuthView struct {
	Site                 ThemeSiteView
	Register             bool
	Action               string
	CSRF                 string
	ReturnTo             string
	SwitchURL            string
	PasswordAutocomplete string
	// PasswordStrength is the design-system meter shown on registration.
	PasswordStrength template.HTML
	// OAuthProviders lists configured social sign-in options; empty when
	// none are configured, so a theme's {{if .OAuthProviders}} renders
	// nothing and the page is visually unchanged.
	OAuthProviders []MemberOAuthProviderView
}

// MemberOAuthProviderView is one "sign in with ..." link on the member auth
// page.
type MemberOAuthProviderView struct {
	Name  string // "google", "github" or "apple"
	Label string // display name, e.g. "Google"
	URL   string // /member/oauth/{Name}/start, with return_to preserved
}

type MemberAccountView struct {
	Site          ThemeSiteView
	Email         string
	Initial       string
	EmailVerified bool
	RegisteredAt  string
	Sessions      []MemberSessionView
	CSRF          string
}

type MemberSessionView struct {
	UserAgent string
	IPAddress string
	LastSeen  string
	Current   bool
}
