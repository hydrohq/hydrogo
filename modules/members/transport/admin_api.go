package transport

import (
	"encoding/csv"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/xlsx"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// AdminBasePath is the JSON admin surface for member management.
const AdminBasePath = "/api/v1/admin/members"

const (
	adminMemberPageSize   = 20
	adminMemberMaxLimit   = 100
	adminMemberMaxOffset  = 200_000
	adminMemberJSONBytes  = 8 << 10
	memberExportPageSize  = 500
	memberExportMaxRows   = 20_000
	memberAuditEventLimit = 20
)

var memberStatusFilters = map[string]bool{"all": true, "active": true, "inactive": true, "verified": true, "unverified": true}

func (h Handler) adminRoutes() []router.Route {
	return []router.Route{
		{Pattern: "GET " + AdminBasePath, Access: router.Protected, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "GET " + AdminBasePath + "/stats", Access: router.Protected, Handler: http.HandlerFunc(h.adminStats)},
		{Pattern: "GET " + AdminBasePath + "/export", Access: router.Protected, Handler: http.HandlerFunc(h.adminExport)},
		{Pattern: "GET " + AdminBasePath + "/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminDetail)},
		{Pattern: "PATCH " + AdminBasePath + "/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.adminUpdate)},
		{Pattern: "POST " + AdminBasePath + "/{id}/verify-email", Access: router.Protected, Handler: http.HandlerFunc(h.adminVerifyEmail)},
		{Pattern: "POST " + AdminBasePath + "/{id}/resend-verification", Access: router.Protected, Handler: http.HandlerFunc(h.adminResendVerification)},
		{Pattern: "POST " + AdminBasePath + "/{id}/sessions/{session_id}/revoke", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeSession)},
		{Pattern: "POST " + AdminBasePath + "/{id}/sessions/revoke-all", Access: router.Protected, Handler: http.HandlerFunc(h.adminRevokeAllSessions)},
	}
}

func (h Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if auth.Can(auth.UserFromContext(r.Context()), auth.PermissionMembersManage) {
		return true
	}
	api.WriteError(w, r, http.StatusForbidden, "member management permission required", nil)
	return false
}

func memberFilterFromQuery(values url.Values) (members.MemberFilter, map[string]string) {
	fields := map[string]string{}
	filter := members.MemberFilter{Query: strings.TrimSpace(values.Get("search")), Status: strings.TrimSpace(values.Get("status")), Limit: adminMemberPageSize}
	if len(filter.Query) > 200 {
		fields["search"] = "invalid"
	}
	if filter.Status == "" {
		filter.Status = "all"
	}
	if !memberStatusFilters[filter.Status] {
		fields["status"] = "invalid"
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > adminMemberMaxLimit {
			fields["limit"] = "invalid"
		} else {
			filter.Limit = limit
		}
	}
	if raw := values.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > adminMemberMaxOffset {
			fields["offset"] = "invalid"
		} else {
			filter.Offset = offset
		}
	}
	return filter, fields
}

func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	filter, fields := memberFilterFromQuery(r.URL.Query())
	if len(fields) > 0 {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", fields)
		return
	}
	result, err := h.Service.Members.Search(r.Context(), filter)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "member list unavailable", nil)
		return
	}
	items := result.Members
	if items == nil {
		items = []members.Member{}
	}
	api.WriteList(w, items, api.PaginationMeta{Limit: filter.Limit, Offset: filter.Offset, Total: result.TotalCount, HasMore: filter.Offset+len(items) < result.TotalCount})
}

func (h Handler) adminStats(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	stats, err := h.Service.Stats(r.Context())
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "member stats unavailable", nil)
		return
	}
	api.WriteData(w, http.StatusOK, stats, nil)
}

type memberDetail struct {
	Member   *members.Member      `json:"member"`
	Sessions []members.Session    `json:"sessions"`
	Audit    []members.AuditEvent `json:"audit"`
}

func (h Handler) loadMember(w http.ResponseWriter, r *http.Request) (*members.Member, bool) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" || len(id) > 128 {
		api.WriteError(w, r, http.StatusNotFound, "member not found", nil)
		return nil, false
	}
	member, err := h.Service.Members.GetByID(r.Context(), id)
	if err != nil || member == nil {
		if err == nil || errors.Is(err, members.ErrMemberNotFound) {
			api.WriteError(w, r, http.StatusNotFound, "member not found", nil)
		} else {
			api.WriteError(w, r, http.StatusServiceUnavailable, "member unavailable", nil)
		}
		return nil, false
	}
	return member, true
}

func (h Handler) adminDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	member, ok := h.loadMember(w, r)
	if !ok {
		return
	}
	detail := memberDetail{Member: member, Sessions: []members.Session{}, Audit: []members.AuditEvent{}}
	if h.Service.Sessions != nil {
		if sessions, err := h.Service.Sessions.ListByMemberID(r.Context(), member.ID); err == nil && sessions != nil {
			detail.Sessions = sessions
		}
	}
	if h.Service.Audit != nil {
		if events, err := h.Service.Audit.ListByTargetID(r.Context(), member.ID, memberAuditEventLimit); err == nil && events != nil {
			detail.Audit = events
		}
	}
	api.WriteData(w, http.StatusOK, detail, nil)
}

type memberUpdateRequest struct {
	Active        *bool `json:"active"`
	EmailVerified *bool `json:"email_verified"`
}

