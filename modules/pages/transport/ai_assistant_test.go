package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type aiPageRepository struct{ page pages.Page }

func (r aiPageRepository) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return r.page, nil
}
func (aiPageRepository) GetPublishedByRoute(context.Context, database.Executor, string, string) (pages.Page, error) {
	return pages.Page{}, errors.New("unused")
}
func (aiPageRepository) ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return nil, errors.New("unused")
}
func (aiPageRepository) CreateDraftContent(context.Context, database.Executor, pages.Page) error {
	return errors.New("unused")
}
func (aiPageRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return errors.New("unused")
}
func (aiPageRepository) Publish(context.Context, database.Executor, string, int) error {
	return errors.New("unused")
}

type aiTransactions struct{}

func (aiTransactions) Within(ctx context.Context, fn func(database.Executor) error) error {
	return fn(nil)
}

type aiAuditRecorder struct {
	calls   int
	actions []string
	targets []string
}

func (a *aiAuditRecorder) Record(_ context.Context, _ database.Executor, _ string, action, target string) error {
	a.calls++
	a.actions = append(a.actions, action)
	a.targets = append(a.targets, target)
	return nil
}

type aiProvider struct{ calls int }

func (p *aiProvider) Suggest(context.Context, ai.Request) (ai.Suggestion, error) {
	p.calls++
	return ai.Suggestion{Title: "Suggested title", Body: "Suggested body"}, nil
}

func TestPageAIAssistantRequiresConsentAndOnlyReturnsDraftPreview(t *testing.T) {
	page := pages.Page{ID: "page-1", Locale: "en", Title: "Current title", Body: "Current body"}
	provider := &aiProvider{}
	auditLog := &aiAuditRecorder{}
	handler := Handler{
		Content:   service.ContentService{Pages: aiPageRepository{page}, Transactions: aiTransactions{}},
		Assistant: ai.Assistant{Enabled: true, Provider: provider}, AIAudit: auditLog,
		AdminCollection: adminstudio.Collection{ID: "pages", Label: "Pages", Actions: []adminstudio.Action{{Label: "Open", Href: "/admin/pages"}}, Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}, {ID: "body", Label: "Body", Kind: adminstudio.FieldRichText, Widget: "richtext"}}},
	}
	post := func(consent string) *httptest.ResponseRecorder {
		form := url.Values{"operation": {"refine"}, "consent": {consent}, "_csrf": {"csrf-test"}}
		request := httptest.NewRequest(http.MethodPost, "/admin/pages/page-1/ai-suggest", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf-test"})
		request.SetPathValue("id", "page-1")
		request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor-1", Email: "editor@example.test", Role: auth.RoleEditor, Active: true}))
		response := httptest.NewRecorder()
		security.CSRFMiddleware(false)(http.HandlerFunc(handler.adminAISuggest)).ServeHTTP(response, request)
		return response
	}
	denied := post("")
	if denied.Code != http.StatusBadRequest || provider.calls != 0 || auditLog.calls != 0 {
		t.Fatalf("missing consent status=%d provider=%d audit=%d", denied.Code, provider.calls, auditLog.calls)
	}
	accepted := post("yes")
	if accepted.Code != http.StatusOK || provider.calls != 1 || auditLog.calls != 1 {
		t.Fatalf("accepted request status=%d provider=%d audit=%d body=%q", accepted.Code, provider.calls, auditLog.calls, accepted.Body.String())
	}
	for _, want := range []string{"Suggested title", "Suggested body", "Current body", "taslağa aktar"} {
		if !strings.Contains(accepted.Body.String(), want) {
			t.Errorf("response missing %q", want)
		}
	}
	applyRequest := httptest.NewRequest(http.MethodPost, "/admin/pages/page-1/ai-applied", nil)
	applyRequest.SetPathValue("id", "page-1")
	applyRequest.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf-test"})
	applyRequest.Header.Set(security.CSRFHeaderName, "csrf-test")
	applyRequest = applyRequest.WithContext(auth.WithUser(applyRequest.Context(), &auth.User{ID: "editor-1", Role: auth.RoleEditor, Active: true}))
	applyResponse := httptest.NewRecorder()
	security.CSRFMiddleware(false)(http.HandlerFunc(handler.adminAIApplied)).ServeHTTP(applyResponse, applyRequest)
	if applyResponse.Code != http.StatusNoContent || auditLog.calls != 2 {
		t.Fatalf("apply audit status=%d audit=%d body=%q", applyResponse.Code, auditLog.calls, applyResponse.Body.String())
	}
	if strings.Join(auditLog.actions, ",") != "ai.suggestion.requested,ai.suggestion.applied" || strings.Join(auditLog.targets, ",") != "page-1,page-1" {
		t.Fatalf("unexpected content-bearing audit data: %+v", auditLog)
	}
}

var _ audit.Writer = (*aiAuditRecorder)(nil)
var _ service.Transactions = aiTransactions{}
var _ service.ContentRepository = aiPageRepository{}
