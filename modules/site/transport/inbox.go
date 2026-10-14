package transport

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/internal/xlsx"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) inbox(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 100)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusInternalServerError)
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(items)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		if query != "" && !strings.Contains(strings.ToLower(item.Email+" "+item.Payload+" "+item.ID), query) {
			continue
		}
		rows = append(rows, []string{item.Email, item.Status, item.LeadStage, item.AssignedTo, submissionLocation(item), item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Gelen talepler", "Form gelen kutusu", "site") != nil {
		return
	}
	_ = inboxToolbar(r.PathValue("form"), query).Render(r.Context(), w)
	_ = uicomponents.Table(uicomponents.TableProps{Caption: "Form talepleri", Columns: []string{"E-posta", "Durum", "Lead aşaması", "Atanan kişi", "Konum", "Tarih"}, Rows: rows, EmptyMessage: "Eşleşen talep yok."}).Render(r.Context(), w)
	if h.assistantEnabled() {
		options := make([]designsystem.AIInboxOption, 0, len(items))
		for _, item := range items {
			options = append(options, designsystem.AIInboxOption{ID: item.ID, Label: "Talep · " + item.CreatedAt.UTC().Format("2006-01-02 15:04 UTC")})
		}
		summary := ""
		if result, ok := r.Context().Value(aiInboxSummaryKey{}).(aiInboxSummary); ok && result.Form == r.PathValue("form") {
			summary = result.Text
		}
		action := "/admin/site/inbox/" + url.PathEscape(r.PathValue("form")) + "/ai-summarize"
		_ = designsystem.AIInboxSummary(options, action, security.CSRFTokenFromContext(r.Context()), summary).Render(r.Context(), w)
	}
	_ = ui.AdminEnd(w)
}

type aiInboxSummaryKey struct{}

type aiInboxSummary struct{ Form, Text string }

