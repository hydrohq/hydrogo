package contenttypes

import (
	"context"
	"net/http"
	"testing"
)

const (
	expandCompanies = `id: companies
label: Companies
public: true
drafts: true
fields:
  - {id: name, label: Name, kind: text, required: true}
`
	expandSecrets = `id: secrets
label: Secrets
fields:
  - {id: note, label: Note, kind: text}
`
	expandDeals = `id: deals
label: Deals
public: true
fields:
  - {id: name, label: Name, kind: text, required: true}
  - {id: company, label: Company, kind: relation, to: companies, filterable: true}
  - {id: partners, label: Partners, kind: relation, to: companies, many: true}
  - {id: secret, label: Secret, kind: relation, to: secrets}
`
)

func TestExpandLoadsRelatedRecords(t *testing.T) {
	service, types := newServiceFor(t, expandCompanies, expandSecrets, expandDeals)
	ctx := context.Background()
	acme, _ := service.CreateRecord(ctx, "companies", map[string]any{"name": "Acme", "status": StatusPublished})
	draft, _ := service.CreateRecord(ctx, "companies", map[string]any{"name": "Hidden", "status": StatusDraft})
	secret, err := service.CreateRecord(ctx, "secrets", map[string]any{"note": "internal"})
	if err != nil {
		t.Fatal(err)
	}
	deal, err := service.CreateRecord(ctx, "deals", map[string]any{"name": "Big", "company": acme.ID, "partners": []any{acme.ID, draft.ID}, "secret": secret.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecord(ctx, "deals", map[string]any{"name": "Lone"}); err != nil {
		t.Fatal(err)
	}

	public := publicMux(PublicAPI{Service: service, Types: types})
	recorder, body := call(t, public, http.MethodGet, "/api/v1/content/deals?expand=company,partners&filter[company]="+acme.ID, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("%d %s", recorder.Code, recorder.Body)
	}
	items := body["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("filter on a relation returned %d deals", len(items))
	}
	item := items[0].(map[string]any)
	expanded := item["expanded"].(map[string]any)
	if item["company"] != acme.ID || expanded["company"].(map[string]any)["name"] != "Acme" {
		t.Fatalf("company not expanded next to its id: %v", item)
	}
	partners := expanded["partners"].([]any)
	if len(partners) != 1 || partners[0].(map[string]any)["id"] != acme.ID {
		t.Fatalf("public expand must hide related drafts: %v", partners)
	}

	// A single record, and a list entry without relations, have no "expanded".
	recorder, body = call(t, public, http.MethodGet, "/api/v1/content/deals/"+deal.ID+"?expand=company", "")
	if recorder.Code != http.StatusOK || data(t, body)["expanded"].(map[string]any)["company"].(map[string]any)["name"] != "Acme" {
		t.Fatalf("get with expand: %d %s", recorder.Code, recorder.Body)
	}
	if _, body = call(t, public, http.MethodGet, "/api/v1/content/deals?expand=company", ""); len(body["data"].([]any)) != 2 {
		t.Fatalf("list: %v", body)
	}

	for _, target := range []string{
		"/api/v1/content/deals?expand=name",                    // not a relation
		"/api/v1/content/deals?expand=missing",                 // unknown field
		"/api/v1/content/deals?expand=secret",                  // target type is not public
		"/api/v1/content/deals?expand=company&expand=partners", // repeated parameter
	} {
		if recorder, _ := call(t, public, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", target, recorder.Code)
		}
	}

	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	recorder, body = call(t, admin, http.MethodGet, AdminAPIPath+"/deals/"+deal.ID+"?expand=secret,partners", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin expand: %d %s", recorder.Code, recorder.Body)
	}
	adminExpanded := data(t, body)["expanded"].(map[string]any)
	if adminExpanded["secret"].(map[string]any)["note"] != "internal" || len(adminExpanded["partners"].([]any)) != 2 {
		t.Fatalf("admin expand sees non-public types and drafts: %v", adminExpanded)
	}
}
