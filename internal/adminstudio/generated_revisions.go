package adminstudio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

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
	rows := make([]generatedDiffRow, 0, len(diffs))
	for _, diff := range diffs {
		if label := labels[diff.Field]; label != "" {
			rows = append(rows, generatedDiffRow{Label: label, Before: diff.Before, After: diff.After})
		}
	}
	card := ui.CardProps{Title: fmt.Sprintf(i18n.T(locale, "admin.generated.revisions.detail_title"), revision.Version), Class: "studio-card", Body: generatedRevisionDiffView(len(diffs) > 0, rows, i18n.T(locale, "admin.generated.revisions.no_changes"))}
	var restore *ui.ActionFormProps
	if h.Options.CanRestore(r, collection, id, version) {
		if expectedVersion := generatedRecordVersion(current); expectedVersion > 0 {
			action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/revisions/" + strconv.FormatInt(version, 10) + "/restore"
			if collection.Localization {
				action = localePath(action, localeState(r).Current)
			}
			form := postForm(action, h.Options.CSRFToken(r), []ui.HiddenInputProps{{Name: "expected_version", Value: strconv.FormatInt(expectedVersion, 10)}}, i18n.T(locale, "admin.generated.revisions.restore"), ui.Primary)
			restore = &form
		}
	}
	back := getForm("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), localeHidden(collection.Localization, localeState(r).Current), i18n.T(locale, "admin.generated.form.back_to_record"))
	body := generatedRevisionPageView(card, restore, back)
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
