package adminstudio

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

// FormBuilderData describes the bounded schema editor used by Client Editions.
// Persistence and authorization stay with the owning module.
type FormBuilderData struct {
	Form      forms.Form
	Action    string
	CSRFToken string
	Submit    string
}

// RenderFormBuilder renders a schema-driven editor. The schema travels as a
// JSON data attribute and static/admin/js/form-builder.js keeps the hidden
// schema input in sync, so the page needs no inline script under the CSP.
func RenderFormBuilder(w io.Writer, data FormBuilderData) error {
	return RenderFormBuilderContext(context.Background(), w, data)
}

// RenderFormBuilderContext is RenderFormBuilder with the request context.
func RenderFormBuilderContext(ctx context.Context, w io.Writer, data FormBuilderData) error {
	if err := data.Form.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(data.Action) == "" {
		data.Action = "#"
	}
	if data.Submit == "" {
		data.Submit = "Formu kaydet"
	}
	payload, err := json.Marshal(data.Form)
	if err != nil {
		return err
	}
	return formBuilder(data, string(payload)).Render(ctx, w)
}
