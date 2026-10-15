package transport

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// Public news pages hand the active theme a typed view model (extension/v1
// NewsListView / NewsDetailView); the theme owns every tag and style.

const (
	designSystemStylesheet   = "/designsystem/static/designsystem.css"
	articleReadingStylesheet = "/designsystem/static/article-reading.css"
	articleReadingScript     = "/designsystem/static/article-reading.js"
	maxSliderItems           = 10
	maxCategoryLinks         = 12
)

func (h Handler) publicListHTML(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(publishedListRepository)
	if !ok {
		http.Error(w, "news list unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.PathValue("locale"))
	if locale == "" {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	items, err := repository.ListPublished(r.Context(), h.DB, locale, 50)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	view := v1.NewsListView{ActiveCategory: categoryKey(r.URL.Query().Get("category")), Total: len(items)}
	view.Site = h.siteView(r.Context(), locale, "")
	if h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		view.Site.Stylesheets = append(view.Site.Stylesheets, designSystemStylesheet)
	}
	view.Site.Widgets = template.HTML(h.socialLinksMarkup(r.Context()) + h.floatingContact(r.Context(), locale))
	view.Site.Categories = categoryLinks(locale, items, view.ActiveCategory)
	for _, item := range items {
		view.Items = append(view.Items, h.newsCard(locale, item))
	}
	if len(items) > 0 {
		lead := items[0]
		for _, candidate := range items {
			if candidate.Featured {
				lead = candidate
				break
			}
		}
		card := h.newsCard(locale, lead)
		view.Lead = &card
		view.Slides = append(view.Slides, card)
		for _, item := range items {
			if item.ID != lead.ID && len(view.Slides) < maxSliderItems {
				view.Slides = append(view.Slides, h.newsCard(locale, item))
			}
		}
		view.Headlines = view.Slides[1:min(3, len(view.Slides))]
		view.Popular = view.Slides[:min(5, len(view.Slides))]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	allowAppearanceStyle(w.Header(), view.Site.AppearanceCSS)
	if err := v1.RenderThemeView(h.Theme, w, v1.ViewNewsList, view); err != nil {
		http.Error(w, "news unavailable", http.StatusInternalServerError)
	}
}

// publicGetHTML exposes a human-facing counterpart to the JSON public route.
// The JSON endpoint remains stable for headless clients.
func (h Handler) publicGetHTML(w http.ResponseWriter, r *http.Request) {
	previewed := r.URL.Query().Get("preview_token") != ""
	if previewed {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Referrer-Policy", "no-referrer")
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.PathValue("locale"))
	slug, err := canonical.Path(r.PathValue("slug"))
	if locale == "" || err != nil {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	item, previewed, err := h.publicHTMLItem(r, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	if previewed {
		security.AllowSameOriginFrame(w.Header())
	}
	var sidebarItems []news.Item
	if repository, ok := h.Service.Repository.(relatedListRepository); ok {
		if list, err := repository.ListRelatedPublished(r.Context(), h.DB, item.ID, locale, 4); err == nil {
			sidebarItems = list
		}
	}
	articleHTML, tocHTML, err := ui.PrepareArticleHTML(ui.RichTextHTML(item.Body))
	if err != nil {
		http.Error(w, "news unavailable", http.StatusInternalServerError)
		return
	}
	var readingTools bytes.Buffer
	if err := ui.PublicArticleTools().Render(r.Context(), &readingTools); err != nil {
		http.Error(w, "news unavailable", http.StatusInternalServerError)
		return
	}
	consentNotice := ""
	if !previewed {
		consentNotice = h.analyticsConsentNotice(r, locale)
	}
	if h.AnalyticsEnabled && !previewed {
		w.Header().Set("Vary", "Cookie")
	}

	view := v1.NewsDetailView{Preview: previewed}
	view.Site = h.siteView(r.Context(), locale, item.Title)
	view.Site.Description = strings.TrimSpace(item.SEODescription)
	if view.Site.Description == "" {
		view.Site.Description = newsLeadItem(item)
	}
	if consentNotice != "" || h.PublicSupport.Enabled || h.PublicContactSettings != nil || h.Feedback != nil {
		view.Site.Stylesheets = append(view.Site.Stylesheets, designSystemStylesheet)
	}
	view.Site.Stylesheets = append(view.Site.Stylesheets, articleReadingStylesheet)
	view.Site.Scripts = append(view.Site.Scripts, articleReadingScript)
	view.Site.Widgets = template.HTML(consentNotice + h.socialLinksMarkup(r.Context()) + h.floatingContact(r.Context(), locale))
	view.Site.Categories = categoryLinks(locale, append([]news.Item{item}, sidebarItems...), "")

	card := h.newsCard(locale, item)
	view.Article = v1.NewsArticleView{
		NewsCardView:    card,
		Body:            template.HTML(articleHTML),
		TableOfContents: template.HTML(tocHTML),
		ReadingTools:    template.HTML(readingTools.String()),
		Feedback:        h.helpfulFeedback(r, item.ID, previewed),
		KeyTakeaways:    keyTakeaways(item.KeyTakeaways),
		Tags:            tagList(item.Tags),
		CategoryURL:     categoryURL(locale, card.CategoryKey),
		ShareWhatsApp:   "https://wa.me/?text=" + url.QueryEscape(item.Title),
		ShareX:          "https://twitter.com/intent/tweet?text=" + url.QueryEscape(item.Title),
	}
	for _, related := range sidebarItems {
		if related.Slug == item.Slug {
			continue
		}
		relatedCard := h.newsCard(locale, related)
		view.Related = append(view.Related, relatedCard)
		if len(view.Trending) < 4 {
			view.Trending = append(view.Trending, relatedCard)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	allowAppearanceStyle(w.Header(), view.Site.AppearanceCSS)
	if err := v1.RenderThemeView(h.Theme, w, v1.ViewNewsDetail, view); err != nil {
		http.Error(w, "news unavailable", http.StatusInternalServerError)
	}
}

func (h Handler) helpfulFeedback(r *http.Request, contentID string, previewed bool) template.HTML {
	if h.Feedback == nil || previewed {
		return ""
	}
	scope, scopeErr := tenant.FromContext(r.Context())
	if scopeErr != nil && h.RequireTenantScope {
		return ""
	}
	tenantID := "default"
	if scopeErr == nil {
		tenantID = scope.TenantID
	}
	totals, _ := h.Feedback.FeedbackTotals(r.Context(), tenantID, "news", contentID)
	var output bytes.Buffer
	props := designsystem.HelpfulFeedbackProps{ContentType: "news", ContentID: contentID, ElementID: "article-feedback", CSRF: security.CSRFTokenFromContext(r.Context()), Up: totals.Up, Down: totals.Down, LoadHTMX: true}
	if err := designsystem.HelpfulFeedback(props).Render(r.Context(), &output); err != nil {
		return ""
	}
	return template.HTML(output.String())
}

func (h Handler) siteView(ctx context.Context, locale, title string) v1.ThemeSiteView {
	return v1.ThemeSiteView{
		Locale:        locale,
		Direction:     canonical.Direction(locale),
		Title:         title,
		Today:         turkishDate(time.Now(), true),
		Navigation:    h.themeNavigation(ctx, locale),
		AppearanceCSS: h.appearanceCSS(ctx),
		NewsURL:       "/public/news/" + url.PathEscape(locale) + "/html",
		PagesURL:      "/public/pages/" + url.PathEscape(locale) + "/",
	}
}

// appearanceCSS returns the admin's validated brand tokens (modules/site
// Appearance) as a `:root{--brand:...}` rule, or empty when unconfigured.
func (h Handler) appearanceCSS(ctx context.Context) template.CSS {
	if h.PublicContactSettings == nil {
		return ""
	}
	settings, err := h.PublicContactSettings(ctx)
	if err != nil {
		return ""
	}
	css, err := site.AppearanceFromSettings(settings).CSS()
	if err != nil {
		return ""
	}
	return template.CSS(css)
}

func (h Handler) newsCard(locale string, item news.Item) v1.NewsCardView {
	src, fallback := h.coverURLs(item)
	return v1.NewsCardView{
		Title:            item.Title,
		Slug:             item.Slug,
		URL:              "/public/news/" + url.PathEscape(locale) + "/" + url.PathEscape(item.Slug) + "/html",
		Excerpt:          newsLeadItem(item),
		Category:         strings.TrimSpace(item.Category),
		CategoryKey:      categoryKey(item.Category),
		Author:           item.AuthorName,
		Date:             turkishDate(item.UpdatedAt, false),
		CoverURL:         src,
		CoverFallbackURL: fallback,
		ReadingMinutes:   readingMinutes(item.Body),
		Featured:         item.Featured,
	}
}

// coverURLs returns the card-variant cover URL (or the CDN URL when one is
// configured) and the original upload as a fallback. Both are empty when the
// story has no cover.
func (h Handler) coverURLs(item news.Item) (string, string) {
	if strings.TrimSpace(item.CoverMediaID) == "" {
		return "", ""
	}
	key := url.QueryEscape(item.CoverMediaID)
	src, fallback := "/media/"+key+"?variant=card", "/media/"+key
	if h.Media != nil {
		if publicURL, err := h.Media.PublicURL(item.CoverMediaID); err == nil && strings.TrimSpace(publicURL) != "" {
			src, fallback = publicURL, publicURL
		}
	}
	return src, fallback
}

var turkishFold = strings.NewReplacer("ğ", "g", "ü", "u", "ş", "s", "ı", "i", "ö", "o", "ç", "c")

// categoryKey turns a category label into a stable, URL-safe filter key.
func categoryKey(label string) string {
	folded := turkishFold.Replace(strings.ToLowerSpecial(unicode.TurkishCase, strings.TrimSpace(label)))
	var key strings.Builder
	dash := false
	for _, r := range folded {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			key.WriteRune(r)
			dash = false
		} else if key.Len() > 0 && !dash {
			key.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(key.String(), "-")
}

func categoryURL(locale, key string) string {
	base := "/public/news/" + url.PathEscape(locale) + "/html"
	if key == "" {
		return base
	}
	return base + "?category=" + url.QueryEscape(key)
}

// categoryLinks lists the distinct categories present in items, in order of
// first appearance, for themes that need a section menu when the site has no
// configured navigation.
func categoryLinks(locale string, items []news.Item, active string) []v1.ThemeCategoryLink {
	seen := map[string]bool{}
	var links []v1.ThemeCategoryLink
	for _, item := range items {
		label := strings.TrimSpace(item.Category)
		key := categoryKey(label)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		links = append(links, v1.ThemeCategoryLink{Key: key, Label: label, URL: categoryURL(locale, key), Active: key == active})
		if len(links) == maxCategoryLinks {
			break
		}
	}
	return links
}

func keyTakeaways(value string) []string {
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "•"))
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func tagList(value string) []string {
	var tags []string
	for _, tag := range strings.Split(value, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func (h Handler) publicHTMLItem(r *http.Request, locale, slug string) (news.Item, bool, error) {
	token := r.URL.Query().Get("preview_token")
	if token == "" {
		item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
		return item, false, err
	}
	if h.PreviewSigner == nil {
		return news.Item{}, false, errors.New("preview unavailable")
	}
	claims, err := h.PreviewSigner.Verify(token, time.Now().UTC())
	if err != nil || claims.ResourceType != "news" {
		return news.Item{}, false, errors.New("preview unavailable")
	}
	if h.RequireTenantScope || claims.TenantID != "" {
		scope, scopeErr := tenant.Require(r.Context())
		if scopeErr != nil || scope.TenantID != claims.TenantID {
			return news.Item{}, false, errors.New("preview unavailable")
		}
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, claims.ResourceID)
	if err != nil || item.Revision != claims.Revision || item.Locale != locale || item.Slug != slug {
		return news.Item{}, false, errors.New("preview unavailable")
	}
	return item, true, nil
}

func (h Handler) analyticsConsentNotice(r *http.Request, locale string) string {
	if !h.AnalyticsEnabled {
		return ""
	}
	if cookie, err := r.Cookie(analytics.ConsentCookie); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return ""
	}
	props := designsystem.CookieNoticeProps{Action: "/analytics/consent", CSRF: security.CSRFTokenFromContext(r.Context()), ReturnTo: r.URL.Path, Variant: "bar"}
	if strings.HasPrefix(canonical.Locale(locale), "en") {
		props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Privacy choices", "Allow anonymous, daily page-view totals? HydroGo does not use advertising cookies.", "Allow analytics", "Decline"
	} else {
		props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Gizlilik tercihiniz", "Anonim günlük sayfa görüntülenme toplamlarına izin verilsin mi? HydroGo reklam çerezi kullanmaz.", "Analitiğe izin ver", "Reddet"
	}
	var notice strings.Builder
	if designsystem.CookieNotice(props).Render(r.Context(), &notice) != nil {
		return ""
	}
	return notice.String()
}

func (h Handler) socialLinksMarkup(ctx context.Context) string {
	if h.PublicContactSettings == nil {
		return ""
	}
	settings, err := h.PublicContactSettings(ctx)
	if err != nil {
		return ""
	}
	profiles := site.SocialProfilesFromSettings(settings)
	links := make([]designsystem.SocialLink, 0, len(profiles))
	for _, profile := range profiles {
		links = append(links, designsystem.SocialLink{Platform: profile.Platform, URL: profile.URL})
	}
	var output bytes.Buffer
	if designsystem.SocialLinks(designsystem.SocialLinksProps{Links: links}).Render(ctx, &output) != nil {
		return ""
	}
	return output.String()
}

func (h Handler) floatingContact(ctx context.Context, locale string) string {
	props := designsystem.FloatingContactProps{Phone: h.PublicSupport.Phone, WhatsAppURL: h.PublicSupport.WhatsAppURL, AriaLabel: "İletişim seçenekleri", PhoneLabel: "Ara", WhatsAppLabel: "WhatsApp", Position: "bottom-right"}
	enabled := h.PublicSupport.Enabled
	if h.PublicContactSettings != nil {
		settings, err := h.PublicContactSettings(ctx)
		if err != nil {
			return ""
		}
		if site.ContactWidgetConfigured(settings) {
			value := site.ContactWidgetFromSettings(settings)
			enabled = value.Enabled
			props.Phone, props.WhatsAppNumber, props.Message = value.Phone, value.WhatsAppNumber, value.Message
			props.Position, props.MobileOnly = value.Position, value.MobileOnly
			props.WhatsAppURL = ""
		}
	}
	if !enabled {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		props.AriaLabel, props.PhoneLabel = "Contact options", "Call"
	}
	var output bytes.Buffer
	if err := designsystem.FloatingContact(props).Render(ctx, &output); err != nil {
		return ""
	}
	return output.String()
}

func (h Handler) themeNavigation(ctx context.Context, locale string) template.HTML {
	if h.NavigationProvider == nil {
		return ""
	}
	items, err := h.NavigationProvider(ctx, locale)
	if err != nil {
		return ""
	}
	markup, err := v1.RenderThemeNavigation(items)
	if err != nil {
		return ""
	}
	return markup
}

var markupTag = regexp.MustCompile(`<[^>]+>`)

func newsLead(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		body = markupTag.ReplaceAllString(ui.RichTextHTML(body), " ")
	}
	body = strings.Join(strings.Fields(body), " ")
	if len([]rune(body)) > 160 {
		return string([]rune(body)[:160]) + "…"
	}
	return body
}

func readingMinutes(text string) int {
	return max(1, (len(strings.Fields(text))+199)/200)
}

func newsLeadItem(item news.Item) string {
	if strings.TrimSpace(item.Excerpt) != "" {
		return strings.TrimSpace(item.Excerpt)
	}
	return newsLead(item.Body)
}
