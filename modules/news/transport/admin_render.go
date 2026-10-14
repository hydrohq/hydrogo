package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

// untitledNews is shown in the editor header while a story has no title; the
// live-title script (static/admin/js/editor-live-title.js) uses the same text.
const untitledNews = "[Başlıksız]"

// newsListView is the view model of /admin/news.
type newsListView struct {
	ViewSort *designsystem.ContentViewSortProps
	List     templ.Component
}

// newsNewView is the view model of /admin/news/new.
type newsNewView struct {
	Locale, CSRF       string
	Content, Meta, SEO templ.Component
}

// newsEditView is the view model of the news editor.
type newsEditView struct {
	ID, Action, Title, Status, Locale, CSRF string
	Version                                 int
	// LiveURL is set for published stories and offered as "copy URL".
	LiveURL, PreviewEndpoint string
	Presence                 templ.Component
	Content, Meta, SEO       templ.Component
	// Taxonomy is nil when the repository has no taxonomy support.
	Taxonomy templ.Component
	Workflow newsWorkflowView
}

// newsWorkflowView holds the editorial transitions offered for a story.
type newsWorkflowView struct {
	Base, CSRF                          string
	Version                             int
	CanSchedule, CanPublish, CanRetract bool
}

// newsReadOnlyView is the view model of the read-only story detail.
type newsReadOnlyView struct {
	Fields   []uicomponents.InputProps
	BodyHTML string
}

// writerComponent adapts a writer-based renderer (adminstudio forms and
// lists) to a templ.Component.
func writerComponent(render func(io.Writer) error) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error { return render(w) })
}

// renderAdminScreen renders body inside the admin shell. The body is rendered
// first so a failing view yields an error status instead of a broken page.
func renderAdminScreen(w http.ResponseWriter, r *http.Request, title, eyebrow string, body templ.Component) {
	var content bytes.Buffer
	if err := body.Render(r.Context(), &content); err != nil {
		http.Error(w, "news admin unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, title, eyebrow, "news"); err != nil {
		return
	}
	_, _ = content.WriteTo(w)
	_ = ui.AdminEnd(w)
}
