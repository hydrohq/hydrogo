package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

type Handler struct {
	Service         news.Service
	DB              database.Executor
	AdminCollection adminstudio.Collection
}

type publishedListRepository interface {
	ListPublished(context.Context, database.Executor, string, int) ([]news.Item, error)
}

type adminListRepository interface {
	ListAdmin(context.Context, database.Executor, int) ([]news.Item, error)
}

type input struct {
	Locale string `json:"locale"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type updateInput struct {
	input
	Version int `json:"version"`
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "POST /api/news", Protected: true, Handler: http.HandlerFunc(h.create)},
		{Pattern: "GET /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.get)},
		{Pattern: "PUT /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.update)},
		{Pattern: "POST /api/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.schedule)},
		{Pattern: "POST /api/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.publish)},
		{Pattern: "POST /api/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.retract)},
		{Pattern: "GET /api/news", Handler: http.HandlerFunc(h.publicList)},
		{Pattern: "GET /api/news/by-slug", Handler: http.HandlerFunc(h.publicGetByQuery)},
		{Pattern: "GET /admin/news", Protected: true, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "GET /admin/news/new", Protected: true, Handler: http.HandlerFunc(h.adminNewPage)},
		{Pattern: "POST /admin/news", Protected: true, Handler: http.HandlerFunc(h.adminCreate)},
		{Pattern: "GET /admin/news/{id}", Protected: true, Handler: http.HandlerFunc(h.adminPage)},
		{Pattern: "POST /admin/news/{id}", Protected: true, Handler: http.HandlerFunc(h.adminUpdate)},
		{Pattern: "POST /admin/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.adminSchedule)},
		{Pattern: "POST /admin/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.adminPublish)},
		{Pattern: "POST /admin/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.adminRetract)},
		{Pattern: "GET /public/news/{locale}/{slug}/html", Handler: http.HandlerFunc(h.publicGetHTML)},
		{Pattern: "GET /public/news/{locale}/html", Handler: http.HandlerFunc(h.publicListHTML)},
		{Pattern: "GET /public/news/{locale}/{slug}", Handler: http.HandlerFunc(h.publicGet)},
	}
}

func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(adminListRepository)
	if !ok {
		http.Error(w, "news admin list unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	rows := make([]adminstudio.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, adminstudio.Row{Values: map[string]string{"locale": item.Locale, "slug": item.Slug, "title": item.Title, "status": string(item.Status)}, Actions: []adminstudio.Action{{Label: "Düzenle", Href: "/admin/news/" + item.ID}}})
	}
	if r.URL.Query().Get("fragment") == "table" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: h.AdminCollection, Rows: rows}); err != nil {
			http.Error(w, "news admin unavailable", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Haberler", "Yayın merkezi", security.NonceFromContext(r.Context()), "news"); err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><h2>İçerik akışı</h2><p>Haberleri taslak olarak oluşturun, zamanlayın ve tek tıkla yayınlayın.</p><div class="studio-actions"><a class="button" href="/admin/news/new">Yeni haber oluştur</a><a class="button button-quiet" href="/public/news/tr/html" target="_blank">Canlı siteyi aç ↗</a></div></section><section class="studio-card"><h2>Haberler <span class="muted">(%d)</span></h2><div class="table-wrap">`, len(rows))
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: h.AdminCollection, Rows: rows}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = fmt.Fprintf(w, `</div></section>`)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminNewPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Yeni haber", "Yayın merkezi", security.NonceFromContext(r.Context()), "news"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><div class="dashboard-card-head"><div><p class="eyebrow">YENİ İÇERİK</p><h2>Yeni haber oluştur</h2><p class="muted">İçerik önce taslak olarak kaydedilir. Daha sonra detay sayfasından düzenleyebilir, zamanlayabilir veya yayınlayabilirsiniz.</p></div><a class="button button-quiet" href="/admin/news">Listeye dön</a></div><form class="studio-form" method="post" action="/admin/news">`)
	if err := adminstudio.RenderCreateForm(w, adminstudio.FormData{Collection: h.AdminCollection, Action: "/admin/news", CSRFToken: security.CSRFTokenFromContext(r.Context()), Nonce: security.NonceFromContext(r.Context()), Values: map[string]string{"locale": "tr"}, SubmitLabel: "Taslak oluştur"}); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `</form></section>`)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	item := news.Item{ID: id, Locale: r.FormValue("locale"), Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), Status: news.StatusDraft}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.URLQueryEscaper(id), http.StatusSeeOther)
}

