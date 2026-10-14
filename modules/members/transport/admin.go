package transport

// Admin member list, detail and export handlers (/admin/members). The views
// are in admin_views.templ.

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/internal/xlsx"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// memberListView is the model of memberListPage.
type memberListView struct {
	Query, Status, CSRF string
	StatusOptions       []memberStatusOption
	ExportLinks         []designsystem.ExportLink
	Rows                []memberListRow
	Total, Page, Pages  int
	PrevURL, NextURL    string
}

type memberStatusOption struct{ Value, Label string }

type memberListRow struct {
	Email, DetailURL   string
	Active             bool
	VerifiedAt         string // empty when the e-mail is not verified
	LastLogin, Created string
	ResendAction       string // empty when no verification resend is offered
}

// memberDetailView is the model of memberDetailPage.
type memberDetailView struct {
	Email, StatusText, VerifyText, LastLogin string
	ToggleActive, Verify                     uicomponents.ActionFormProps
	RevokeAll                                *uicomponents.ActionFormProps
	Sessions                                 []memberSessionRow
	Audit                                    []memberAuditRow
}

type memberSessionRow struct {
	IP, UserAgent, LastSeen, Expires string
	Revoke                           uicomponents.ActionFormProps
}

type memberAuditRow struct{ Action, RequestID string }

var memberStatusOptions = []memberStatusOption{
	{"all", "Tüm Durumlar"},
	{"active", "Sadece Aktifler"},
	{"inactive", "Sadece Pasifler"},
	{"verified", "E-postası Doğrulanmış"},
	{"unverified", "E-postası Doğrulanmamış"},
}

