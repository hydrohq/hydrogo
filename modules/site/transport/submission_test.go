package transport

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	formstore "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/geolocation"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

type verifierStub struct {
	valid bool
	seen  string
}

type aiSummaryProvider struct{ request ai.Request }

func (p *aiSummaryProvider) Suggest(_ context.Context, request ai.Request) (ai.Suggestion, error) {
	p.request = request
	return ai.Suggestion{Body: "Summary without contact details"}, nil
}

type aiAuditRecorder struct{ action, target string }

func (a *aiAuditRecorder) Record(_ context.Context, _ database.Executor, _, action, target string) error {
	a.action, a.target = action, target
	return nil
}

func TestInboxAISummaryRequiresConsentAndDoesNotPersistContent(t *testing.T) {
	handler, _ := submissionHandler(t, forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{
		{ID: "email", Type: forms.Email, Label: "Email", Required: true},
		{ID: "message", Type: forms.Textarea, Label: "Message", Required: true},
	}})
	values := url.Values{"email": {"visitor@example.test"}, "message": {"Need help with my order"}, "_submission_token": {"ai-summary-submission-token-123"}}
	if response := submitRequest(handler, values); response.Code != http.StatusAccepted {
		t.Fatalf("submission status=%d body=%s", response.Code, response.Body.String())
	}
	items, err := handler.Store.ListSubmissions(context.Background(), handler.DB, "contact", 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("submissions=%+v err=%v", items, err)
	}
	provider, auditLog := &aiSummaryProvider{}, &aiAuditRecorder{}
	handler.Assistant = ai.Assistant{Enabled: true, Provider: provider}
	handler.AIAudit = auditLog
	post := func(consent string) *httptest.ResponseRecorder {
		form := url.Values{"submission_id": {items[0].ID}, "consent": {consent}, "_csrf": {"csrf-inbox-ai"}}
		request := httptest.NewRequest(http.MethodPost, "/admin/site/inbox/contact/ai-summarize", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf-inbox-ai"})
		request.SetPathValue("form", "contact")
		request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true}))
		response := httptest.NewRecorder()
		security.CSRFMiddleware(false)(http.HandlerFunc(handler.summarizeInboxAI)).ServeHTTP(response, request)
		return response
	}
	denied := post("")
	if denied.Code != http.StatusBadRequest || provider.request.Body != "" || auditLog.action != "" {
		t.Fatalf("missing consent status=%d request=%+v audit=%+v", denied.Code, provider.request, auditLog)
	}
	accepted := post("yes")
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), "Summary without contact details") {
		t.Fatalf("summary status=%d body=%q", accepted.Code, accepted.Body.String())
	}
	if provider.request.Operation != ai.Summary || !strings.Contains(provider.request.Body, "Need help with my order") || strings.Contains(provider.request.Body, "visitor@example.test") {
		t.Fatalf("provider received unsafe summary request: %+v", provider.request)
	}
	if auditLog.action != "ai.submission.summary.requested" || auditLog.target != "submission:"+items[0].ID {
		t.Fatalf("submission audit=%+v", auditLog)
	}
	if strings.Contains(accepted.Body.String(), "Need help with my order") {
		t.Fatal("raw form content was returned in the summary page")
	}
}

var _ audit.Writer = (*aiAuditRecorder)(nil)

type realtimeRecorder struct{ events []realtime.Event }

type geoResolverStub struct{ location geolocation.Location }

func (s geoResolverStub) ResolveRequest(*http.Request) geolocation.Location { return s.location }

func (r *realtimeRecorder) Publish(_ context.Context, event realtime.Event) error {
	r.events = append(r.events, event)
	return nil
}

func (v *verifierStub) Verify(_ context.Context, token, _ string) (bool, error) {
	v.seen = token
	return v.valid, nil
}

