package adminstudio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h GeneratedAdminHandler) edit(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if id == "global" && collection.Type != "global" {
		http.NotFound(w, r)
		return
	}
	apiPath := "/api/v1/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
	response, status, err := h.callAPI(r, http.MethodGet, apiPath, nil)
	if err != nil {
		h.apiError(w, r)
		return
	}
	if status == http.StatusForbidden || !collection.Drafts && status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	if status != http.StatusOK && !(collection.Drafts && status == http.StatusNotFound) {
		h.apiError(w, r)
		return
	}
	record := map[string]any{"id": id}
	publishedVersion := int64(0)
	if status == http.StatusOK {
		if json.Unmarshal(response, &record) != nil {
			h.apiError(w, r)
			return
		}
		publishedVersion = generatedRecordVersion(record)
	}
	if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
		values, version, err := h.Options.GetLocalized(r, collection, id, localeState(r).Current)
		if err != nil || version < 0 || applyLocalizedValues(collection, record, values) != nil {
			h.apiError(w, r)
			return
		}
		record["locale_version"] = version
	}
	if !collection.Drafts && !h.resourceAccess(r, collection, id).Update {
		h.renderReadOnlyRecord(w, r, collection, record)
		return
	}
	if collection.Drafts {
		draft, err := h.Options.GetDraft(r, collection, id, localeState(r).Current)
		if err != nil || draft.ID != "" && draft.ID != id || draft.Version < 0 || draft.PublishedVersion < 0 || draft.PublishedVersion > publishedVersion || draft.Exists && draft.Version < 1 || !draft.Exists && (draft.Version != 0 || len(draft.Fields) != 0) {
			h.apiError(w, r)
			return
		}
		if draft.Exists && applyDraftFields(collection, record, draft.Fields, localeState(r).Current, localeState(r).Options.Default) != nil {
			h.apiError(w, r)
			return
		}
		state := draft.State
		if state == "" {
			state = collection.Workflow.States[0]
		}
		if !slices.Contains(collection.Workflow.States, state) {
			h.apiError(w, r)
			return
		}
		record[collection.Workflow.StateField] = string(state)
		record["draft_state"] = string(state)
		record["draft_version"] = draft.Version
		record["published_version"] = draft.PublishedVersion
		record["draft_exists"] = draft.Exists
		record["draft_updated_at"] = draft.UpdatedAt
		record["published_conflict"] = draft.Exists && draft.PublishedVersion != publishedVersion
		record["_draft"] = draft
	}
	h.renderForm(w, r, collection, id, record, true)
}

func (h GeneratedAdminHandler) renderReadOnlyRecord(w http.ResponseWriter, r *http.Request, collection Collection, record map[string]any) {
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<dl class="studio-record-details">`); err != nil {
			return err
		}
		for _, field := range collection.Fields {
			if !field.VisibleInList {
				continue
			}
			if _, err := fmt.Fprintf(out, `<div><dt>%s</dt><dd>%s</dd></div>`, template.HTMLEscapeString(field.Label), template.HTMLEscapeString(displayFormValue(field, record[field.ID]))); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</dl>`); err != nil {
			return err
		}
		return ui.Link(ui.LinkProps{Href: "/admin/resources/" + url.PathEscape(collection.ID), Label: i18n.T(generatedAdminLocale(r), "admin.generated.form.back_to_list"), Class: "button button-quiet"}).Render(ctx, out)
	})
	card := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		return ui.Card(ui.CardProps{Title: collection.Label + " · " + displayValue(record[collection.TitleField]), Class: "studio-card", Body: body}).Render(ctx, out)
	})
	if err := h.Options.RenderPage(w, r, collection.Label, card); err != nil {
		h.apiError(w, r)
	}
}

func (h GeneratedAdminHandler) renderForm(w http.ResponseWriter, r *http.Request, collection Collection, id string, record map[string]any, editing bool) {
	h.renderFormState(w, r, collection, id, record, editing, nil, nil, 0, "")
}

