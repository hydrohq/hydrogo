package transport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type publicFormView struct {
	ID               string        `json:"id"`
	Title            string        `json:"title"`
	Fields           []forms.Field `json:"fields"`
	TurnstileEnabled bool          `json:"turnstile_enabled"`
	MultiStep        bool          `json:"multi_step"`
	AfterHoursNotice string        `json:"after_hours_notice,omitempty"`
}

func defaultContactForm() forms.Form {
	return forms.Form{ID: "contact", Title: "İletişim", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "E-posta", Required: true}, {ID: "name", Type: forms.Text, Label: "Adınız"}, {ID: "message", Type: forms.Textarea, Label: "Mesaj", Required: true}}}
}

func (h Handler) loadPublicForm(ctx context.Context, id string) (forms.Form, error) {
	if h.Service != nil && !guards.IsNil(h.Service.Forms) {
		form, err := h.Service.Forms.Get(ctx, id)
		if err == nil {
			return form, nil
		}
		if id != "contact" {
			return forms.Form{}, err
		}
	}
	if _, ok := h.AllowedForms[id]; !ok || id != "contact" {
		return forms.Form{}, site.ErrInvalidSubmission
	}
	return defaultContactForm(), nil
}

// publicForm describes a form so a frontend can render it.
func (h Handler) publicForm(w http.ResponseWriter, r *http.Request) {
	form, err := h.loadPublicForm(r.Context(), strings.TrimSpace(r.PathValue("form")))
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "form not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, publicFormView{ID: form.ID, Title: form.Title, Fields: forms.StableFields(form.Fields), TurnstileEnabled: form.TurnstileEnabled, MultiStep: form.MultiStep, AfterHoursNotice: form.PublicWorkingHoursNotice(time.Now().UTC())}, nil)
}

func (h Handler) recordFormView(w http.ResponseWriter, r *http.Request) {
	if h.FormMetrics == nil || h.VisitorHasher == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	formID := r.PathValue("form")
	if _, err := h.loadPublicForm(r.Context(), formID); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.FormViewLimiter != nil && !h.FormViewLimiter.Allow(tenantID+":"+formID+":"+requestIP(r)) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	now := time.Now().UTC()
	if visitor := h.VisitorHasher.Hash(now.Format("2006-01-02"), tenantID, requestIP(r), r.UserAgent()); visitor != "" {
		_, _ = h.FormMetrics.RecordFormView(r.Context(), tenantID, formID, visitor, now)
	}
	w.WriteHeader(http.StatusNoContent)
}

// submissionInput is what the public submit endpoint reads from either a JSON
// body or a form/multipart body.
type submissionInput struct {
	values         map[string]string
	files          map[string]*multipart.FileHeader
	turnstileToken string
	idempotencyKey string
	honeypot       bool
}

func (h Handler) readSubmission(w http.ResponseWriter, r *http.Request, form forms.Form) (submissionInput, bool) {
	input := submissionInput{values: make(map[string]string, len(form.Fields)), files: map[string]*multipart.FileHeader{}}
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var payload struct {
			Values         map[string]string `json:"values"`
			TurnstileToken string            `json:"turnstile_token"`
			IdempotencyKey string            `json:"idempotency_key"`
			Website        string            `json:"website_url"`
		}
		if !api.DecodeJSON(w, r, &payload, 64<<10) {
			return input, false
		}
		for _, field := range form.Fields {
			input.values[field.ID] = payload.Values[field.ID]
		}
		input.turnstileToken, input.idempotencyKey, input.honeypot = payload.TurnstileToken, strings.TrimSpace(payload.IdempotencyKey), strings.TrimSpace(payload.Website) != ""
		return input, true
	}
	maxBody := int64(64 << 10)
	hasDocuments := false
	for _, field := range form.Fields {
		if field.Type == forms.Document {
			hasDocuments = true
			maxBody = 21 << 20
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	var parseErr error
	if hasDocuments && strings.HasPrefix(contentType, "multipart/form-data") {
		parseErr = r.ParseMultipartForm(1 << 20) // #nosec G120 -- the body is capped by MaxBytesReader above
	} else {
		parseErr = r.ParseForm()
	}
	if parseErr != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid form submission", nil)
		return input, false
	}
	for _, field := range form.Fields {
		input.values[field.ID] = r.PostForm.Get(field.ID)
		if field.Type == forms.Document && r.MultipartForm != nil {
			files := r.MultipartForm.File[field.ID]
			if len(files) > 1 {
				api.WriteError(w, r, http.StatusBadRequest, "invalid form submission", nil)
				return input, false
			}
			if len(files) == 1 && strings.TrimSpace(files[0].Filename) != "" {
				input.files[field.ID] = files[0]
				input.values[field.ID] = "pending-document"
			}
		}
	}
	for _, field := range []string{"website_url", "hp_token", "_gotcha"} {
		if strings.TrimSpace(r.PostForm.Get(field)) != "" {
			input.honeypot = true
		}
	}
	input.turnstileToken = r.PostForm.Get("cf-turnstile-response")
	input.idempotencyKey = strings.TrimSpace(r.PostForm.Get("_submission_token"))
	return input, true
}