func (h Handler) publicList(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(publishedListRepository)
	if !ok {
		http.Error(w, "news list unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if locale == "" {
		http.Error(w, "locale is required", http.StatusBadRequest)
		return
	}
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	items, err := repository.ListPublished(r.Context(), h.DB, locale, limit)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value input
	if !decodeJSON(w, r, &value) {
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	item := news.Item{ID: id, Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, Status: news.StatusDraft}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) { h.write(w, r, r.PathValue("id")) }

func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value updateInput
	if !decodeJSON(w, r, &value) {
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, Status: news.StatusDraft}
	if err := h.Service.UpdateDraft(r.Context(), item, value.Version); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) schedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value struct {
		Version     int    `json:"version"`
		ScheduledAt string `json:"scheduled_at"`
	}
	if !decodeJSON(w, r, &value) {
		return
	}
	at, err := time.Parse(time.RFC3339, value.ScheduledAt)
	if err != nil {
		http.Error(w, "invalid schedule", http.StatusBadRequest)
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), value.Version, at); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := h.Service.Publish(r.Context(), r.PathValue("id"), version); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) retract(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := h.Service.Retract(r.Context(), r.PathValue("id"), version); err != nil {
		writeError(w, err)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publicGet(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.PathValue("locale"), r.PathValue("slug"))
}

func (h Handler) publicGetByQuery(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"))
}

func (h Handler) publicGetWithRoute(w http.ResponseWriter, r *http.Request, rawLocale, rawSlug string) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(rawLocale)
	slug, err := canonical.Path(rawSlug)
	if locale == "" || err != nil || slug == "" {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	item, err := h.Service.Repository.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// publicGetHTML exposes a human-facing counterpart to the JSON public route.
// The JSON endpoint remains stable for headless clients.
func (h Handler) publicGetHTML(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.PathValue("locale"))
	slug, err := canonical.Path(r.PathValue("slug"))
	if locale == "" || err != nil || slug == "" {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	item, err := h.Service.Repository.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, newsDocumentStart(locale, item.Title))
	_, _ = fmt.Fprintf(w, `<main class="news-portal news-article"><a class="news-back" href="/public/news/%s/html">← Tüm haberler</a><article><div class="news-article-kicker">HYDROGO / EDITORIAL</div><h1>%s</h1><div class="news-article-meta"><span>Yayınlanmış içerik</span><span>%d. revizyon</span></div><p class="news-lead">%s</p><div class="news-prose">%s</div><div class="news-share"><strong>Paylaş</strong><a href="mailto:?subject=%s">E-posta</a><a href="/public/news/%s/html">Permalink</a></div></article></main><footer class="news-footer"><span>HydroGo Newsroom</span><a href="/shop">Mağazaya geç</a></footer></body></html>`, template.HTMLEscapeString(locale), template.HTMLEscapeString(item.Title), item.PublishedRevision, template.HTMLEscapeString(newsLead(item.Body)), renderParagraphs(item.Body), template.URLQueryEscaper(item.Title), template.HTMLEscapeString(r.PathValue("slug")))
}

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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, newsDocumentStart(locale, "Haberler"))
	_, _ = fmt.Fprintf(w, `<main class="news-portal"><section class="news-intro"><div><div class="news-article-kicker">HYDROGO / NEWSROOM</div><h1>Ürün, içerik ve teknoloji notları.</h1><p>HydroGo’nun nasıl geliştiğini, yeni yetenekleri ve ekipten kısa notları takip edin.</p></div><a class="news-primary" href="/public/pages/tr/showcase/html">Platformu keşfet →</a></section>`)
	if len(items) == 0 {
		_, _ = fmt.Fprint(w, `<section class="news-empty">Henüz yayınlanmış haber yok.</section>`)
	} else {
		_, _ = fmt.Fprint(w, `<section class="news-lead-grid"><article class="news-feature"><div class="news-number">01</div><div class="news-article-kicker">ÖNE ÇIKAN</div><h2><a href="`)
		_, _ = fmt.Fprintf(w, `/public/news/%s/%s/html">%s</a></h2><p>%s</p><a class="news-read" href="/public/news/%s/%s/html">Hikâyeyi oku →</a></article><div class="news-side-list">`, template.HTMLEscapeString(locale), template.HTMLEscapeString(items[0].Slug), template.HTMLEscapeString(items[0].Title), template.HTMLEscapeString(newsLead(items[0].Body)), template.HTMLEscapeString(locale), template.HTMLEscapeString(items[0].Slug))
		for i, item := range items[1:] {
			_, _ = fmt.Fprintf(w, `<article class="news-side-card"><span>%02d</span><div><div class="news-article-kicker">YAYIN</div><h3><a href="/public/news/%s/%s/html">%s</a></h3><p>%s</p></div></article>`, i+2, template.HTMLEscapeString(locale), template.HTMLEscapeString(item.Slug), template.HTMLEscapeString(item.Title), template.HTMLEscapeString(newsLead(item.Body)))
		}
		_, _ = fmt.Fprint(w, `</div></section><section class="news-latest"><div class="news-section-title"><div><div class="news-article-kicker">ARŞİV</div><h2>Son yayınlar</h2></div><span>Güncel içerikler</span></div>`)
		for _, item := range items {
			_, _ = fmt.Fprintf(w, `<a class="news-row" href="/public/news/%s/%s/html"><span class="news-row-arrow">↗</span><span><strong>%s</strong><small>%s</small></span><time>Rev. %d</time></a>`, template.HTMLEscapeString(locale), template.HTMLEscapeString(item.Slug), template.HTMLEscapeString(item.Title), template.HTMLEscapeString(newsLead(item.Body)), item.PublishedRevision)
		}
		_, _ = fmt.Fprint(w, `</section>`)
	}
	_, _ = fmt.Fprint(w, `</main><footer class="news-footer"><span>HydroGo Newsroom</span><nav><a href="/public/pages/tr/demo/html">Ana sayfa</a><a href="/shop">Mağaza</a></nav></footer></body></html>`)
}