// adminUpdate toggles activation and the verified flag. Deactivating a member
// also ends every browser and mobile session so the lockout is immediate.
func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	member, ok := h.loadMember(w, r)
	if !ok {
		return
	}
	var input memberUpdateRequest
	if !api.DecodeJSON(w, r, &input, adminMemberJSONBytes) {
		return
	}
	if input.Active == nil && input.EmailVerified == nil {
		api.WriteValidation(w, r, map[string]string{"active": "required"})
		return
	}
	if input.Active != nil {
		if err := h.Service.Members.SetActive(r.Context(), member.ID, *input.Active); err != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "member update failed", nil)
			return
		}
		if !*input.Active {
			if err := h.Service.RevokeAllSessions(r.Context(), member.ID); err != nil {
				api.WriteError(w, r, http.StatusServiceUnavailable, "member session update failed", nil)
				return
			}
			if h.Mobile != nil {
				if err := h.Mobile.RevokeAll(r.Context(), member.ID); err != nil {
					api.WriteError(w, r, http.StatusServiceUnavailable, "member session update failed", nil)
					return
				}
			}
		}
	}
	if input.EmailVerified != nil {
		if err := h.Service.Members.SetEmailVerified(r.Context(), member.ID, *input.EmailVerified); err != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "member verification update failed", nil)
			return
		}
	}
	updated, err := h.Service.Members.GetByID(r.Context(), member.ID)
	if err != nil || updated == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "member unavailable", nil)
		return
	}
	api.WriteData(w, http.StatusOK, updated, nil)
}

func (h Handler) adminVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	member, ok := h.loadMember(w, r)
	if !ok {
		return
	}
	if err := h.Service.Members.SetEmailVerified(r.Context(), member.ID, true); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "member verification update failed", nil)
		return
	}
	updated, err := h.Service.Members.GetByID(r.Context(), member.ID)
	if err != nil || updated == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "member unavailable", nil)
		return
	}
	api.WriteData(w, http.StatusOK, updated, nil)
}

func (h Handler) adminResendVerification(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	admin := auth.UserFromContext(r.Context())
	if !h.VerificationEnabled || h.VerificationNotifier == nil || strings.TrimSpace(h.PublicOrigin) == "" || h.DB == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "verification email is unavailable", nil)
		return
	}
	member, ok := h.loadMember(w, r)
	if !ok {
		return
	}
	if h.VerificationLimiter != nil && !security.AllowForRequest(h.VerificationLimiter, r.Context(), "member-verification:"+admin.ID+":"+member.ID) {
		security.WriteRateLimitedResponse(w, "verification email temporarily unavailable", time.Hour)
		return
	}
	if member.EmailVerifiedAt != nil {
		api.WriteError(w, r, http.StatusConflict, "member email is already verified", nil)
		return
	}
	var token string
	err := h.VerificationTx.Within(r.Context(), func(tx database.Executor) error {
		var issueErr error
		token, issueErr = h.Verification.Issue(r.Context(), tx, member, 24*time.Hour)
		return issueErr
	})
	if err != nil || token == "" {
		api.WriteError(w, r, http.StatusServiceUnavailable, "verification email could not be sent", nil)
		return
	}
	link := strings.TrimRight(h.PublicOrigin, "/") + "/member/email/verify?token=" + url.QueryEscape(token)
	if err := h.VerificationNotifier.SendEmailVerification(r.Context(), member.Email, link); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "verification email could not be sent", nil)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "member.email_verification.resent", member.ID)
	}
	api.WriteData(w, http.StatusAccepted, map[string]any{"sent": true, "email": member.Email}, nil)
}

func (h Handler) adminRevokeSession(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	memberID, sessionID := r.PathValue("id"), r.PathValue("session_id")
	if err := h.Service.RevokeSessionByID(r.Context(), memberID, sessionID); err != nil {
		if errors.Is(err, members.ErrMemberNotFound) {
			api.WriteError(w, r, http.StatusNotFound, "session not found", nil)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "session revoke failed", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) adminRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	if err := h.Service.RevokeAllSessions(r.Context(), r.PathValue("id")); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "session revoke failed", nil)
		return
	}
	if h.Mobile != nil {
		if err := h.Mobile.RevokeAll(r.Context(), r.PathValue("id")); err != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "session revoke failed", nil)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

var errMemberExportTooLarge = errors.New("member export exceeds row limit")

// adminExport streams the filtered members as CSV or XLSX, bounded so a
// runaway export cannot hold a connection and the database open for minutes.
func (h Handler) adminExport(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "xlsx" {
		api.WriteError(w, r, http.StatusBadRequest, "invalid export format", map[string]string{"format": "invalid"})
		return
	}
	filter, fields := memberFilterFromQuery(r.URL.Query())
	if len(fields) > 0 {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", fields)
		return
	}
	rows, err := h.memberExportRows(r, filter)
	if err != nil {
		if errors.Is(err, errMemberExportTooLarge) {
			api.WriteError(w, r, http.StatusRequestEntityTooLarge, "export is too large; narrow the member filter", nil)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "member export unavailable", nil)
		return
	}
	if format == "xlsx" {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="members.xlsx"`)
		_ = xlsx.Write(w, rows)
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

func (h Handler) memberExportRows(r *http.Request, filter members.MemberFilter) ([][]string, error) {
	filter.Limit, filter.Offset = memberExportPageSize, 0
	first, err := h.Service.Members.Search(r.Context(), filter)
	if err != nil {
		return nil, err
	}
	if first.TotalCount > memberExportMaxRows {
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
	for offset := len(first.Members); offset < first.TotalCount; offset += memberExportPageSize {
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

// csvSafeMember neutralises spreadsheet formula injection in exported cells.
func csvSafeMember(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
