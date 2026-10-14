package transport

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/a-h/templ"
	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

func (h Handler) adminNewPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	values := map[string]string{"locale": h.defaultLocale(), "expires_at": ""}
	view := newsNewView{Locale: h.defaultLocale(), CSRF: security.CSRFTokenFromContext(r.Context())}
	view.Content, view.Meta, view.SEO = h.newsFormSections(r, "/admin/news", values)
	renderAdminScreen(w, r, "Yeni haber", "Yayın merkezi", newsNewScreen(view))
}

// newsFormSections renders the metadata-driven fields of the news form in
// the three groups the split editor shows: content, general and SEO.
func (h Handler) newsFormSections(r *http.Request, action string, values map[string]string) (content, meta, seo templ.Component) {
	fieldMap := make(map[string]adminstudio.Field)
	for _, f := range h.AdminCollection.Fields {
		fieldMap[f.ID] = f
	}
	pick := func(ids ...string) []adminstudio.Field {
		var fields []adminstudio.Field
		for _, id := range ids {
			if f, ok := fieldMap[id]; ok {
				fields = append(fields, f)
			}
		}
		return fields
	}
	metaFields := pick("featured", "author_name", "category", "tags", "expires_at")
	if !hasAdminField(metaFields, "expires_at") {
		metaFields = append(metaFields, adminstudio.Field{ID: "expires_at", Label: "Otomatik yayından kalkma", Kind: adminstudio.FieldDate, Widget: "datetime-local"})
	}
	csrf := security.CSRFTokenFromContext(r.Context())
	uiLocale := ui.AdminLocaleFromRequest(r)
	section := func(id, label string, fields []adminstudio.Field) templ.Component {
		collection := adminstudio.Collection{ID: id, Label: label, Fields: fields, Actions: []adminstudio.Action{{Label: "Kaydet", Href: action}}}
		return writerComponent(func(out io.Writer) error {
			return adminstudio.RenderCreateForm(out, adminstudio.FormData{Collection: collection, Action: action, CSRFToken: csrf, Values: values, HideSubmit: true, UILocale: uiLocale})
		})
	}
	return section("news_left", "Haber İçeriği", pick("title", "slug", "cover_media_id", "excerpt", "key_takeaways", "body")),
		section("news_meta", "Genel Bilgiler", metaFields),
		section("news_seo", "SEO & Arama Motoru", pick("seo_title", "seo_description"))
}

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	if !editorAllowed(r) {
		h.renderReadOnlyNews(w, r, item)
		return
	}
	expiresValue := ""
	if item.ExpiresAt != nil {
		expiresValue = item.ExpiresAt.UTC().Format("2006-01-02T15:04")
	}
	values := map[string]string{
		"locale":          item.Locale,
		"slug":            item.Slug,
		"title":           item.Title,
		"body":            item.Body,
		"cover_media_id":  item.CoverMediaID,
		"excerpt":         item.Excerpt,
		"key_takeaways":   item.KeyTakeaways,
		"category":        item.Category,
		"author_name":     item.AuthorName,
		"featured":        boolText(item.Featured),
		"tags":            item.Tags,
		"seo_title":       item.SEOTitle,
		"seo_description": item.SEODescription,
		"expires_at":      expiresValue,
	}
	csrf := security.CSRFTokenFromContext(r.Context())
	base := "/admin/news/" + url.PathEscape(item.ID)
	view := newsEditView{
		ID:              item.ID,
		Action:          base,
		Title:           item.Title,
		Status:          string(item.Status),
		Locale:          item.Locale,
		Version:         item.Version,
		CSRF:            csrf,
		PreviewEndpoint: base + "/preview",
		Workflow:        newsWorkflow(h.Service, item, csrf),
	}
	if strings.TrimSpace(view.Title) == "" {
		view.Title = untitledNews
	}
	if item.Status == news.StatusPublished && item.Slug != "" {
		publicPath := "/public/news/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
		view.LiveURL = strings.TrimRight(strings.TrimSpace(h.PublicOrigin), "/") + publicPath
	}
	view.Content, view.Meta, view.SEO = h.newsFormSections(r, "/admin/news/"+item.ID, values)
	if taxonomy, ok := h.Service.Repository.(taxonomyAdminRepository); ok {
		terms, termsErr := taxonomy.ListTaxonomyTerms(r.Context(), h.DB)
		selected, selectedErr := taxonomy.ListTaxonomy(r.Context(), h.DB, item.ID)
		if termsErr != nil || selectedErr != nil {
			http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
			return
		}
		selectedIDs := make([]string, 0, len(selected))
		for _, term := range selected {
			selectedIDs = append(selectedIDs, term.ID)
		}
		view.Taxonomy = writerComponent(func(out io.Writer) error {
			return adminstudio.RenderTaxonomyEditor(out, adminstudio.TaxonomyEditorOptions{Terms: terms, Selected: selectedIDs})
		})
	}
	if h.Presence != nil {
		view.Presence = designsystem.EditPresence(item.ID, base+"/presence", csrf)
	}
	renderAdminScreen(w, r, "Haber düzenle", "Yayın merkezi", newsEditScreen(view))
}

func (h Handler) renderReadOnlyNews(w http.ResponseWriter, r *http.Request, item news.Item) {
	view := newsReadOnlyView{BodyHTML: ui.RichTextHTML(item.Body), Fields: []uicomponents.InputProps{
		{ID: "news-title-readonly", Name: "title", Label: "Başlık", Value: item.Title, ReadOnly: true},
		{ID: "news-slug-readonly", Name: "slug", Label: "Adres", Value: item.Slug, ReadOnly: true},
		{ID: "news-locale-readonly", Name: "locale", Label: "Dil", Value: item.Locale, ReadOnly: true},
		{ID: "news-status-readonly", Name: "status", Label: "Durum", Value: string(item.Status), ReadOnly: true},
		{ID: "news-category-readonly", Name: "category", Label: "Kategori", Value: item.Category, ReadOnly: true},
		{ID: "news-author-readonly", Name: "author", Label: "Yazar", Value: item.AuthorName, ReadOnly: true},
		{ID: "news-tags-readonly", Name: "tags", Label: "Etiketler", Value: item.Tags, ReadOnly: true},
	}}
	renderAdminScreen(w, r, "Haberi görüntüle", "Salt okunur içerik", uicomponents.Card(uicomponents.CardProps{Title: "Haber içeriği", Class: "studio-card", Body: newsReadOnlyBody(view)}))
}

// newsWorkflow lists the editorial transitions the configured workflow allows
// from the item's current state.
func newsWorkflow(service news.Service, item news.Item, csrf string) newsWorkflowView {
	return newsWorkflowView{
		Base:        "/admin/news/" + url.PathEscape(item.ID),
		Version:     item.Version,
		CSRF:        csrf,
		CanSchedule: service.CanTransition(item.Status, v1.EditorialScheduled),
		CanPublish:  service.CanTransition(item.Status, v1.EditorialPublished),
		CanRetract:  service.CanTransition(item.Status, v1.EditorialRetracted),
	}
}

func boolText(value bool) string {
	if value {
		return "on"
	}
	return ""
}

func hasAdminField(fields []adminstudio.Field, id string) bool {
	for _, field := range fields {
		if field.ID == id {
			return true
		}
	}
	return false
}

func parseExpiresAt(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04", value, time.UTC)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}