func (h Handler) submitForm(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "form inbox unavailable", nil)
		return
	}
	formKey := strings.TrimSpace(r.PathValue("form"))
	form, err := h.loadPublicForm(r.Context(), formKey)
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "form not found", nil)
		return
	}
	if h.SubmissionLimiter != nil && !h.SubmissionLimiter.Allow(formKey+":"+requestIP(r)) {
		security.WriteRateLimitedResponse(w, "too many form submissions", time.Minute)
		return
	}
	input, ok := h.readSubmission(w, r, form)
	if !ok {
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if input.honeypot {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if form.TurnstileEnabled {
		if h.Turnstile == nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "form temporarily unavailable", nil)
			return
		}
		verified, verifyErr := h.Turnstile.Verify(r.Context(), input.turnstileToken, requestIP(r))
		if verifyErr != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "form temporarily unavailable", nil)
			return
		}
		if !verified {
			api.WriteValidation(w, r, map[string]string{"turnstile_token": "invalid"})
			return
		}
	}
	clean, validation, err := form.NormalizeSubmission(input.values)
	if err != nil || len(validation) != 0 {
		if len(validation) == 0 {
			validation = map[string]string{"form": "invalid"}
		}
		api.WriteValidation(w, r, validation)
		return
	}
	if !h.storeDocuments(w, r, form, input) {
		return
	}
	if len(input.files) > 0 {
		clean, validation, err = form.NormalizeSubmission(input.values)
		if err != nil || len(validation) != 0 {
			api.WriteValidation(w, r, validation)
			return
		}
	}
	if source := formSourcePath(r.Referer()); source != "" {
		clean["_source_url"] = source
	}
	email := ""
	for _, field := range form.Fields {
		if field.Type == forms.Email {
			email = clean[field.ID]
			break
		}
	}
	if email == "" {
		api.WriteValidation(w, r, map[string]string{"email": "required"})
		return
	}
	payload, err := json.Marshal(clean)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid form submission", nil)
		return
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "form inbox unavailable", nil)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = input.idempotencyKey
	}
	if site.ValidateIdempotencyKey(idempotencyKey) != nil {
		api.WriteValidation(w, r, map[string]string{"idempotency_key": "invalid"})
		return
	}
	requestBytes, _ := json.Marshal(struct {
		Form   string            `json:"form"`
		Values map[string]string `json:"values"`
	}{Form: formKey, Values: clean})
	requestHash := sha256.Sum256(requestBytes)
	digest := sha256.Sum256([]byte(requestIP(r)))
	submission := site.Submission{ID: "form-" + hex.EncodeToString(idBytes), FormKey: formKey, Email: email, Payload: string(payload), SourceIPHash: hex.EncodeToString(digest[:]), IdempotencyKey: idempotencyKey, RequestHash: hex.EncodeToString(requestHash[:])}
	if h.GeoResolver != nil {
		location := h.GeoResolver.ResolveRequest(r)
		submission.LocationCountryCode, submission.LocationCity, submission.LocationLocal = location.CountryCode, location.City, location.Local
	}
	submission.Notifications = formNotifications(form, clean, submission.ID)
	replayed, err := h.Service.SubmitIdempotent(r.Context(), submission)
	if err != nil {
		writeSiteError(w, r, err, "form submission failed")
		return
	}
	if !replayed && h.Realtime != nil {
		data, _ := json.Marshal(map[string]string{"form_key": formKey})
		_ = h.Realtime.Publish(r.Context(), realtime.Event{ID: submission.ID, Topic: "forms:submission_created", Time: time.Now().UTC(), Data: data})
	}
	api.WriteData(w, http.StatusAccepted, map[string]any{"id": submission.ID, "ticket_code": strings.ToUpper(strings.TrimPrefix(submission.ID, "form-")), "replayed": replayed}, nil)
}