func submissionHandler(t *testing.T, form forms.Form) (Handler, database.Executor) {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "submissions.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	transactor := database.NewTenantTransactor(db, database.SQLite, false)
	registry := formstore.Store{Executor: db, Dialect: database.SQLite}
	if err := registry.Upsert(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	store := sitestore.Store{Dialect: database.SQLite}
	service := site.Service{Transactions: transactor, Store: store, Forms: registry}
	return Handler{DB: db, Store: store, Service: &service, FormsEnabled: true, AllowedForms: map[string]struct{}{"contact": {}}}, db
}

func submitRequest(handler Handler, form url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/forms/contact/submit", strings.NewReader(form.Encode()))
	request.SetPathValue("form", "contact")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.submitForm(response, request)
	return response
}

func TestDynamicPublicFormNormalizesAndDeduplicatesSubmission(t *testing.T) {
	handler, db := submissionHandler(t, forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{
		{ID: "email", Type: forms.Email, Label: "Email", Required: true},
		{ID: "message", Type: forms.Textarea, Label: "Message", Required: true},
	}})
	var events realtimeRecorder
	handler.Realtime = &events
	values := url.Values{"email": {" visitor@example.test "}, "message": {"Hello"}, "_submission_token": {"0123456789abcdef0123456789abcdef"}}
	if response := submitRequest(handler, values); response.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d body=%s", response.Code, response.Body.String())
	}
	if response := submitRequest(handler, values); response.Code != http.StatusAccepted {
		t.Fatalf("replay status=%d body=%s", response.Code, response.Body.String())
	}
	if len(events.events) != 1 || events.events[0].Topic != "forms:submission_created" || strings.Contains(string(events.events[0].Data), "visitor@example.test") {
		t.Fatalf("submission realtime events=%+v", events.events)
	}
	store := sitestore.Store{Dialect: database.SQLite}
	items, err := store.ListSubmissions(context.Background(), db, "contact", 10)
	if err != nil || len(items) != 1 || items[0].Email != "visitor@example.test" || !strings.Contains(items[0].Payload, `"message":"Hello"`) {
		t.Fatalf("submissions=%#v err=%v", items, err)
	}
	values.Set("message", "Changed")
	if response := submitRequest(handler, values); response.Code != http.StatusConflict {
		t.Fatalf("conflicting replay status=%d", response.Code)
	}
}

func TestFormSubmissionPersistsResolvedLocation(t *testing.T) {
	handler, db := submissionHandler(t, forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}}})
	handler.GeoResolver = geoResolverStub{location: geolocation.Location{CountryCode: "TR", City: "Istanbul"}}
	values := url.Values{"email": {"visitor@example.test"}, "_submission_token": {"geo-location-token-123456"}}
	if response := submitRequest(handler, values); response.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d body=%s", response.Code, response.Body.String())
	}
	items, err := (sitestore.Store{Dialect: database.SQLite}).ListSubmissions(context.Background(), db, "contact", 10)
	if err != nil || len(items) != 1 || items[0].LocationCountryCode != "TR" || items[0].LocationCity != "Istanbul" {
		t.Fatalf("location metadata=%#v err=%v", items, err)
	}
}

