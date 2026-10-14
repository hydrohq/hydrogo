package transport

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func (h Handler) adminNewPage(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	layouts := append(pages.BuiltinLayoutTemplates(), h.LayoutTemplates...)
	if err := pages.ValidateLayoutTemplates(layouts); err != nil {
		http.Error(w, "page templates unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := ui.AdminLocaleFromRequest(r)
	options := make([]designsystem.PageLayoutOption, 0, len(layouts))
	for _, layout := range layouts {
		name, description := pageLayoutCopy(layout, locale)
		options = append(options, designsystem.PageLayoutOption{ID: layout.ID, Name: name, Description: description})
	}
	view := pageNewView{Layouts: designsystem.PageLayoutPickerProps{Options: options, Legend: "Sayfa düzeni", EmptyMessage: "Kullanılabilir sayfa düzeni yok."}, SubmitLabel: "Taslak oluştur"}
	if locale == "en" {
		view.Layouts.Legend, view.Layouts.EmptyMessage, view.SubmitLabel = "Page layout", "No layout options available.", "Create draft"
	}
	view.Fields = writerComponent(func(out io.Writer) error {
		return adminstudio.RenderCreateForm(out, adminstudio.FormData{
			Collection: h.AdminCollection,
			Action:     "/admin/pages",
			CSRFToken:  security.CSRFTokenFromContext(r.Context()),
			Values:     map[string]string{"locale": "tr"},
			HideSubmit: true,
			UILocale:   locale,
		})
	})
	renderAdminScreen(w, r, "Yeni sayfa", "Sayfa yönetimi", pageNewScreen(view))
}

func pageLayoutCopy(layout pages.LayoutTemplate, locale string) (string, string) {
	if locale == "en" {
		return layout.Name, layout.Description
	}
	switch layout.ID {
	case "blank":
		return "Boş sayfa", "Sıfırdan başlayıp içeriğinizi oluşturun."
	case "service_showcase":
		return "Hizmet tanıtımı", "Hizmetinizi ve sunduğu faydaları tanıtın."
	case "landing":
		return "Açılış sayfası", "Ziyaretçileri net bir mesaj ve eylemle karşılayın."
	case "case_study":
		return "Başarı hikâyesi", "İhtiyacı, sonucu ve müşteri deneyimini anlatın."
	default:
		return layout.Name, layout.Description
	}
}

func (h Handler) adminCreateForm(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	templateID := r.FormValue("layout_template")
	if templateID == "" {
		templateID = "blank"
	}
	blocks, err := pages.ResolveLayoutTemplate(templateID, h.LayoutTemplates)
	if err != nil {
		http.Error(w, "invalid page layout", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "page service unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := r.FormValue("locale")
	if locale == "" {
		locale = "tr"
	}
	page := pages.Page{
		ID:     id,
		Locale: locale,
		Slug:   r.FormValue("slug"),
		Title:  r.FormValue("title"),
		Body:   r.FormValue("body"),
		Blocks: blocks,
		Tags:   strings.Split(r.FormValue("tags"), ","),
		Status: pages.StatusDraft,
	}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Content.CreateDraft(r.Context(), page); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(id), http.StatusSeeOther)
}