func newsDocumentStart(locale, title string) string {
	return fmt.Sprintf(`<!doctype html><html lang="%s"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s · HydroGo Newsroom</title><link rel="stylesheet" href="/theme/static/theme.css?v=20260910"></head><body><header class="news-header"><a class="news-logo" href="/public/news/%s/html">HYDROGO <span>NEWSROOM</span></a><nav><a href="/public/pages/tr/demo/html">Ana sayfa</a><a href="/public/pages/tr/showcase/html">Showcase</a><a href="/public/pages/tr/about/html">Hakkında</a><a href="/shop">Mağaza</a></nav></header>`, template.HTMLEscapeString(locale), template.HTMLEscapeString(title), template.HTMLEscapeString(locale))
}

func newsLead(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		body = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(ui.RichTextHTML(body), " ")
	}
	body = strings.Join(strings.Fields(body), " ")
	if len([]rune(body)) > 180 {
		return string([]rune(body)[:180]) + "…"
	}
	return body
}
func renderParagraphs(body string) string {
	return ui.RichTextHTML(body)
}

func (h Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
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
	var form bytes.Buffer
	if err := adminstudio.RenderCreateForm(&form, adminstudio.FormData{Collection: h.AdminCollection, Action: "/admin/news/" + item.ID, CSRFToken: security.CSRFTokenFromContext(r.Context()), Nonce: security.NonceFromContext(r.Context()), SubmitLabel: "Taslağı kaydet", Values: map[string]string{"locale": item.Locale, "slug": item.Slug, "title": item.Title, "body": item.Body}}); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Haber düzenle", "Yayın merkezi", security.NonceFromContext(r.Context()), "news"); err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><div class="studio-meta"><div class="meta-item"><strong>%s</strong><span>Durum</span></div><div class="meta-item"><strong>%d</strong><span>Versiyon</span></div><div class="meta-item"><strong>%d</strong><span>Yayın revizyonu</span></div><div class="meta-item"><strong>%s</strong><span>Dil</span></div></div><form class="studio-form" method="post" action="/admin/news/%s">%s<input type="hidden" name="version" value="%d"><button type="submit">Taslağı kaydet</button></form></section><section class="studio-card"><h2>Yayın akışı</h2><p class="muted">Taslağı kaydedin, ileri tarihli yayın planlayın veya anında yayınlayın.</p><div class="studio-actions">`, template.HTMLEscapeString(string(item.Status)), item.Version, item.PublishedRevision, template.HTMLEscapeString(item.Locale), template.HTMLEscapeString(item.ID), template.HTML(form.String()), item.Version)
	_, _ = fmt.Fprintf(w, `<form method="post" action="/admin/news/%s/schedule"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><input name="scheduled_at" type="datetime-local"><button type="submit">Zamanla</button></form><form method="post" action="/admin/news/%s/publish"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><button type="submit">Yayınla</button></form><form method="post" action="/admin/news/%s/retract"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="version" value="%d"><button type="submit" class="button-quiet">Geri çek</button></form></div></section>`, template.HTMLEscapeString(item.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), item.Version, template.HTMLEscapeString(item.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), item.Version, template.HTMLEscapeString(item.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), item.Version)
	_ = ui.AdminEnd(w)
}

func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: r.FormValue("locale"), Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), Status: news.StatusDraft}
	if err := h.Service.UpdateDraft(r.Context(), item, version); err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.HTMLEscapeString(item.ID), http.StatusSeeOther)
}

func (h Handler) adminSchedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	at, err := time.Parse("2006-01-02T15:04", r.FormValue("scheduled_at"))
	if err != nil {
		http.Error(w, "invalid schedule", http.StatusBadRequest)
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), version, at); err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.HTMLEscapeString(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "publish")
}

func (h Handler) adminRetract(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "retract")
}

func (h Handler) adminTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if action == "publish" {
		err = h.Service.Publish(r.Context(), r.PathValue("id"), version)
	} else {
		err = h.Service.Retract(r.Context(), r.PathValue("id"), version)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/news/"+template.HTMLEscapeString(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) write(w http.ResponseWriter, r *http.Request, id string) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && (user.Role == auth.RoleAdmin || user.Role == auth.RoleEditor)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, news.ErrRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	status := http.StatusInternalServerError
	if errors.Is(err, news.ErrInvalidItem) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, news.ErrConflict) || errors.Is(err, news.ErrScheduleStale) {
		status = http.StatusConflict
	}
	http.Error(w, "news operation failed", status)
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