func (h Handler) storeDocuments(w http.ResponseWriter, r *http.Request, form forms.Form, input submissionInput) bool {
	for _, field := range form.Fields {
		header, ok := input.files[field.ID]
		if field.Type != forms.Document || !ok {
			continue
		}
		if h.Media == nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "form temporarily unavailable", nil)
			return false
		}
		file, openErr := header.Open()
		if openErr != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid form submission", nil)
			return false
		}
		asset, uploadErr := h.Media.UploadPrivateDocument(r.Context(), file, header.Filename)
		_ = file.Close()
		if uploadErr != nil {
			status := http.StatusBadRequest
			if errors.Is(uploadErr, media.ErrUploaderRequired) || errors.Is(uploadErr, media.ErrAssetRepositoryRequired) {
				status = http.StatusServiceUnavailable
			}
			api.WriteError(w, r, status, "document upload failed", map[string]string{field.ID: "invalid"})
			return false
		}
		input.values[field.ID] = asset.ID
	}
	return true
}

func formNotifications(form forms.Form, values map[string]string, submissionID string) []site.EmailNotification {
	return formNotificationsAt(form, values, submissionID, time.Now().UTC())
}

func formNotificationsAt(form forms.Form, values map[string]string, submissionID string, now time.Time) []site.EmailNotification {
	if len(form.NotificationRecipients) == 0 && !form.Autoresponder {
		return nil
	}
	ticketCode := strings.ToUpper(strings.TrimPrefix(submissionID, "form-"))
	var summary strings.Builder
	fmt.Fprintf(&summary, "New submission: %s\n\n", form.Title)
	for _, field := range form.Fields {
		fmt.Fprintf(&summary, "%s: %s\n", field.Label, values[field.ID])
	}
	result := make([]site.EmailNotification, 0, len(form.NotificationRecipients)+1)
	for index, recipient := range form.NotificationRecipients {
		result = append(result, site.EmailNotification{EventKey: fmt.Sprintf("%s:notification:%d", submissionID, index), To: recipient, Subject: "New form submission: " + form.Title, Body: summary.String()})
	}
	if form.Autoresponder {
		email := ""
		for _, field := range form.Fields {
			if field.Type == forms.Email {
				email = values[field.ID]
				break
			}
		}
		subject := form.AutoresponderSubject
		if subject == "" {
			subject = "We received your message"
		}
		body, err := form.AutoresponderBodyFor(now)
		if err != nil {
			return result
		}
		if body == "" {
			body = "Thank you for contacting us. We have received your message. Your reference is {{ticket_code}}."
		}
		// Only ticket_code and the submitter's own bounded-choice answers
		// (select/checkbox/cascading_select, whose values the admin defined
		// in Options/CascadeOptions) are available to the template. A free-
		// text field (text/textarea/email/phone/number) is never
		// interpolated: the autoresponder's To address is the submitter's
		// own claimed e-mail, so the site's mail server would otherwise be
		// an open relay for attacker-written content to any inbox.
		variables := autoresponderVariables(form, values, ticketCode)
		subject = expandEmailTemplate(subject, variables)
		body = expandEmailTemplate(body, variables)
		htmlBody, err := mailer.RenderAutoresponderHTML(subject, body)
		if err == nil {
			result = append(result, site.EmailNotification{EventKey: submissionID + ":autoresponder", To: email, Subject: subject, Body: body, HTMLBody: htmlBody})
		}
	}
	return result
}

func expandEmailTemplate(value string, variables map[string]string) string {
	for key, replacement := range variables {
		value = strings.ReplaceAll(value, "{{"+key+"}}", replacement)
	}
	return value
}

// autoresponderVariables is the allowlist of template placeholders the
// autoresponder's subject/body may reference: the server-generated
// ticket_code, and any field whose possible values the admin fixed ahead of
// time (select/checkbox/cascading_select), never free text a submitter
// typed. See formNotificationsAt's own comment for why.
func autoresponderVariables(form forms.Form, values map[string]string, ticketCode string) map[string]string {
	variables := map[string]string{"ticket_code": ticketCode}
	for _, field := range form.Fields {
		switch field.Type {
		case forms.Select, forms.Checkbox, forms.CascadingSelect:
			variables[field.ID] = values[field.ID]
		}
	}
	return variables
}

func formSourcePath(value string) string {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || u.User != nil || u.Path == "" || strings.HasPrefix(u.Path, "//") || strings.ContainsAny(u.Path, "\\\r\n\x00") {
		return ""
	}
	return u.EscapedPath()
}
