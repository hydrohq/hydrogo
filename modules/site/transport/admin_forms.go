package transport

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/xlsx"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) requireForms(w http.ResponseWriter, r *http.Request) bool {
	if !h.requireSiteManage(w, r) {
		return false
	}
	if guards.IsNil(h.Service.Forms) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "forms unavailable", nil)
		return false
	}
	return true
}

func (h Handler) adminForms(w http.ResponseWriter, r *http.Request) {
	if !h.requireForms(w, r) {
		return
	}
	items, err := h.Service.ListForms(r.Context())
	if err != nil {
		writeSiteError(w, r, err, "forms unavailable")
		return
	}
	if items == nil {
		items = []forms.Form{}
	}
	api.WriteData(w, http.StatusOK, items, nil)
}

func (h Handler) adminForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireForms(w, r) {
		return
	}
	form, err := h.Service.Forms.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSiteError(w, r, err, "form unavailable")
		return
	}
	api.WriteData(w, http.StatusOK, form, nil)
}

func (h Handler) saveForm(w http.ResponseWriter, r *http.Request, form forms.Form, status int) {
	if err := form.Validate(); err != nil {
		api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), map[string]string{"form": "invalid"})
		return
	}
	if err := h.Service.SaveForm(r.Context(), form); err != nil {
		writeSiteError(w, r, err, "form could not be saved")
		return
	}
	api.WriteData(w, status, form, nil)
}

func (h Handler) adminCreateForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireForms(w, r) {
		return
	}
	var form forms.Form
	if !decode(w, r, &form) {
		return
	}
	if _, err := h.Service.Forms.Get(r.Context(), form.ID); err == nil {
		api.WriteError(w, r, http.StatusConflict, "a form with this id already exists", map[string]string{"id": "taken"})
		return
	}
	h.saveForm(w, r, form, http.StatusCreated)
}

func (h Handler) adminUpdateForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireForms(w, r) {
		return
	}
	id := r.PathValue("id")
	if _, err := h.Service.Forms.Get(r.Context(), id); err != nil {
		writeSiteError(w, r, err, "form unavailable")
		return
	}
	var form forms.Form
	if !decode(w, r, &form) {
		return
	}
	form.ID = id
	h.saveForm(w, r, form, http.StatusOK)
}