func collectionField(collection Collection, id string) (Field, bool) {
	for _, field := range collection.Fields {
		if field.ID == id {
			return field, true
		}
	}
	return Field{}, false
}

func (h GeneratedAdminHandler) renderFormState(w http.ResponseWriter, r *http.Request, collection Collection, id string, record map[string]any, editing bool, submitted map[string]string, fieldErrors []ui.FieldValidationError, status int, failureKind string) {
	if h.Options.CSRFToken(r) == "" {
		http.Error(w, "admin page unavailable", http.StatusServiceUnavailable)
		return
	}
	values := make(map[string]string, len(collection.Fields))
	for _, field := range collection.Fields {
		values[field.ID] = displayFormValue(field, record[field.ID])
	}
	for fieldID, value := range submitted {
		if _, ok := collectionField(collection, fieldID); ok || fieldID == "version" || fieldID == "locale_version" || fieldID == "draft_version" || fieldID == "published_version" {
			values[fieldID] = value
		}
	}
	if collection.Versioned {
		if _, ok := submitted["version"]; !ok {
			values["version"] = displayValue(record["version"])
		}
	}
	if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
		if _, ok := submitted["locale_version"]; !ok {
			values["locale_version"] = displayValue(record["locale_version"])
		}
	}
	if collection.Drafts {
		if _, ok := submitted["draft_version"]; !ok {
			values["draft_version"] = displayValue(record["draft_version"])
		}
		if _, ok := submitted["published_version"]; !ok {
			values["published_version"] = displayValue(record["published_version"])
		}
	}
	var revisions []RevisionSummary
	if editing && collection.Revisions {
		var err error
		revisions, err = h.Options.ListRevisions(r, collection, id, 50)
		if err != nil || len(revisions) > 50 {
			h.apiError(w, r)
			return
		}
		seen := make(map[int64]struct{}, len(revisions))
		for _, revision := range revisions {
			if revision.Version < 1 {
				h.apiError(w, r)
				return
			}
			if _, exists := seen[revision.Version]; exists {
				h.apiError(w, r)
				return
			}
			seen[revision.Version] = struct{}{}
		}
	}
	relationOptions := make(map[string][]ui.SelectOption)
	for _, field := range collection.Fields {
		if field.Kind != FieldRelation || field.Relation == nil || field.ReadOnly {
			continue
		}
		options, err := h.loadRelationOptions(r, field, values[field.ID])
		if err != nil {
			h.apiError(w, r)
			return
		}
		relationOptions[field.ID] = options
	}
	action := "/admin/resources/" + url.PathEscape(collection.ID)
	if editing {
		action += "/" + url.PathEscape(id)
		if collection.Drafts {
			action += "/draft"
		} else if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
			action += "/locale"
		}
	} else {
		action += "/new"
		if collection.Type == "global" {
			id = "global"
		}
		if id == "" {
			id = randomAdminID()
			if id == "" {
				http.Error(w, "admin page unavailable", http.StatusInternalServerError)
				return
			}
		}
	}
	if collection.Localization {
		action = localePath(action, localeState(r).Current)
	}
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if err := h.generatedLocaleNav(ctx, out, r, collection, id); err != nil {
			return err
		}
		localeInfo := localeState(r)
		localizedEdit := collection.Localization && localeInfo.Current != localeInfo.Options.Default
		draftEdit := collection.Drafts
		saveAllowed := !draftEdit || h.Options.CanSaveDraft(r, collection, id, localeInfo.Current) && record["published_conflict"] != true
		if draftEdit && record["published_conflict"] == true {
			if _, err := fmt.Fprintf(out, `<p role="alert">%s</p>`, template.HTMLEscapeString(i18n.T(generatedAdminLocale(r), "admin.generated.error.published_conflict"))); err != nil {
				return err
			}
		}
		fields := generatedFormFields(collection, values, relationOptions, id, editing, localeInfo.Current, localeInfo.Options.Default, localizedEdit, draftEdit, saveAllowed, generatedAdminLocale(r), h.Options.BlockEditor)
		formBody := templ.Component(fields)
		if status == http.StatusUnprocessableEntity || len(fieldErrors) > 0 {
			formBody = templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
				if err := ui.ValidationSummary(ui.ValidationSummaryProps{
					Title:   i18n.T(generatedAdminLocale(r), "admin.generated.validation.title"),
					Message: i18n.T(generatedAdminLocale(r), "admin.generated.validation.message"),
					Fields:  fieldErrors,
				}).Render(ctx, out); err != nil {
					return err
				}
				return fields.Render(ctx, out)
			})
		} else if status == http.StatusBadGateway || status == http.StatusConflict {
			formBody = templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
				titleKey, messageKey := "admin.generated.error.title", "admin.generated.error.operation"
				if failureKind == "delete" {
					titleKey, messageKey = "admin.generated.error.delete_title", "admin.generated.error.delete_operation"
				} else if failureKind == "conflict" || status == http.StatusConflict {
					titleKey, messageKey = "admin.generated.error.title", "admin.generated.error.conflict"
				}
				if err := ui.ValidationSummary(ui.ValidationSummaryProps{
					Title:   i18n.T(generatedAdminLocale(r), titleKey),
					Message: i18n.T(generatedAdminLocale(r), messageKey),
				}).Render(ctx, out); err != nil {
					return err
				}
				return fields.Render(ctx, out)
			})
		}
		props := ui.FormProps{Action: action, Method: "post", Class: "studio-form", CSRF: h.Options.CSRFToken(r), Body: formBody}
		if err := ui.Form(props).Render(ctx, out); err != nil {
			return err
		}
		if editing && !collection.Drafts && h.resourceAccess(r, collection, id).Delete && collection.Type != "global" && (!collection.Localization || localeState(r).Current == localeState(r).Options.Default) {
			deleteAction := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/delete"
			if collection.Localization {
				deleteAction = localePath(deleteAction, localeState(r).Current)
			}
			locale := generatedAdminLocale(r)
			if err := ui.Form(ui.FormProps{Action: deleteAction, Method: "post", CSRF: h.Options.CSRFToken(r), ConfirmAction: i18n.T(locale, "admin.generated.form.delete_confirm"), Body: ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.form.delete"), Variant: ui.Danger})}).Render(ctx, out); err != nil {
				return err
			}
		}
		if editing && collection.Workflow != nil && (!collection.Localization || localeState(r).Current == localeState(r).Options.Default || collection.Drafts) {
			if err := h.generatedWorkflowActions(ctx, out, r, collection, id, record); err != nil {
				return err
			}
		}
		if editing && collection.Revisions {
			if err := h.generatedRevisionHistory(ctx, out, r, collection, id, record, revisions); err != nil {
				return err
			}
		}
		if editing && collection.Localization {
			if err := h.generatedCopyLocaleActions(ctx, out, r, collection, id, record); err != nil {
				return err
			}
		}
		if editing && collection.Drafts && record["draft_exists"] == true {
			return h.generatedDiscardDraft(ctx, out, r, collection, id, record)
		}
		return nil
	})
	title := i18n.T(generatedAdminLocale(r), "admin.generated.form.create_title") + " · " + collection.Label
	if editing {
		title = i18n.T(generatedAdminLocale(r), "admin.generated.form.edit_title") + " · " + collection.Label
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	h.renderPage(w, r, title, body)
}