func (h Handler) summarizeInboxAI(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !h.admin(r) || !h.assistantEnabled() || h.Service == nil || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if r.FormValue("consent") != "yes" || strings.TrimSpace(r.FormValue("submission_id")) == "" {
		http.Error(w, "Form yanıtı seçin ve içerik paylaşımını onaylayın.", http.StatusBadRequest)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 100)
	if err != nil {
		http.Error(w, "Form yanıtı şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	var selected *site.Submission
	for i := range items {
		if items[i].ID == r.FormValue("submission_id") {
			selected = &items[i]
			break
		}
	}
	if selected == nil {
		http.Error(w, "Form yanıtı bulunamadı.", http.StatusNotFound)
		return
	}
	if err := h.Service.Transactions.Within(r.Context(), func(tx database.Executor) error {
		auditID := "ai-submission-" + fmt.Sprintf("%x", sha256.Sum256([]byte(selected.ID+"\x00"+user.ID+"\x00"+time.Now().UTC().Format(time.RFC3339Nano))))[:32]
		return h.AIAudit.Record(r.Context(), tx, auditID, "ai.submission.summary.requested", "submission:"+selected.ID)
	}); err != nil {
		http.Error(w, "Özet hizmeti şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	result, err := h.Assistant.Suggest(r.Context(), ai.Request{Operation: ai.Summary, Locale: "tr", Title: "Form yanıtları", Body: selected.Payload, Consent: true})
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, ai.ErrInvalidRequest) {
			status = http.StatusBadRequest
		}
		http.Error(w, "Özet oluşturulamadı. Form yanıtı değiştirilmedi.", status)
		return
	}
	ctx := context.WithValue(r.Context(), aiInboxSummaryKey{}, aiInboxSummary{Form: r.PathValue("form"), Text: result.Body})
	h.inbox(w, r.WithContext(ctx))
}

func (h Handler) inboxStatus(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil || site.ValidateInboxStatus(r.FormValue("status")) != nil {
		http.Error(w, "invalid inbox status", http.StatusBadRequest)
		return
	}
	if err := h.Service.SetSubmissionStatus(r.Context(), r.PathValue("id"), r.FormValue("status")); err != nil {
		http.Error(w, "inbox update failed", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) leadUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid lead update", http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateLead(r.Context(), r.PathValue("id"), r.FormValue("lead_status"), r.FormValue("assigned_to")); err != nil {
		http.Error(w, "lead update failed", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) leadNoteCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !h.admin(r) || h.Service == nil || user == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 9<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid note", http.StatusBadRequest)
		return
	}
	id, err := randomID("note-")
	if err != nil {
		http.Error(w, "notes unavailable", http.StatusServiceUnavailable)
		return
	}
	note := site.LeadNote{ID: id, SubmissionID: r.PathValue("id"), AuthorID: user.ID, Note: r.FormValue("note"), CreatedAt: time.Now().UTC()}
	if err := h.Service.AddLeadNote(r.Context(), note); err != nil {
		http.Error(w, "note could not be saved", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h Handler) leadNotes(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Service.ListLeadNotes(r.Context(), r.PathValue("id"), 100)
	if err != nil {
		http.Error(w, "notes unavailable", siteStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(items)
}

func (h Handler) exportInboxCSV(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || guards.IsNil(h.Store) || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 1000)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="form-inbox.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"ID", "Form", "Email", "Country", "City", "Local network", "Created at", "Inbox status", "Lead stage", "Assigned to", "Values"})
	for _, item := range items {
		created := item.CreatedAt.UTC().Format(time.RFC3339)
		_ = writer.Write([]string{csvSafe(item.ID), csvSafe(item.FormKey), csvSafe(item.Email), item.LocationCountryCode, csvSafe(item.LocationCity), strconv.FormatBool(item.LocationLocal), created, csvSafe(item.Status), csvSafe(item.LeadStage), csvSafe(item.AssignedTo), csvSafe(item.Payload)})
	}
	writer.Flush()
	if err := writer.Error(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return
	}
}

func (h Handler) exportInboxXLSX(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || guards.IsNil(h.Store) || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 1000)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	rows := [][]string{{"ID", "Form", "Email", "Country", "City", "Local network", "Created at", "Inbox status", "Lead stage", "Assigned to", "Values"}}
	for _, item := range items {
		rows = append(rows, []string{item.ID, item.FormKey, item.Email, item.LocationCountryCode, item.LocationCity, strconv.FormatBool(item.LocationLocal), item.CreatedAt.UTC().Format(time.RFC3339), item.Status, item.LeadStage, item.AssignedTo, item.Payload})
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="form-inbox.xlsx"`)
	if err := xlsx.Write(w, rows); err != nil {
		return
	}
}

func (h Handler) mailSpool(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.MailSpool == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != mailer.SpoolPending && status != mailer.SpoolSending && status != mailer.SpoolDelivered && status != mailer.SpoolFailedPermanent {
		http.Error(w, "invalid mail queue filter", http.StatusBadRequest)
		return
	}
	items, err := h.MailSpool.List(r.Context(), status, 100)
	if err != nil {
		http.Error(w, "mail queue unavailable", http.StatusServiceUnavailable)
		return
	}
	var rows [][]string
	var actions [][]uicomponents.TableAction
	for _, item := range items {
		nextAttempt := "—"
		if !item.NextAttemptAt.IsZero() {
			nextAttempt = item.NextAttemptAt.Local().Format("02.01.2006 15:04")
		}
		rows = append(rows, []string{mailSpoolStatusLabel(item.Status), item.Message.To, item.Message.Subject, strconv.Itoa(item.Attempts), nextAttempt})
		rowActions := []uicomponents.TableAction{}
		if item.Status == mailer.SpoolFailedPermanent {
			rowActions = append(rowActions, uicomponents.TableAction{Label: "Yeniden dene", Action: "/admin/site/mail/" + url.PathEscape(item.ID) + "/retry", CSRF: security.CSRFTokenFromContext(r.Context()), Variant: uicomponents.Quiet})
		}
		actions = append(actions, rowActions)
	}
	filter := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		if err := uicomponents.Select(uicomponents.SelectProps{ID: "mail-status", Name: "status", Label: "Durum", Value: status, Items: []uicomponents.SelectOption{{Value: "", Label: "Tüm durumlar"}, {Value: mailer.SpoolPending, Label: "Bekliyor"}, {Value: mailer.SpoolSending, Label: "Gönderiliyor"}, {Value: mailer.SpoolDelivered, Label: "Teslim edildi"}, {Value: mailer.SpoolFailedPermanent, Label: "Kalıcı hata"}}}).Render(ctx, output); err != nil {
			return err
		}
		return uicomponents.Button(uicomponents.ButtonProps{Label: "Filtrele", Type: "submit", Variant: uicomponents.Primary}).Render(ctx, output)
	})
	content := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		if err := designsystem.Callout(designsystem.CalloutProps{Tone: "info", Title: "Teslimat hakkında", Text: "Bekleyen iletiler otomatik olarak yeniden denenir. Sağlayıcı ayarlarını düzelttikten sonra kalıcı hatalı iletileri yeniden kuyruğa alabilirsiniz."}).Render(ctx, output); err != nil {
			return err
		}
		if err := uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/mail", Method: http.MethodGet, Class: "studio-form", Body: filter}).Render(ctx, output); err != nil {
			return err
		}
		return uicomponents.Card(uicomponents.CardProps{Title: "İletiler", Body: uicomponents.Table(uicomponents.TableProps{Caption: "E-posta teslim kuyruğu", Columns: []string{"Durum", "Alıcı", "Konu", "Deneme", "Sonraki deneme"}, Rows: rows, RowActions: actions, EmptyMessage: "Bu filtrede ileti bulunamadı."})}).Render(ctx, output)
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "E-posta kuyruğu", "Site", "site"); err != nil {
		return
	}
	if err := content.Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

func mailSpoolStatusLabel(status string) string {
	switch status {
	case mailer.SpoolPending:
		return "Bekliyor"
	case mailer.SpoolSending:
		return "Gönderiliyor"
	case mailer.SpoolDelivered:
		return "Teslim edildi"
	case mailer.SpoolFailedPermanent:
		return "Kalıcı hata"
	default:
		return "Bilinmiyor"
	}
}

func (h Handler) mailSpoolRetry(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.MailSpool == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.MailSpool.RetryNow(r.Context(), r.PathValue("id")); err != nil {
		http.Error(w, "message could not be retried", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/mail", http.StatusSeeOther)
}

func csvSafe(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
