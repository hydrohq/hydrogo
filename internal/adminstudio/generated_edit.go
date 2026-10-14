package adminstudio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

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
	rows := make([]generatedDetailRow, 0, len(collection.Fields))
	for _, field := range collection.Fields {
		if field.VisibleInList {
			rows = append(rows, generatedDetailRow{Label: field.Label, Value: displayFormValue(field, record[field.ID])})
		}
	}
	back := ui.LinkProps{Href: "/admin/resources/" + url.PathEscape(collection.ID), Label: i18n.T(generatedAdminLocale(r), "admin.generated.form.back_to_list"), Class: "button button-quiet"}
	card := ui.Card(ui.CardProps{Title: collection.Label + " · " + displayValue(record[collection.TitleField]), Class: "studio-card", Body: generatedRecordDetailsView(rows, back)})
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
	uiLocale := generatedAdminLocale(r)
	localeInfo := localeState(r)
	localizedEdit := collection.Localization && localeInfo.Current != localeInfo.Options.Default
	draftEdit := collection.Drafts
	saveAllowed := !draftEdit || h.Options.CanSaveDraft(r, collection, id, localeInfo.Current) && record["published_conflict"] != true
	fields := generatedFormFields(collection, values, relationOptions, id, editing, localeInfo.Current, localeInfo.Options.Default, localizedEdit, draftEdit, saveAllowed, uiLocale, h.Options.BlockEditor)
	var summary *ui.ValidationSummaryProps
	if status == http.StatusUnprocessableEntity || len(fieldErrors) > 0 {
		summary = &ui.ValidationSummaryProps{
			Title:   i18n.T(uiLocale, "admin.generated.validation.title"),
			Message: i18n.T(uiLocale, "admin.generated.validation.message"),
			Fields:  fieldErrors,
		}
	} else if status == http.StatusBadGateway || status == http.StatusConflict {
		titleKey, messageKey := "admin.generated.error.title", "admin.generated.error.operation"
		if failureKind == "delete" {
			titleKey, messageKey = "admin.generated.error.delete_title", "admin.generated.error.delete_operation"
		} else if failureKind == "conflict" || status == http.StatusConflict {
			titleKey, messageKey = "admin.generated.error.title", "admin.generated.error.conflict"
		}
		summary = &ui.ValidationSummaryProps{Title: i18n.T(uiLocale, titleKey), Message: i18n.T(uiLocale, messageKey)}
	}
	view := generatedFormPage{
		LocaleNav: h.generatedLocaleNav(r, collection),
		Form:      ui.FormProps{Action: action, Method: "post", Class: "studio-form", CSRF: h.Options.CSRFToken(r), Body: generatedFormBodyView(summary, fields)},
	}
	if draftEdit && record["published_conflict"] == true {
		view.ConflictMessage = i18n.T(uiLocale, "admin.generated.error.published_conflict")
	}
	defaultLocale := !collection.Localization || localeInfo.Current == localeInfo.Options.Default
	if editing && !collection.Drafts && h.resourceAccess(r, collection, id).Delete && collection.Type != "global" && defaultLocale {
		deleteAction := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/delete"
		if collection.Localization {
			deleteAction = localePath(deleteAction, localeInfo.Current)
		}
		form := postForm(deleteAction, h.Options.CSRFToken(r), nil, i18n.T(uiLocale, "admin.generated.form.delete"), ui.Danger)
		form.ConfirmAction = i18n.T(uiLocale, "admin.generated.form.delete_confirm")
		view.Delete = &form
	}
	if editing && collection.Workflow != nil && (defaultLocale || collection.Drafts) {
		view.Workflow = h.generatedWorkflowActions(r, collection, id, record)
	}
	if editing && collection.Revisions {
		view.Revisions = h.generatedRevisionHistory(r, collection, id, record, revisions)
	}
	if editing && collection.Localization {
		view.CopyLocale = h.generatedCopyLocaleActions(r, collection, id, record)
	}
	if editing && collection.Drafts && record["draft_exists"] == true {
		view.Discard = h.generatedDiscardDraft(r, collection, id, record)
	}
	title := i18n.T(generatedAdminLocale(r), "admin.generated.form.create_title") + " · " + collection.Label
	if editing {
		title = i18n.T(generatedAdminLocale(r), "admin.generated.form.edit_title") + " · " + collection.Label
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	h.renderPage(w, r, title, generatedFormPageView(view))
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

// generatedWorkflowActions lists the transitions the user may apply to the
// record (or its draft), or nil when there are none.
func (h GeneratedAdminHandler) generatedWorkflowActions(r *http.Request, collection Collection, id string, record map[string]any) *generatedActionGroup {
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
	locale := generatedAdminLocale(r)
	action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/workflow"
	if collection.Localization {
		action = localePath(action, localeState(r).Current)
	}
	group := &generatedActionGroup{Title: i18n.T(locale, "admin.generated.workflow.title")}
	for _, target := range collection.Workflow.Transitions[current] {
		allowed := false
		if collection.Drafts {
			allowed = h.Options.CanTransitionDraft(r, collection, draft, target)
		} else {
			allowed = h.Options.CanTransition(r, collection, record, target)
		}
		if !allowed {
			continue
		}
		hidden := []ui.HiddenInputProps{{Name: "target", Value: string(target)}}
		if collection.Drafts {
			hidden = append(hidden,
				ui.HiddenInputProps{Name: "expected_draft_version", Value: strconv.FormatInt(expectedVersion, 10)},
				ui.HiddenInputProps{Name: "expected_published_version", Value: strconv.FormatInt(draft.PublishedVersion, 10)})
		} else {
			hidden = append(hidden, ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(expectedVersion, 10)})
		}
		group.Forms = append(group.Forms, postForm(action, h.Options.CSRFToken(r), hidden, workflowActionLabel(locale, target), ui.Primary))
	}
	if len(group.Forms) == 0 {
		return nil
	}
	return group
}