func (h GeneratedAdminHandler) loadRelationOptions(r *http.Request, field Field, selectedValue any) ([]ui.SelectOption, error) {
	selected := relationIDs(selectedValue)
	target, targetRegistered := h.Registry.Collection(field.Relation.Collection)
	titleField := ""
	if targetRegistered {
		titleField = target.TitleField
	}
	query := url.Values{"limit": {"50"}, "offset": {"0"}}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(field.Relation.Collection)+"?"+query.Encode(), nil)
	if err != nil || status != http.StatusOK {
		return nil, errors.New("related resource is unavailable")
	}
	var records []map[string]any
	if json.Unmarshal(response, &records) != nil {
		return nil, errors.New("related resource response is invalid")
	}
	options := make([]ui.SelectOption, 0, min(len(records), 50)+len(selected))
	known := make(map[string]struct{}, cap(options))
	add := func(record map[string]any) {
		id, _ := record["id"].(string)
		if id == "" {
			return
		}
		if _, exists := known[id]; exists {
			return
		}
		label := displayValue(record[titleField])
		if label == "" {
			label = id
		}
		known[id] = struct{}{}
		options = append(options, ui.SelectOption{Value: id, Label: label})
	}
	for _, record := range records[:min(len(records), 50)] {
		add(record)
	}
	for _, id := range selected {
		if _, exists := known[id]; exists {
			continue
		}
		response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(field.Relation.Collection)+"/"+url.PathEscape(id), nil)
		if err != nil || status != http.StatusOK {
			return nil, errors.New("selected relation is unavailable")
		}
		var record map[string]any
		if json.Unmarshal(response, &record) != nil {
			return nil, errors.New("selected relation response is invalid")
		}
		add(record)
	}
	return options, nil
}