func (h Handler) adminDeleteForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireForms(w, r) {
		return
	}
	if err := h.Service.DeleteForm(r.Context(), r.PathValue("id")); err != nil {
		writeSiteError(w, r, err, "form could not be deleted")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type inboxLocation struct {
	CountryCode string `json:"country_code,omitempty"`
	City        string `json:"city,omitempty"`
	Local       bool   `json:"local,omitempty"`
}

type inboxItem struct {
	ID         string            `json:"id"`
	FormKey    string            `json:"form_key"`
	Email      string            `json:"email"`
	Values     map[string]string `json:"values"`
	Status     string            `json:"status"`
	LeadStage  string            `json:"lead_stage"`
	AssignedTo string            `json:"assigned_to"`
	Location   *inboxLocation    `json:"location,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

func inboxItemOf(item site.Submission) inboxItem {
	values := map[string]string{}
	if err := json.Unmarshal([]byte(item.Payload), &values); err != nil {
		values = map[string]string{"_raw": item.Payload}
	}
	view := inboxItem{ID: item.ID, FormKey: item.FormKey, Email: item.Email, Values: values, Status: item.Status, LeadStage: item.LeadStage, AssignedTo: item.AssignedTo, CreatedAt: item.CreatedAt}
	if item.LocationLocal || item.LocationCountryCode != "" || item.LocationCity != "" {
		view.Location = &inboxLocation{CountryCode: item.LocationCountryCode, City: item.LocationCity, Local: item.LocationLocal}
	}
	return view
}

func inboxFilter(w http.ResponseWriter, r *http.Request) (site.SubmissionFilter, bool) {
	query := r.URL.Query()
	filter := site.SubmissionFilter{FormKey: strings.TrimSpace(query.Get("form")), Status: strings.TrimSpace(query.Get("status")), Limit: 50}
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			api.WriteError(w, r, http.StatusBadRequest, "invalid limit", nil)
			return filter, false
		}
		filter.Limit = parsed
	}
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 1_000_000 {
			api.WriteError(w, r, http.StatusBadRequest, "invalid offset", nil)
			return filter, false
		}
		filter.Offset = parsed
	}
	if filter.Status != "" && site.ValidateInboxStatus(filter.Status) != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid status filter", nil)
		return filter, false
	}
	return filter, true
}

func (h Handler) adminInbox(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	filter, ok := inboxFilter(w, r)
	if !ok {
		return
	}
	limit := filter.Limit
	filter.Limit = limit + 1
	items, err := h.Service.ListInbox(r.Context(), filter)
	if err != nil {
		writeSiteError(w, r, err, "inbox unavailable")
		return
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	result := make([]inboxItem, 0, len(items))
	for _, item := range items {
		result = append(result, inboxItemOf(item))
	}
	api.WriteList(w, result, api.PaginationMeta{Limit: limit, Offset: filter.Offset, HasMore: hasMore})
}

func (h Handler) inboxItem(w http.ResponseWriter, r *http.Request) (site.Submission, bool) {
	item, found, err := h.Service.GetSubmission(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSiteError(w, r, err, "inbox unavailable")
		return site.Submission{}, false
	}
	if !found {
		api.WriteError(w, r, http.StatusNotFound, "submission not found", nil)
		return site.Submission{}, false
	}
	return item, true
}

func (h Handler) adminInboxItem(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	item, ok := h.inboxItem(w, r)
	if !ok {
		return
	}
	api.WriteData(w, http.StatusOK, inboxItemOf(item), nil)
}

func (h Handler) adminUpdateInboxItem(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	item, ok := h.inboxItem(w, r)
	if !ok {
		return
	}
	var payload struct {
		Status     *string `json:"status"`
		Stage      *string `json:"stage"`
		AssignedTo *string `json:"assigned_to"`
	}
	if !decode(w, r, &payload) {
		return
	}
	if payload.Status == nil && payload.Stage == nil && payload.AssignedTo == nil {
		api.WriteError(w, r, http.StatusBadRequest, "nothing to update", nil)
		return
	}
	fields := map[string]string{}
	if payload.Status != nil && site.ValidateInboxStatus(*payload.Status) != nil {
		fields["status"] = "invalid"
	}
	stage, assignee := item.LeadStage, item.AssignedTo
	if payload.Stage != nil {
		stage = strings.TrimSpace(*payload.Stage)
	}
	if payload.AssignedTo != nil {
		assignee = strings.TrimSpace(*payload.AssignedTo)
	}
	if (payload.Stage != nil || payload.AssignedTo != nil) && site.ValidateLeadStage(stage) != nil {
		fields["stage"] = "invalid"
	}
	if site.ValidateLeadAssignee(assignee) != nil {
		fields["assigned_to"] = "invalid"
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	if payload.Status != nil {
		if err := h.Service.SetSubmissionStatus(r.Context(), item.ID, *payload.Status); err != nil {
			writeSiteError(w, r, err, "status could not be updated")
			return
		}
	}
	if payload.Stage != nil || payload.AssignedTo != nil {
		if err := h.Service.UpdateLead(r.Context(), item.ID, stage, assignee); err != nil {
			writeSiteError(w, r, err, "lead could not be updated")
			return
		}
	}
	updated, ok := h.inboxItem(w, r)
	if !ok {
		return
	}
	api.WriteData(w, http.StatusOK, inboxItemOf(updated), nil)
}

func (h Handler) adminInboxNotes(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	if _, ok := h.inboxItem(w, r); !ok {
		return
	}
	notes, err := h.Service.ListLeadNotes(r.Context(), r.PathValue("id"), 100)
	if err != nil {
		writeSiteError(w, r, err, "notes unavailable")
		return
	}
	if notes == nil {
		notes = []site.LeadNote{}
	}
	api.WriteData(w, http.StatusOK, notes, nil)
}

func (h Handler) adminAddInboxNote(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	user := auth.UserFromContext(r.Context())
	item, ok := h.inboxItem(w, r)
	if !ok {
		return
	}
	var payload struct {
		Text string `json:"text"`
	}
	if !decode(w, r, &payload) {
		return
	}
	id, err := randomID("note-")
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notes unavailable", nil)
		return
	}
	note := site.LeadNote{ID: id, TenantID: h.tenantID(r), SubmissionID: item.ID, AuthorID: user.ID, Note: strings.TrimSpace(payload.Text), CreatedAt: time.Now().UTC()}
	if err := site.ValidateLeadNote(note); err != nil {
		api.WriteValidation(w, r, map[string]string{"text": "invalid"})
		return
	}
	if err := h.Service.AddLeadNote(r.Context(), note); err != nil {
		writeSiteError(w, r, err, "note could not be saved")
		return
	}
	api.WriteData(w, http.StatusCreated, note, nil)
}

func (h Handler) adminPurgeInbox(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var payload struct {
		Before time.Time `json:"before"`
	}
	if !decode(w, r, &payload) {
		return
	}
	if payload.Before.IsZero() {
		api.WriteValidation(w, r, map[string]string{"before": "required"})
		return
	}
	purged, err := h.Service.PurgeInbox(r.Context(), payload.Before)
	if err != nil {
		writeSiteError(w, r, err, "inbox could not be purged")
		return
	}
	api.WriteData(w, http.StatusOK, map[string]int64{"purged": purged}, nil)
}

var inboxExportHeader = []string{"ID", "Form", "Email", "Country", "City", "Local network", "Created at", "Inbox status", "Lead stage", "Assigned to", "Values"}

func (h Handler) exportRows(w http.ResponseWriter, r *http.Request) ([][]string, bool) {
	filter, ok := inboxFilter(w, r)
	if !ok {
		return nil, false
	}
	filter.Limit, filter.Offset = 1000, 0
	items, err := h.Service.ListInbox(r.Context(), filter)
	if err != nil {
		writeSiteError(w, r, err, "inbox unavailable")
		return nil, false
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.ID, item.FormKey, item.Email, item.LocationCountryCode, item.LocationCity, strconv.FormatBool(item.LocationLocal), item.CreatedAt.UTC().Format(time.RFC3339), item.Status, item.LeadStage, item.AssignedTo, item.Payload})
	}
	return rows, true
}

func (h Handler) adminInboxExportCSV(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	rows, ok := h.exportRows(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="form-inbox.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write(inboxExportHeader)
	for _, row := range rows {
		safe := make([]string, len(row))
		for i, cell := range row {
			safe[i] = csvSafe(cell)
		}
		_ = writer.Write(safe)
	}
	writer.Flush()
	if err := writer.Error(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return
	}
}

func (h Handler) adminInboxExportXLSX(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	rows, ok := h.exportRows(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="form-inbox.xlsx"`)
	_ = xlsx.Write(w, append([][]string{inboxExportHeader}, rows...))
}

func csvSafe(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
