package adminstudio

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"

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
	// The admin shell lists the content types in its sidebar itself.
	if err := h.Options.RenderPage(w, r, title, body); err != nil {
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