func relationIDs(value any) []string {
	switch value := value.(type) {
	case string:
		if value == "" {
			return nil
		}
		return []string{value}
	case []string:
		return append([]string(nil), value...)
	case []any:
		ids := make([]string, 0, len(value))
		for _, item := range value {
			if id, ok := item.(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	default:
		return nil
	}
}

func (h GeneratedAdminHandler) generatedWorkflowActions(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any) error {
	current := PublicationState(displayValue(record[collection.Workflow.StateField]))
	expectedVersion := generatedRecordVersion(record)
	draft := draftForRecord(collection, id, record)
	if collection.Drafts {
		current = draft.State
		expectedVersion = draft.Version
		if !draft.Exists {
			return nil
		}
	}
	if !slices.Contains(collection.Workflow.States, current) || current == "" || expectedVersion < 1 {
		return nil
	}
	allowedTargets := make([]PublicationState, 0, len(collection.Workflow.Transitions[current]))
	for _, target := range collection.Workflow.Transitions[current] {
		allowed := false
		if collection.Drafts {
			allowed = h.Options.CanTransitionDraft(r, collection, draft, target)
		} else {
			allowed = h.Options.CanTransition(r, collection, record, target)
		}
		if allowed {
			allowedTargets = append(allowedTargets, target)
		}
	}
	if len(allowedTargets) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(out, `<section class="studio-card" aria-labelledby="generated-workflow-heading"><h2 id="generated-workflow-heading">%s</h2><div class="studio-actions">`, template.HTMLEscapeString(i18n.T(generatedAdminLocale(r), "admin.generated.workflow.title"))); err != nil {
		return err
	}
	for _, target := range allowedTargets {
		body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "target", Value: string(target)}).Render(ctx, form); err != nil {
				return err
			}
			versionName := "expected_version"
			if collection.Drafts {
				versionName = "expected_draft_version"
			}
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: versionName, Value: strconv.FormatInt(expectedVersion, 10)}).Render(ctx, form); err != nil {
				return err
			}
			if collection.Drafts {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_published_version", Value: strconv.FormatInt(draft.PublishedVersion, 10)}).Render(ctx, form); err != nil {
					return err
				}
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: workflowActionLabel(generatedAdminLocale(r), target), Variant: ui.Primary}).Render(ctx, form)
		})
		action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/workflow"
		if collection.Localization {
			action = localePath(action, localeState(r).Current)
		}
		if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: body}).Render(ctx, out); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</div></section>`)
	return err
}

func draftForRecord(collection Collection, id string, record map[string]any) DraftDocument {
	if draft, ok := record["_draft"].(DraftDocument); ok {
		return draft
	}
	return DraftDocument{ID: id, Version: recordVersion(record["draft_version"]), PublishedVersion: recordVersion(record["published_version"]), State: PublicationState(displayValue(record[collection.Workflow.StateField])), Exists: record["draft_exists"] == true}
}

func (h GeneratedAdminHandler) generatedDiscardDraft(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any) error {
	draft := draftForRecord(collection, id, record)
	if !draft.Exists || !h.Options.CanDiscardDraft(r, collection, draft) {
		return nil
	}
	body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
		if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(draft.Version, 10)}).Render(ctx, form); err != nil {
			return err
		}
		return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(generatedAdminLocale(r), "admin.generated.form.discard_draft"), Variant: ui.Danger}).Render(ctx, form)
	})
	action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/draft/discard"
	if collection.Localization {
		action = localePath(action, localeState(r).Current)
	}
	return ui.Form(ui.FormProps{Action: action, Method: http.MethodPost, CSRF: h.Options.CSRFToken(r), ConfirmAction: i18n.T(generatedAdminLocale(r), "admin.generated.form.discard_draft_confirm"), Body: body}).Render(ctx, out)
}

func (h GeneratedAdminHandler) generatedRevisionHistory(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any, revisions []RevisionSummary) error {
	locale := generatedAdminLocale(r)
	if _, err := fmt.Fprintf(out, `<section class="studio-card" aria-labelledby="generated-revisions-heading"><h2 id="generated-revisions-heading">%s</h2>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.revisions.title"))); err != nil {
		return err
	}
	if len(revisions) == 0 {
		if _, err := fmt.Fprintf(out, `<p role="status">%s</p></section>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.revisions.empty"))); err != nil {
			return err
		}
		return nil
	}
	revisions = append([]RevisionSummary(nil), revisions...)
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].Version > revisions[j].Version })
	if _, err := io.WriteString(out, `<ol class="revision-list">`); err != nil {
		return err
	}
	currentVersion := generatedRecordVersion(record)
	for _, revision := range revisions {
		href := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/revisions/" + strconv.FormatInt(revision.Version, 10)
		if _, err := io.WriteString(out, `<li><div class="revision-list-item">`); err != nil {
			return err
		}
		label := fmt.Sprintf(i18n.T(locale, "admin.generated.revisions.version"), revision.Version)
		visit := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if collection.Localization {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: localeState(r).Current}).Render(ctx, form); err != nil {
					return err
				}
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Quiet}).Render(ctx, form)
		})
		if err := ui.Form(ui.FormProps{Action: href, Method: http.MethodGet, Body: visit}).Render(ctx, out); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `<p>`+template.HTMLEscapeString(revisionSummaryText(locale, revision))+`</p></div>`); err != nil {
			return err
		}
		if revision.Version != currentVersion && h.Options.CanRestore(r, collection, id, revision.Version) {
			body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(currentVersion, 10)}).Render(ctx, form); err != nil {
					return err
				}
				return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.revisions.restore"), Variant: ui.Quiet}).Render(ctx, form)
			})
			action := href + "/restore"
			if collection.Localization {
				action = localePath(action, localeState(r).Current)
			}
			if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: body}).Render(ctx, out); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</li>`); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</ol></section>`)
	return err
}

func revisionSummaryText(locale string, revision RevisionSummary) string {
	parts := make([]string, 0, 3)
	if !revision.CreatedAt.IsZero() {
		parts = append(parts, revision.CreatedAt.Local().Format("02.01.2006 15:04"))
	}
	if strings.TrimSpace(revision.Actor) != "" {
		parts = append(parts, strings.TrimSpace(revision.Actor))
	}
	if strings.TrimSpace(revision.State) != "" {
		parts = append(parts, generatedStatusLabel(locale, revision.State))
	}
	return strings.Join(parts, " · ")
}