func draftForRecord(collection Collection, id string, record map[string]any) DraftDocument {
	if draft, ok := record["_draft"].(DraftDocument); ok {
		return draft
	}
	return DraftDocument{ID: id, Version: recordVersion(record["draft_version"]), PublishedVersion: recordVersion(record["published_version"]), State: PublicationState(displayValue(record[collection.Workflow.StateField])), Exists: record["draft_exists"] == true}
}

// generatedDiscardDraft is the discard-draft form, or nil when the draft
// cannot be discarded.
func (h GeneratedAdminHandler) generatedDiscardDraft(r *http.Request, collection Collection, id string, record map[string]any) *ui.ActionFormProps {
	draft := draftForRecord(collection, id, record)
	if !draft.Exists || !h.Options.CanDiscardDraft(r, collection, draft) {
		return nil
	}
	action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/draft/discard"
	if collection.Localization {
		action = localePath(action, localeState(r).Current)
	}
	locale := generatedAdminLocale(r)
	form := postForm(action, h.Options.CSRFToken(r), []ui.HiddenInputProps{{Name: "expected_version", Value: strconv.FormatInt(draft.Version, 10)}}, i18n.T(locale, "admin.generated.form.discard_draft"), ui.Danger)
	form.Method = http.MethodPost
	form.ConfirmAction = i18n.T(locale, "admin.generated.form.discard_draft_confirm")
	return &form
}

// generatedRevisionHistory lists the record's revisions, newest first, each
// with a link to its diff and, where allowed, a restore form.
func (h GeneratedAdminHandler) generatedRevisionHistory(r *http.Request, collection Collection, id string, record map[string]any, revisions []RevisionSummary) *generatedRevisionHistory {
	locale := generatedAdminLocale(r)
	history := &generatedRevisionHistory{Title: i18n.T(locale, "admin.generated.revisions.title"), Empty: i18n.T(locale, "admin.generated.revisions.empty")}
	revisions = append([]RevisionSummary(nil), revisions...)
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].Version > revisions[j].Version })
	currentVersion := generatedRecordVersion(record)
	for _, revision := range revisions {
		href := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/revisions/" + strconv.FormatInt(revision.Version, 10)
		item := generatedRevisionItem{
			Open:    getForm(href, localeHidden(collection.Localization, localeState(r).Current), fmt.Sprintf(i18n.T(locale, "admin.generated.revisions.version"), revision.Version)),
			Summary: revisionSummaryText(locale, revision),
		}
		if revision.Version != currentVersion && h.Options.CanRestore(r, collection, id, revision.Version) {
			action := href + "/restore"
			if collection.Localization {
				action = localePath(action, localeState(r).Current)
			}
			restore := postForm(action, h.Options.CSRFToken(r), []ui.HiddenInputProps{{Name: "expected_version", Value: strconv.FormatInt(currentVersion, 10)}}, i18n.T(locale, "admin.generated.revisions.restore"), ui.Quiet)
			item.Restore = &restore
		}
		history.Items = append(history.Items, item)
	}
	return history
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
