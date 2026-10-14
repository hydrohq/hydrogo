package adminstudio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h GeneratedAdminHandler) callAPI(r *http.Request, method, target string, body []byte) ([]byte, int, error) {
	req := r.Clone(r.Context())
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, 0, err
	}
	req.Method, req.URL = method, parsed
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	recorder := newBoundedRecorder(generatedAdminBodyLimit)
	h.Options.API.ServeHTTP(recorder, req)
	return recorder.body.Bytes(), recorder.status, recorder.err
}

type boundedResponseRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
	limit  int
	err    error
}

func newBoundedRecorder(limit int) *boundedResponseRecorder {
	return &boundedResponseRecorder{header: make(http.Header), limit: limit}
}

func (r *boundedResponseRecorder) Header() http.Header { return r.header }

func (r *boundedResponseRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

func (r *boundedResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if r.body.Len()+len(data) > r.limit {
		r.err = errors.New("generated Admin API response exceeds limit")
		return 0, r.err
	}
	return r.body.Write(data)
}

func (h GeneratedAdminHandler) renderPage(w http.ResponseWriter, r *http.Request, title string, body templ.Component) {
	h.renderPageStatus(w, r, title, body, 0)
}

func (h GeneratedAdminHandler) renderPageStatus(w http.ResponseWriter, r *http.Request, title string, body templ.Component, status int) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != 0 {
		w.WriteHeader(status)
	}
	locale := generatedAdminLocale(r)
	navigation := &generatedNavigationWriter{ResponseWriter: w, items: h.visibleCollections(r), current: r.URL.Path, locale: locale}
	if err := h.Options.RenderPage(navigation, r, title, body); err != nil {
		http.Error(w, "admin page unavailable", http.StatusInternalServerError)
	}
}

func generatedAdminLocale(r *http.Request) string {
	if r != nil {
		if cookie, err := r.Cookie("hydro_admin_locale"); err == nil && i18n.IsSupportedCatalogLocale(cookie.Value) {
			return i18n.ResolveCatalogLocale(cookie.Value)
		}
	}
	return i18n.ResolveCatalogLocale("")
}

type generatedNavigationWriter struct {
	http.ResponseWriter
	items   []Collection
	current string
	locale  string
	wrote   bool
}

func (w *generatedNavigationWriter) Write(payload []byte) (int, error) {
	if !w.wrote && len(w.items) > 0 {
		var err error
		payload, err = injectGeneratedNavigation(payload, w.items, w.current, w.locale)
		if err != nil {
			return 0, err
		}
		w.wrote = true
	}
	return w.ResponseWriter.Write(payload)
}

func injectGeneratedNavigation(markup []byte, collections []Collection, current, locale string) ([]byte, error) {
	items := make([]ui.SidebarItem, 0, len(collections))
	for _, collection := range collections {
		if collection.Hidden {
			continue
		}
		group := strings.TrimSpace(collection.Group)
		if group == "" || strings.EqualFold(group, "content") {
			group = "İÇERİK"
			if locale == "en" {
				group = "CONTENT"
			}
		}
		route := "/admin/resources/" + url.PathEscape(collection.ID)
		icon := collection.Icon
		if icon == "" {
			icon = "collection"
		}
		items = append(items, ui.SidebarItem{Route: route, Label: collection.Label, Icon: icon, Group: group, Active: current == route || strings.HasPrefix(current, route+"/")})
	}
	if len(items) == 0 {
		return markup, nil
	}
	var sidebar bytes.Buffer
	if err := ui.SidebarItems(items).Render(context.Background(), &sidebar); err != nil {
		return nil, err
	}
	navigation := sidebar.String()
	result := string(markup)
	if end := strings.Index(result, `</nav>`); end >= 0 {
		result = result[:end] + navigation + result[end:]
	}
	return []byte(result), nil
}

func (h GeneratedAdminHandler) apiError(w http.ResponseWriter, r *http.Request) {
	locale := generatedAdminLocale(r)
	summary := &ui.ValidationSummaryProps{
		Title:   i18n.T(locale, "admin.generated.error.load_title"),
		Message: i18n.T(locale, "admin.generated.error.load_message"),
	}
	retry := ui.Form(ui.FormProps{
		Action: r.URL.Path,
		Method: http.MethodGet,
		Class:  "generated-load-retry",
		Body:   ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.error.retry"), Variant: ui.Primary}),
	})
	body := generatedFormBodyView(summary, retry)
	h.renderPageStatus(w, r, i18n.T(locale, "admin.generated.error.load_title"), body, http.StatusBadGateway)
}
