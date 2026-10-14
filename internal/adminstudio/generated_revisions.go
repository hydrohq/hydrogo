package adminstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h GeneratedAdminHandler) revision(w http.ResponseWriter, r *http.Request, collection Collection, id, rawVersion string) {
	if !collection.Revisions || h.Options.GetRevision == nil {
		http.NotFound(w, r)
		return
	}
	version, err := strconv.ParseInt(rawVersion, 10, 64)
	if err != nil || version < 1 {
		http.NotFound(w, r)
		return
	}
	revision, err := h.Options.GetRevision(r, collection, id, version)
	if err != nil || revision.Version != version || len(revision.Fields) > len(collection.Fields) {
		http.NotFound(w, r)
		return
	}
	snapshotJSON, err := json.Marshal(revision.Fields)
	if err != nil || len(snapshotJSON) > generatedAdminBodyLimit {
		http.Error(w, "revision unavailable", http.StatusRequestEntityTooLarge)
		return
	}
	current, err := h.currentResource(r, collection, id)
	if err != nil {
		h.apiError(w, r)
		return
	}
	before, after := make(map[string]string), make(map[string]string)
	labels := make(map[string]string)
	for _, field := range collection.Fields {
		if field.ReadOnly {
			continue
		}
		labels[field.ID] = field.Label
		if value, ok := revision.Fields[field.ID]; ok {
			before[field.ID] = value
		}
		after[field.ID] = displayValue(current[field.ID])
	}
	for fieldID := range revision.Fields {
		if _, ok := labels[fieldID]; !ok {
			http.NotFound(w, r)
			return
		}
	}
	diffs := CompareFields(before, after)
	locale := generatedAdminLocale(r)
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		content := templ.ComponentFunc(func(_ context.Context, body io.Writer) error {
			if len(diffs) == 0 {
				_, err := fmt.Fprintf(body, `<p role="status">%s</p>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.revisions.no_changes")))
				return err
			}
			if _, err := io.WriteString(body, `<dl class="revision-diff-list">`); err != nil {
				return err
			}
			for _, diff := range diffs {
				label := labels[diff.Field]
				if label == "" {
					continue
				}
				if _, err := fmt.Fprintf(body, `<div><dt>%s</dt><dd><del>%s</del><br><ins>%s</ins></dd></div>`, template.HTMLEscapeString(label), template.HTMLEscapeString(diff.Before), template.HTMLEscapeString(diff.After)); err != nil {
					return err
				}
			}
			_, err := io.WriteString(body, `</dl>`)
			return err
		})
		if err := ui.Card(ui.CardProps{Title: fmt.Sprintf(i18n.T(locale, "admin.generated.revisions.detail_title"), revision.Version), Class: "studio-card", Body: content}).Render(ctx, out); err != nil {
			return err
		}
		if h.Options.CanRestore(r, collection, id, version) {
			expectedVersion := generatedRecordVersion(current)
			if expectedVersion > 0 {
				formBody := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
					if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(expectedVersion, 10)}).Render(ctx, form); err != nil {
						return err
					}
					return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.revisions.restore"), Variant: ui.Primary}).Render(ctx, form)
				})
				action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/revisions/" + strconv.FormatInt(version, 10) + "/restore"
				if collection.Localization {
					action = localePath(action, localeState(r).Current)
				}
				if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: formBody}).Render(ctx, out); err != nil {
					return err
				}
			}
		}
		back := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
		backBody := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if collection.Localization {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: localeState(r).Current}).Render(ctx, form); err != nil {
					return err
				}
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.form.back_to_record"), Variant: ui.Quiet}).Render(ctx, form)
		})
		return ui.Form(ui.FormProps{Action: back, Method: http.MethodGet, Body: backBody}).Render(ctx, out)
	})
	h.renderPage(w, r, i18n.T(locale, "admin.generated.revisions.title")+" · "+collection.Label, body)
}

func (h GeneratedAdminHandler) restoreRevision(w http.ResponseWriter, r *http.Request, collection Collection, id, rawVersion string) {
	if !collection.Revisions || h.Options.RestoreRevision == nil || h.Options.CanRestore == nil {
		http.NotFound(w, r)
		return
	}
	revisionVersion, err := strconv.ParseInt(rawVersion, 10, 64)
	if err != nil || revisionVersion < 1 {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != 2 || len(r.PostForm["expected_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	expectedVersion, err := strconv.ParseInt(r.PostFormValue("expected_version"), 10, 64)
	if err != nil || expectedVersion < 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	current, err := h.currentResource(r, collection, id)
	if err != nil {
		h.apiError(w, r)
		return
	}
	if generatedRecordVersion(current) != expectedVersion {
		http.Error(w, "document changed; reload before restoring", http.StatusConflict)
		return
	}
	if !h.Options.CanRestore(r, collection, id, revisionVersion) {
		http.Error(w, "restore unavailable", http.StatusForbidden)
		return
	}
	if err := h.Options.RestoreRevision(r, collection, id, revisionVersion, expectedVersion); err != nil {
		http.Error(w, "restore unavailable", http.StatusConflict)
		return
	}
	redirect := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
	if collection.Localization {
		redirect = localePath(redirect, localeState(r).Current)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