func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	allowedStatus := map[string]bool{"all": true, "active": true, "inactive": true, "verified": true, "unverified": true}
	if !allowedStatus[status] {
		status = "all"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	limit := 20
	offset := (page - 1) * limit

	filter := members.MemberFilter{
		Query:  q,
		Status: status,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.Service.Members.Search(r.Context(), filter)
	if err != nil {
		http.Error(w, "member list unavailable", 503)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Üyeler", "", "members"); err != nil {
		return
	}

	totalPages := (result.TotalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}
	csrf := security.CSRFTokenFromContext(r.Context())
	exportURL := "/admin/members/export.xlsx?q=" + url.QueryEscape(q) + "&status=" + url.QueryEscape(status)
	view := memberListView{
		Query: q, Status: status, CSRF: csrf, StatusOptions: memberStatusOptions,
		ExportLinks: []designsystem.ExportLink{{Label: "CSV dışa aktar", URL: strings.Replace(exportURL, "/export.xlsx", "/export.csv", 1)}, {Label: "Excel dışa aktar", URL: exportURL}},
		Total:       result.TotalCount, Page: page, Pages: totalPages,
	}
	for _, member := range result.Members {
		row := memberListRow{
			Email:     member.Email,
			DetailURL: "/admin/members/" + url.PathEscape(member.ID),
			Active:    member.Active,
			Created:   member.CreatedAt.Format("2006-01-02"),
		}
		if member.EmailVerifiedAt != nil {
			row.VerifiedAt = member.EmailVerifiedAt.Format("2006-01-02 15:04")
		} else if h.VerificationEnabled {
			row.ResendAction = "/admin/members/" + url.PathEscape(member.ID) + "/resend-verification"
		}
		if member.LastLoginAt != nil {
			row.LastLogin = member.LastLoginAt.Format("2006-01-02 15:04")
		}
		view.Rows = append(view.Rows, row)
	}
	if page > 1 {
		view.PrevURL = fmt.Sprintf("/admin/members?q=%s&status=%s&page=%d", url.QueryEscape(q), url.QueryEscape(status), page-1)
	}
	if page < totalPages {
		view.NextURL = fmt.Sprintf("/admin/members?q=%s&status=%s&page=%d", url.QueryEscape(q), url.QueryEscape(status), page+1)
	}
	var body bytes.Buffer
	if err := memberListPage(view).Render(r.Context(), &body); err == nil {
		_, _ = w.Write(body.Bytes())
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) adminExportXLSX(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	rows, err := h.memberExportRows(r)
	if err != nil {
		writeMemberExportError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="members.xlsx"`)
	if err := xlsx.Write(w, rows); err != nil {
		return
	}
}

func (h Handler) adminExportCSV(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	rows, err := h.memberExportRows(r)
	if err != nil {
		writeMemberExportError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="members.csv"`)
	writer := csv.NewWriter(w)
	for index, row := range rows {
		if index > 0 && len(row) > 0 {
			row[0] = csvSafeMember(row[0])
		}
		if err := writer.Write(row); err != nil {
			return
		}
	}
	writer.Flush()
}

func csvSafeMember(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func (h Handler) memberExportRows(r *http.Request) ([][]string, error) {
	const pageSize, maxRows = 500, 20_000
	filter := members.MemberFilter{Query: strings.TrimSpace(r.URL.Query().Get("q")), Status: r.URL.Query().Get("status"), Limit: pageSize}
	first, err := h.Service.Members.Search(r.Context(), filter)
	if err != nil {
		return nil, err
	}
	if first.TotalCount > maxRows {
		return nil, errMemberExportTooLarge
	}
	rows := make([][]string, 1, first.TotalCount+1)
	rows[0] = []string{"Email", "Active", "Email verified at", "Last login at", "Created at"}
	appendMembers := func(values []members.Member) {
		for _, member := range values {
			verified, lastLogin := "", ""
			if member.EmailVerifiedAt != nil {
				verified = member.EmailVerifiedAt.UTC().Format(time.RFC3339)
			}
			if member.LastLoginAt != nil {
				lastLogin = member.LastLoginAt.UTC().Format(time.RFC3339)
			}
			rows = append(rows, []string{member.Email, strconv.FormatBool(member.Active), verified, lastLogin, member.CreatedAt.UTC().Format(time.RFC3339)})
		}
	}
	appendMembers(first.Members)
	for offset := len(first.Members); offset < first.TotalCount; offset += pageSize {
		filter.Offset = offset
		page, err := h.Service.Members.Search(r.Context(), filter)
		if err != nil {
			return nil, err
		}
		appendMembers(page.Members)
		if len(page.Members) == 0 {
			break
		}
	}
	return rows, nil
}

var errMemberExportTooLarge = errors.New("member export exceeds row limit")

func writeMemberExportError(w http.ResponseWriter, err error) {
	if errors.Is(err, errMemberExportTooLarge) {
		http.Error(w, "export is too large; narrow the member filter", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "member export unavailable", http.StatusServiceUnavailable)
}

func (h Handler) adminDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	member, err := h.Service.Members.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, members.ErrMemberNotFound) {
			http.Error(w, "member not found", 404)
			return
		}
		http.Error(w, "member lookup failed", 500)
		return
	}

	var sessions []members.Session
	if h.Service.Sessions != nil {
		sessions, _ = h.Service.Sessions.ListByMemberID(r.Context(), id)
	}
	var auditEvents []members.AuditEvent
	if h.Service.Audit != nil {
		auditEvents, _ = h.Service.Audit.ListByTargetID(r.Context(), id, 20)
	}

	csrf := security.CSRFTokenFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Üye Detayı", "", "members"); err != nil {
		return
	}
	_ = memberDetailPage(memberDetailViewFor(member, sessions, auditEvents, csrf)).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func memberDetailViewFor(member *members.Member, sessions []members.Session, auditEvents []members.AuditEvent, csrf string) memberDetailView {
	base := "/admin/members/" + url.PathEscape(member.ID)
	form := func(action, label string, variant uicomponents.Variant, class string, hidden ...uicomponents.HiddenInputProps) uicomponents.ActionFormProps {
		return uicomponents.ActionFormProps{Action: action, Method: "post", CSRF: csrf, Hidden: hidden, Button: uicomponents.ButtonProps{Type: "submit", Label: label, Variant: variant, Class: class}}
	}
	view := memberDetailView{Email: member.Email, StatusText: "Aktif", VerifyText: "Doğrulanmamış", LastLogin: "Giriş Yok"}
	if member.Active {
		view.ToggleActive = form(base+"/active", "Hesabı Devre Dışı Bırak", uicomponents.Danger, "member-action-button", uicomponents.HiddenInputProps{Name: "active", Value: "false"})
	} else {
		view.StatusText = "Pasif"
		view.ToggleActive = form(base+"/active", "Hesabı Etkinleştir", "success", "member-action-button", uicomponents.HiddenInputProps{Name: "active", Value: "true"})
	}
	if member.EmailVerifiedAt != nil {
		view.VerifyText = member.EmailVerifiedAt.Format("2006-01-02 15:04")
		view.Verify = form(base+"/verify-email", "Doğrulama Durumunu Kaldır", uicomponents.Quiet, "member-action-button", uicomponents.HiddenInputProps{Name: "verified", Value: "false"})
	} else {
		view.Verify = form(base+"/verify-email", "E-postayı Doğrulandı Olarak İşaretle", uicomponents.Primary, "member-action-button", uicomponents.HiddenInputProps{Name: "verified", Value: "true"})
	}
	if member.LastLoginAt != nil {
		view.LastLogin = member.LastLoginAt.Format("2006-01-02 15:04")
	}
	if len(sessions) > 0 {
		revokeAll := form(base+"/sessions/revoke-all", "Tüm Oturumları Kapat", uicomponents.Quiet, "button-danger member-small-button")
		view.RevokeAll = &revokeAll
	}
	for _, session := range sessions {
		view.Sessions = append(view.Sessions, memberSessionRow{
			IP: session.IPAddress, UserAgent: session.UserAgent,
			LastSeen: session.LastSeenAt.Format("2006-01-02 15:04:05"),
			Expires:  session.ExpiresAt.Format("2006-01-02 15:04"),
			Revoke:   form(base+"/sessions/"+url.PathEscape(session.ID)+"/revoke", "Sonlandır", uicomponents.Quiet, "button-danger member-tiny-button"),
		})
	}
	for _, event := range auditEvents {
		view.Audit = append(view.Audit, memberAuditRow{Action: event.Action, RequestID: event.RequestID})
	}
	return view
}
