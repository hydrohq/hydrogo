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

const pageTagsLabel = "İçerik etiketleri (virgülle ayrılmış)"

// previewFrameSandbox keeps scripts, forms and navigation disabled in page
// preview frames. allow-same-origin only lets the frame load the admin's own
// stylesheet and media: the server's same-origin resource policy refuses them
// to an opaque origin, and without allow-scripts nothing in the frame can use
// that origin.
const previewFrameSandbox = "allow-same-origin"

// pageListView is the view model of /admin/pages.
type pageListView struct {
	// ImportLabel is empty for users who may not import content.
	ImportLabel string
	ViewSort    *designsystem.ContentViewSortProps
	List        templ.Component
}

// pageNewView is the view model of /admin/pages/new.
type pageNewView struct {
	Layouts     designsystem.PageLayoutPickerProps
	Fields      templ.Component
	SubmitLabel string
}

// pageEditView is the view model of the page editor. Before and After hold
// the optional panels around the form (presence, previews, revision history,
// feedback, AI assistant) in display order.
type pageEditView struct {
	ID, Action, PublishAction, CSRF string
	Version, BlockCount             int
	Fields                          templ.Component
	Blocks                          blockEditorView
	Before, After                   []templ.Component
}

// pageReadOnlyView is the view model of the read-only page detail.
type pageReadOnlyView struct {
	Title, Preview string
	Fields         []uicomponents.InputProps
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
		http.Error(w, "page admin unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, title, eyebrow, "pages"); err != nil {
		return
	}
	_, _ = content.WriteTo(w)
	_ = ui.AdminEnd(w)
}

func renderString(ctx context.Context, component templ.Component) (string, error) {
	var out bytes.Buffer
	if err := component.Render(ctx, &out); err != nil {
		return "", err
	}
	return out.String(), nil
}