func TestJobFormStoresOnlyValidatedPrivateDocuments(t *testing.T) {
	handler, db := submissionHandler(t, forms.Form{ID: "contact", Title: "Job application", Fields: []forms.Field{
		{ID: "email", Type: forms.Email, Label: "Email", Required: true},
		{ID: "cv", Type: forms.Document, Label: "CV", Required: true},
	}})
	uploader, err := media.NewUploader(filepath.Join(t.TempDir(), "documents"), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler.Media = &media.Service{Transactions: handler.Service.Transactions, Uploader: uploader, Assets: mediastore.Store{Dialect: database.SQLite}}
	post := func(filename, content string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("email", "candidate@example.test")
		_ = writer.WriteField("_submission_token", "0123456789abcdef0123456789abcdef")
		file, err := writer.CreateFormFile("cv", filename)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(content))
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/forms/contact/submit", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.SetPathValue("form", "contact")
		res := httptest.NewRecorder()
		handler.submitForm(res, req)
		return res
	}
	if got := post("candidate.pdf", "%PDF-1.7\nresume"); got.Code != http.StatusAccepted {
		t.Fatalf("valid document submit = %d %s", got.Code, got.Body.String())
	}
	assets, err := (mediastore.Store{Dialect: database.SQLite}).List(context.Background(), db, 10)
	if err != nil || len(assets) != 1 || assets[0].Visibility != media.VisibilityPrivate || assets[0].MimeType != "application/pdf" {
		t.Fatalf("private assets=%#v err=%v", assets, err)
	}
	items, err := (sitestore.Store{Dialect: database.SQLite}).ListSubmissions(context.Background(), db, "contact", 10)
	if err != nil || len(items) != 1 || !strings.Contains(items[0].Payload, assets[0].ID) {
		t.Fatalf("submission payload=%#v err=%v", items, err)
	}
	if got := post("malware.exe", "%PDF-1.7"); got.Code != http.StatusBadRequest {
		t.Fatalf("executable upload status=%d body=%s", got.Code, got.Body.String())
	}
	if got := post("fake.pdf", "not a pdf"); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid PDF upload status=%d", got.Code)
	}
}

func TestHoneypotIsAcknowledgedWithoutPersistence(t *testing.T) {
	handler, db := submissionHandler(t, forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}}})
	values := url.Values{"email": {"visitor@example.test"}, "website_url": {"https://bot.test"}, "_submission_token": {"0123456789abcdef"}}
	if response := submitRequest(handler, values); response.Code != http.StatusAccepted {
		t.Fatalf("honeypot status=%d", response.Code)
	}
	items, err := (sitestore.Store{Dialect: database.SQLite}).ListSubmissions(context.Background(), db, "contact", 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("honeypot persisted submission %#v/%v", items, err)
	}
}

func TestTurnstileIsRequiredOnlyWhenConfiguredOnForm(t *testing.T) {
	form := forms.Form{ID: "contact", Title: "Contact", TurnstileEnabled: true, Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}}}
	handler, _ := submissionHandler(t, form)
	values := url.Values{"email": {"visitor@example.test"}, "_submission_token": {"0123456789abcdef"}, "cf-turnstile-response": {"proof"}}
	if response := submitRequest(handler, values); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing verifier status=%d", response.Code)
	}
	verifier := &verifierStub{}
	handler.Turnstile = verifier
	if response := submitRequest(handler, values); response.Code != http.StatusBadRequest || verifier.seen != "proof" {
		t.Fatalf("failed challenge response=%d token=%q", response.Code, verifier.seen)
	}
	verifier.valid = true
	if response := submitRequest(handler, values); response.Code != http.StatusAccepted {
		t.Fatalf("verified submit status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAfterHoursNoticeFlowsIntoRealAutoresponderNotification(t *testing.T) {
	var days [7]forms.DailyHours
	for day := time.Monday; day <= time.Friday; day++ {
		days[day] = forms.DailyHours{Open: "09:00", Close: "17:00"}
	}
	form := forms.Form{ID: "contact", Title: "Contact", Autoresponder: true, AutoresponderSubject: "Thanks {{name}}", AutoresponderBody: "We received your request.", WorkingHours: &forms.WorkingHours{Timezone: "UTC", Days: days, Notice: "We reopen {weekday} at {time}."}, Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}, {ID: "name", Type: forms.Text, Label: "Name"}}}
	notifications := formNotificationsAt(form, map[string]string{"email": "visitor@example.test", "name": "Sam"}, "form-ABC", time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC))
	if len(notifications) != 1 || notifications[0].To != "visitor@example.test" || notifications[0].Subject != "Thanks Sam" || notifications[0].Body != "We received your request.\n\nWe reopen Tuesday at 09:00." || notifications[0].HTMLBody == "" {
		t.Fatalf("notifications=%+v", notifications)
	}
}
