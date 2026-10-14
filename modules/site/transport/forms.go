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

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) forms(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Service.ListForms(r.Context())
	if err != nil {
		http.Error(w, "forms unavailable", http.StatusServiceUnavailable)
		return
	}
	var formMetrics []designsystem.FormConversionMetric
	if h.FormMetrics != nil && h.VisitorHasher != nil {
		tenantID := "default"
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantID = scope.TenantID
		}
		to := time.Now().UTC()
		from := to.AddDate(0, 0, -29)
		daily, metricsErr := h.FormMetrics.ListFormDaily(r.Context(), tenantID, "", from.Format("2006-01-02"), to.Format("2006-01-02"))
		if metricsErr == nil {
			formMetrics = make([]designsystem.FormConversionMetric, 0, len(items))
			byForm := make(map[string][]analytics.FormDailyMetric)
			for _, metric := range daily {
				byForm[metric.FormID] = append(byForm[metric.FormID], metric)
			}
			for _, item := range items {
				var row designsystem.FormConversionMetric
				row.FormID, row.Title = item.ID, item.Title
				for _, metric := range byForm[item.ID] {
					row.Views += metric.Views
					row.Submissions += metric.Submissions
					row.Trend = append(row.Trend, designsystem.FormConversionDay{Day: metric.Day, Submissions: metric.Submissions})
				}
				if row.Views > 0 {
					row.ConversionPercent = float64(row.Submissions) * 100 / float64(row.Views)
				}
				formMetrics = append(formMetrics, row)
			}
		}
	}
	selected := forms.Form{ID: "contact", Title: "İletişim", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "E-posta", Required: true}, {ID: "message", Type: forms.Textarea, Label: "Mesaj", Required: true}}}
	for _, item := range items {
		if item.ID == r.URL.Query().Get("id") || (r.URL.Query().Get("id") == "" && selected.ID == "contact") {
			selected = item
			if r.URL.Query().Get("id") != "" {
				break
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Formlar", "Schema yönetimi", "site"); err != nil {
		return
	}
	savedForms := make([]savedFormLink, 0, len(items))
	for _, item := range items {
		savedForms = append(savedForms, savedFormLink{ID: item.ID, Title: item.Title})
	}
	_ = savedFormsList(savedForms, selected.ID).Render(r.Context(), w)
	if len(formMetrics) > 0 {
		_ = designsystem.FormConversionMetrics(formMetrics).Render(r.Context(), w)
	}
	_ = adminstudio.RenderFormBuilderContext(r.Context(), w, adminstudio.FormBuilderData{Form: selected, Action: "/admin/site/forms", CSRFToken: security.CSRFTokenFromContext(r.Context())})
	workingHours := forms.WorkingHours{Timezone: "UTC"}
	if selected.WorkingHours != nil {
		workingHours = *selected.WorkingHours
	}
	_ = ui.WorkingHoursEditor(selected.ID, workingHours.Timezone, workingHours.Notice, security.CSRFTokenFromContext(r.Context()), ui.WorkingHoursDays(workingHours)).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) recordFormView(w http.ResponseWriter, r *http.Request) {
	if h.FormMetrics == nil || h.VisitorHasher == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	formID := r.PathValue("form")
	if _, err := h.publicForm(r.Context(), formID); err != nil {
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
	day := now.Format("2006-01-02")
	visitor := h.VisitorHasher.Hash(day, tenantID, requestIP(r), r.UserAgent())
	if visitor != "" {
		_, _ = h.FormMetrics.RecordFormView(r.Context(), tenantID, formID, visitor, now)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) saveFormWorkingHours(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || h.Service.Forms == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid working-hours settings", http.StatusBadRequest)
		return
	}
	form, err := h.Service.Forms.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "form unavailable", http.StatusNotFound)
		return
	}
	workingHours := forms.WorkingHours{Timezone: strings.TrimSpace(r.FormValue("timezone")), Notice: strings.TrimSpace(r.FormValue("notice"))}
	for day := range workingHours.Days {
		workingHours.Days[day] = forms.DailyHours{Open: r.FormValue(fmt.Sprintf("open_%d", day)), Close: r.FormValue(fmt.Sprintf("close_%d", day))}
	}
	form.WorkingHours = &workingHours
	if err := h.Service.SaveForm(r.Context(), form); err != nil {
		http.Error(w, "working-hours settings are invalid", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/forms?id="+url.QueryEscape(form.ID), http.StatusSeeOther)
}

func (h Handler) saveForm(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	var form forms.Form
	if err := json.Unmarshal([]byte(r.FormValue("schema")), &form); err != nil {
		http.Error(w, "invalid form schema", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveForm(r.Context(), form); err != nil {
		http.Error(w, "invalid form schema", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/forms?id="+url.QueryEscape(form.ID), http.StatusSeeOther)
}

func (h Handler) deleteForm(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.Service.DeleteForm(r.Context(), r.PathValue("id")); err != nil {
		http.Error(w, "form deletion failed", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/forms", http.StatusSeeOther)
}

func (h Handler) submitForm(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "form inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	formKey := strings.TrimSpace(r.PathValue("form"))
	if _, ok := h.AllowedForms[formKey]; !ok {
		http.Error(w, "form not found", http.StatusNotFound)
		return
	}
	if h.SubmissionLimiter != nil && !h.SubmissionLimiter.Allow(formKey+":"+requestIP(r)) {
		security.WriteRateLimitedResponse(w, "too many form submissions", time.Minute)
		return
	}
	form, err := h.publicForm(r.Context(), formKey)
	if err != nil {
		http.Error(w, "form not found", http.StatusNotFound)
		return
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
	if hasDocuments {
		parseErr = r.ParseMultipartForm(1 << 20)
	} else {
		parseErr = r.ParseForm()
	}
	if parseErr != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if honeypotFilled(r) {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if form.TurnstileEnabled {
		if h.Turnstile == nil {
			http.Error(w, "form temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		verified, verifyErr := h.Turnstile.Verify(r.Context(), r.PostForm.Get("cf-turnstile-response"), requestIP(r))
		if verifyErr != nil {
			http.Error(w, "form temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if !verified {
			http.Error(w, "please check your form and try again", http.StatusBadRequest)
			return
		}
	}
	values := make(map[string]string, len(form.Fields))
	for _, field := range form.Fields {
		values[field.ID] = r.PostForm.Get(field.ID)
		if field.Type == forms.Document {
			var files []*multipart.FileHeader
			if r.MultipartForm != nil {
				files = r.MultipartForm.File[field.ID]
			}
			if len(files) > 1 {
				http.Error(w, "invalid form submission", http.StatusBadRequest)
				return
			}
			if len(files) == 1 && strings.TrimSpace(files[0].Filename) != "" {
				values[field.ID] = "pending-document"
			}
		}
	}
	clean, validation, err := form.NormalizeSubmission(values)
	if err != nil || len(validation) != 0 {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	for _, field := range form.Fields {
		if field.Type != forms.Document || values[field.ID] == "" {
			continue
		}
		if h.Media == nil {
			http.Error(w, "form temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.MultipartForm == nil || len(r.MultipartForm.File[field.ID]) != 1 {
			continue
		}
		header := r.MultipartForm.File[field.ID][0]
		file, openErr := header.Open()
		if openErr != nil {
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
		asset, uploadErr := h.Media.UploadPrivateDocument(r.Context(), file, header.Filename)
		_ = file.Close()
		if uploadErr != nil {
			status := http.StatusBadRequest
			if errors.Is(uploadErr, media.ErrUploaderRequired) || errors.Is(uploadErr, media.ErrAssetRepositoryRequired) {
				status = http.StatusServiceUnavailable
			}
			http.Error(w, "document upload failed", status)
			return
		}
		values[field.ID] = asset.ID
	}
	if len(form.Fields) > 0 {
		clean, validation, err = form.NormalizeSubmission(values)
		if err != nil || len(validation) != 0 {
			http.Error(w, "invalid form submission", http.StatusBadRequest)
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
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	payload, err := json.Marshal(clean)
	if err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	digest := sha256.Sum256([]byte(requestIP(r)))
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		http.Error(w, "form inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(r.PostForm.Get("_submission_token"))
	}
	if site.ValidateIdempotencyKey(idempotencyKey) != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	requestBytes, _ := json.Marshal(struct {
		Form   string            `json:"form"`
		Values map[string]string `json:"values"`
	}{Form: formKey, Values: clean})
	requestHash := sha256.Sum256(requestBytes)
	submission := site.Submission{ID: "form-" + hex.EncodeToString(idBytes), FormKey: formKey, Email: email, Payload: string(payload), SourceIPHash: hex.EncodeToString(digest[:]), IdempotencyKey: idempotencyKey, RequestHash: hex.EncodeToString(requestHash[:])}
	if h.GeoResolver != nil {
		location := h.GeoResolver.ResolveRequest(r)
		submission.LocationCountryCode, submission.LocationCity, submission.LocationLocal = location.CountryCode, location.City, location.Local
	}
	submission.Notifications = formNotifications(form, clean, submission.ID)
	replayed, err := h.Service.SubmitIdempotent(r.Context(), submission)
	if err != nil {
		http.Error(w, "form submission failed", siteStatus(err))
		return
	}
	if !replayed && h.Realtime != nil {
		data, _ := json.Marshal(map[string]string{"form_key": formKey})
		_ = h.Realtime.Publish(r.Context(), realtime.Event{ID: submission.ID, Topic: "forms:submission_created", Time: time.Now().UTC(), Data: data})
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h Handler) publicForm(ctx context.Context, id string) (forms.Form, error) {
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
	return forms.Form{ID: "contact", Title: "İletişim", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "E-posta", Required: true}, {ID: "name", Type: forms.Text, Label: "Adınız"}, {ID: "message", Type: forms.Textarea, Label: "Mesaj", Required: true}}}, nil
}

func honeypotFilled(r *http.Request) bool {
	for _, field := range []string{"website_url", "hp_token", "_gotcha"} {
		if strings.TrimSpace(r.PostForm.Get(field)) != "" {
			return true
		}
	}
	return false
}

func formNotifications(form forms.Form, values map[string]string, submissionID string) []site.EmailNotification {
	return formNotificationsAt(form, values, submissionID, time.Now().UTC())
}

func formNotificationsAt(form forms.Form, values map[string]string, submissionID string, now time.Time) []site.EmailNotification {
	if len(form.NotificationRecipients) == 0 && !form.Autoresponder {
		return nil
	}
	variables := make(map[string]string, len(values)+1)
	for key, value := range values {
		variables[key] = value
	}
	variables["ticket_code"] = strings.ToUpper(strings.TrimPrefix(submissionID, "form-"))
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

func requestIP(r *http.Request) string { return requestmeta.ClientIP(r) }

func formSourcePath(value string) string {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || u.User != nil || u.Path == "" || strings.HasPrefix(u.Path, "//") || strings.ContainsAny(u.Path, "\\\r\n\x00") {
		return ""
	}
	return u.EscapedPath()
}
