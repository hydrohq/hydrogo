package adminstudio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h GeneratedAdminHandler) mutateDraft(w http.ResponseWriter, r *http.Request, collection Collection, id string, creating bool) {
	if !collection.Drafts {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if creating {
		id = strings.TrimSpace(r.PostFormValue("id"))
	}
	if !draftIDPattern.MatchString(id) {
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	draftVersions, publishedVersions := r.PostForm["draft_version"], r.PostForm["published_version"]
	if len(draftVersions) != 1 || len(publishedVersions) != 1 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	expectedDraft, err := strconv.ParseInt(draftVersions[0], 10, 64)
	if err != nil || expectedDraft < 0 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	expectedPublished, err := strconv.ParseInt(publishedVersions[0], 10, 64)
	if err != nil || expectedPublished < 0 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	fields := make(map[string]any, len(collection.Fields))
	allowed := map[string]struct{}{"_csrf": {}, "draft_version": {}, "published_version": {}}
	if creating {
		allowed["id"] = struct{}{}
	}
	for _, field := range collection.Fields {
		if collection.Workflow != nil && field.ID == collection.Workflow.StateField {
			allowed[field.ID] = struct{}{}
			continue
		}
		if field.ReadOnly {
			continue
		}
		if collection.Localization && localeState(r).Current != localeState(r).Options.Default && !field.Localized {
			continue
		}
		allowed[field.ID] = struct{}{}
		values, exists := r.PostForm[field.ID]
		if !exists {
			continue
		}
		value, err := parseAdminField(field, values)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		fields[field.ID] = value
	}
	if creating {
		fields[collection.Workflow.StateField] = string(collection.Workflow.States[0])
	}
	for key, values := range r.PostForm {
		if _, ok := allowed[key]; !ok || len(values) != 1 {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
	}
	if !h.Options.CanSaveDraft(r, collection, id, localeState(r).Current) {
		http.Error(w, "draft editing unavailable", http.StatusForbidden)
		return
	}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
	publishedVersion := int64(0)
	if err != nil || status == http.StatusOK && func() bool {
		var published map[string]any
		if json.Unmarshal(response, &published) != nil {
			return true
		}
		publishedVersion = generatedRecordVersion(published)
		return false
	}() || status != http.StatusOK && status != http.StatusNotFound {
		h.apiError(w, r)
		return
	}
	if publishedVersion != expectedPublished || creating && publishedVersion != 0 {
		http.Error(w, "published document changed; reload before saving", http.StatusConflict)
		return
	}
	if err := h.Options.SaveDraft(r, collection, id, localeState(r).Current, fields, expectedDraft, expectedPublished); err != nil {
		http.Error(w, "draft could not be saved", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), localeState(r).Current), http.StatusSeeOther)
}

func (h GeneratedAdminHandler) discardDraft(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if !collection.Drafts {
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
	expected, err := strconv.ParseInt(r.PostFormValue("expected_version"), 10, 64)
	if err != nil || expected < 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	draft, err := h.Options.GetDraft(r, collection, id, localeState(r).Current)
	if err != nil || !draft.Exists || draft.ID != id || draft.Version != expected {
		http.Error(w, "draft changed; reload before discarding", http.StatusConflict)
		return
	}
	if !h.Options.CanDiscardDraft(r, collection, draft) {
		http.Error(w, "draft discard unavailable", http.StatusForbidden)
		return
	}
	if err := h.Options.DiscardDraft(r, collection, id, localeState(r).Current, expected); err != nil {
		http.Error(w, "draft discard unavailable", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/drafts", localeState(r).Current), http.StatusSeeOther)
}

func workflowActionLabel(locale string, state PublicationState) string {
	key := "admin.generated.workflow.change_state"
	switch state {
	case StateDraft:
		key = "admin.generated.workflow.draft"
	case StateScheduled:
		key = "admin.generated.workflow.schedule"
	case StatePublished:
		key = "admin.generated.workflow.publish"
	case StateRetracted:
		key = "admin.generated.workflow.retract"
	}
	label := i18n.T(locale, key)
	if state != StateDraft && state != StateScheduled && state != StatePublished && state != StateRetracted {
		return fmt.Sprintf(label, state)
	}
	return label
}

func generatedStatusLabel(locale, status string) string {
	status = strings.TrimSpace(status)
	key := "admin.generated.drafts.state." + strings.ToLower(status)
	label := i18n.T(locale, key)
	if label == key {
		return status
	}
	return label
}

func (h GeneratedAdminHandler) transition(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if collection.Workflow == nil || collection.Drafts && (h.Options.TransitionDraft == nil || h.Options.CanTransitionDraft == nil) || !collection.Drafts && (h.Options.Transition == nil || h.Options.CanTransition == nil) || collection.Localization && !collection.Drafts && localeState(r).Current != localeState(r).Options.Default {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	versionKey, expectedFields := "expected_version", 3
	if collection.Drafts {
		versionKey, expectedFields = "expected_draft_version", 4
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != expectedFields || len(r.PostForm["target"]) != 1 || len(r.PostForm[versionKey]) != 1 || collection.Drafts && len(r.PostForm["expected_published_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	target := PublicationState(r.PostFormValue("target"))
	if !slices.Contains(collection.Workflow.States, target) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
	if err != nil || status != http.StatusOK {
		h.apiError(w, r)
		return
	}
	var record map[string]any
	if json.Unmarshal(response, &record) != nil {
		h.apiError(w, r)
		return
	}
	expectedVersion, err := strconv.ParseInt(r.PostFormValue(versionKey), 10, 64)
	if err != nil || expectedVersion < 1 {
		http.Error(w, "document changed; reload before transitioning", http.StatusConflict)
		return
	}
	current := PublicationState(displayValue(record[collection.Workflow.StateField]))
	if collection.Drafts {
		draft, err := h.Options.GetDraft(r, collection, id, localeState(r).Current)
		publishedVersion := generatedRecordVersion(record)
		expectedPublished, parseErr := strconv.ParseInt(r.PostFormValue("expected_published_version"), 10, 64)
		if err != nil || parseErr != nil || !draft.Exists || draft.ID != id || draft.Version != expectedVersion || draft.PublishedVersion != expectedPublished || publishedVersion != expectedPublished {
			http.Error(w, "draft or published document changed; reload before transitioning", http.StatusConflict)
			return
		}
		current = draft.State
		if current == target || !collection.Workflow.CanTransition(current, target) || !h.Options.CanTransitionDraft(r, collection, draft, target) {
			http.Error(w, "transition unavailable", http.StatusForbidden)
			return
		}
		if err := h.Options.TransitionDraft(r, collection, id, localeState(r).Current, target, expectedVersion, expectedPublished); err != nil {
			http.Error(w, "transition unavailable", http.StatusConflict)
			return
		}
	} else if generatedRecordVersion(record) != expectedVersion {
		http.Error(w, "document changed; reload before transitioning", http.StatusConflict)
		return
	} else if current == target || !collection.Workflow.CanTransition(current, target) || !h.Options.CanTransition(r, collection, record, target) {
		http.Error(w, "transition unavailable", http.StatusForbidden)
		return
	} else if err := h.Options.Transition(r, collection, record, target, expectedVersion); err != nil {
		http.Error(w, "transition unavailable", http.StatusConflict)
		return
	}
	redirect := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
	if collection.Localization {
		redirect = localePath(redirect, localeState(r).Current)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (h GeneratedAdminHandler) mutate(w http.ResponseWriter, r *http.Request, collection Collection, id string, deleting bool) {
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if h.Options.ValidateCSRF == nil || !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	method, path := http.MethodPost, "/api/v1/"+url.PathEscape(collection.ID)
	var body []byte
	creating := !deleting && strings.HasSuffix(r.URL.Path, "/new")
	if deleting {
		method = http.MethodDelete
		path += "/" + url.PathEscape(id)
	} else {
		data := make(map[string]any, len(collection.Fields)+1)
		if collection.Versioned && !creating {
			versions := r.Form["version"]
			if len(versions) != 1 {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			version, err := strconv.ParseInt(versions[0], 10, 64)
			if err != nil || version < 1 {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			data["version"] = version
		}
		if id == "" {
			id = r.FormValue("id")
		}
		if collection.Type == "global" && id == "" {
			id = "global"
		}
		if id != "" {
			data["id"] = id
		}
		for _, field := range collection.Fields {
			if field.ReadOnly || (collection.Workflow != nil && field.ID == collection.Workflow.StateField) {
				continue
			}
			values, exists := r.Form[field.ID]
			if !exists {
				if field.Kind == FieldBoolean {
					data[field.ID] = false
				} else if field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
					data[field.ID] = []string{}
				}
				continue
			}
			value, err := parseAdminField(field, values)
			if err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			data[field.ID] = value
		}
		if collection.Workflow != nil && creating {
			data[collection.Workflow.StateField] = string(collection.Workflow.States[0])
		} else if collection.Workflow != nil {
			current, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
			var record map[string]any
			if err != nil || status != http.StatusOK || json.Unmarshal(current, &record) != nil {
				h.renderMutationFailure(w, r, collection, id, creating, deleting)
				return
			}
			data[collection.Workflow.StateField] = record[collection.Workflow.StateField]
		}
		if id != "" && !creating {
			method = http.MethodPut
			path += "/" + url.PathEscape(id)
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		body = encoded
	}
	responseBody, status, err := h.callAPI(r, method, path, body)
	if err != nil {
		h.renderMutationFailure(w, r, collection, id, creating, deleting)
		return
	}
	switch status {
	case http.StatusConflict:
		record := map[string]any{}
		if !creating {
			record, _ = h.currentResource(r, collection, id)
		}
		h.renderFormState(w, r, collection, id, record, !creating, submittedGeneratedValues(r, collection), nil, http.StatusConflict, "conflict")
		return
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		if !deleting && status == http.StatusUnprocessableEntity {
			fieldErrors, valid := generatedAdminValidationErrors(collection, responseBody, generatedAdminLocale(r))
			if valid {
				record := map[string]any{}
				if !creating {
					var loadErr error
					record, loadErr = h.currentResource(r, collection, id)
					if loadErr != nil {
						h.apiError(w, r)
						return
					}
				}
				h.renderFormState(w, r, collection, id, record, !creating, submittedGeneratedValues(r, collection), fieldErrors, http.StatusUnprocessableEntity, "")
				return
			}
		}
		http.Error(w, "Bilgiler geçersiz veya eksik. Alanları kontrol edip yeniden deneyin.", http.StatusBadRequest)
		return
	case http.StatusForbidden:
		http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
		return
	case http.StatusNotFound:
		http.NotFound(w, r)
		return
	}
	if status < 200 || status >= 300 {
		h.renderMutationFailure(w, r, collection, id, creating, deleting)
		return
	}
	redirect := "/admin/resources/" + url.PathEscape(collection.ID)
	if collection.Localization {
		redirect = localePath(redirect, localeState(r).Current)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (h GeneratedAdminHandler) renderMutationFailure(w http.ResponseWriter, r *http.Request, collection Collection, id string, creating, deleting bool) {
	if deleting {
		record, err := h.currentResource(r, collection, id)
		if err != nil {
			locale := generatedAdminLocale(r)
			title := i18n.T(locale, "admin.generated.error.delete_title")
			message := i18n.T(locale, "admin.generated.error.delete_operation")
			http.Error(w, title+": "+message, http.StatusBadGateway)
			return
		}
		h.renderFormState(w, r, collection, id, record, true, nil, nil, http.StatusBadGateway, "delete")
		return
	}
	h.renderFormState(w, r, collection, id, map[string]any{}, !creating, submittedGeneratedValues(r, collection), nil, http.StatusBadGateway, "save")
}

func submittedGeneratedValues(r *http.Request, collection Collection) map[string]string {
	values := make(map[string]string, len(collection.Fields)+4)
	for _, field := range collection.Fields {
		if field.ReadOnly || collection.Workflow != nil && field.ID == collection.Workflow.StateField {
			continue
		}
		items, exists := r.Form[field.ID]
		if !exists {
			if field.Kind == FieldBoolean || field.Kind == FieldArray || field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
				values[field.ID] = ""
			}
			continue
		}
		value, err := parseAdminField(field, items)
		if err != nil {
			continue
		}
		switch typed := value.(type) {
		case []string:
			if field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
				values[field.ID] = strings.Join(typed, ",")
			} else {
				encoded, _ := json.Marshal(typed)
				values[field.ID] = string(encoded)
			}
		case bool:
			values[field.ID] = strconv.FormatBool(typed)
		default:
			values[field.ID] = fmt.Sprint(typed)
		}
	}
	for _, key := range []string{"version", "locale_version", "draft_version", "published_version"} {
		if items, ok := r.Form[key]; ok && len(items) == 1 {
			values[key] = items[0]
		}
	}
	return values
}

func generatedAdminValidationErrors(collection Collection, response []byte, locale string) ([]ui.FieldValidationError, bool) {
	var payload struct {
		Code   string            `json:"code"`
		Fields map[string]string `json:"fields"`
	}
	if len(response) == 0 || json.Unmarshal(response, &payload) != nil || payload.Code != "validation_failed" {
		return nil, false
	}
	result := make([]ui.FieldValidationError, 0, len(payload.Fields))
	for _, field := range collection.Fields {
		code, exists := payload.Fields[field.ID]
		if !exists || field.ReadOnly || collection.Workflow != nil && field.ID == collection.Workflow.StateField {
			continue
		}
		target := field.ID
		switch {
		case field.Kind == FieldBlocks || field.Widget == "block-editor":
			target += "-mount"
		case field.Widget == "richtext":
			target += "-editor"
		case field.Kind == FieldMedia || field.Widget == "media":
			target += "-field"
		}
		messageKey := "admin.generated.validation.field.invalid"
		if code == "required" {
			messageKey = "admin.generated.validation.field.required"
		}
		result = append(result, ui.FieldValidationError{FieldID: target, Label: field.Label, Message: i18n.T(locale, messageKey)})
	}
	return result, true
}
