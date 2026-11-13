package contenttypes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const ownedReviewsYAML = `id: reviews
label: Reviews
title_field: author_name
public: true
drafts: true
owner_field: author_email
access_rules:
  list: "status == 'published'"
  create: "public"
fields:
  - {id: author_name, label: Name, kind: text, required: true}
  - {id: author_email, label: Email, kind: text, private: true}
  - {id: body, label: Body, kind: textarea, required: true}
`

func TestMemberSeesOwnDraftsWithMine(t *testing.T) {
	service, types := newServiceFor(t, ownedReviewsYAML)
	handler := baasPublicMux(PublicAPI{Service: service, Types: types, ResolvePrincipal: testPrincipalResolver})
	alice := Principal{ID: "a", Email: "alice@example.com", Role: "member", IsMember: true}
	bob := Principal{ID: "b", Email: "bob@example.com", Role: "member", IsMember: true}

	post := func(body string) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/content/reviews", strings.NewReader(body)))
		if rec.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "@example.com") {
			t.Fatalf("create response leaked the e-mail: %s", rec.Body)
		}
	}
	postAs := func(p *Principal, body string) string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/content/reviews", strings.NewReader(body))
		if p != nil {
			req = req.WithContext(withTestPrincipal(req.Context(), *p))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "@example.com") {
			t.Fatalf("create response leaked the e-mail: %s", rec.Body)
		}
		return rec.Body.String()
	}
	_ = post
	// The owner is the signed-in member, whatever the body claims.
	postAs(&alice, `{"author_name":"Alice","author_email":"someone-else@example.com","body":"great belt"}`)
	postAs(&bob, `{"author_name":"Bob","body":"fine"}`)
	// Anonymous: no owner at all, and status=published is ignored.
	postAs(nil, `{"author_name":"Eve","author_email":"alice@example.com","status":"published","body":"planted"}`)

	list := func(path string, p *Principal) (int, string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if p != nil {
			req = req.WithContext(withTestPrincipal(req.Context(), *p))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	if _, body := list("/api/v1/content/reviews", nil); strings.Contains(body, "great belt") {
		t.Fatal("a draft must not be public")
	}
	code, body := list("/api/v1/content/reviews?mine=1", &alice)
	if code != http.StatusOK || !strings.Contains(body, "great belt") || strings.Contains(body, "fine") || strings.Contains(body, "@example.com") {
		t.Fatalf("alice should see only her own draft, no e-mails: %d %s", code, body)
	}
	if code, _ := list("/api/v1/content/reviews?mine=1", nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous mine=1: %d, want 401", code)
	}
	if _, body := list("/api/v1/content/reviews", nil); strings.Contains(body, "planted") {
		t.Fatal("a public create must not be able to publish itself")
	}
	if _, body := list("/api/v1/content/reviews?mine=1", &alice); strings.Contains(body, "planted") {
		t.Fatal("an anonymous create must not land in a member's list")
	}
	if _, body := list("/api/v1/content/reviews?mine=1", &bob); strings.Contains(body, "great belt") {
		t.Fatal("bob must not see alice's draft")
	}
}
