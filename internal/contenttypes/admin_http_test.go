package contenttypes

import (
	"context"
	"net/http"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

const adminFAQ = `id: faq
label: FAQ
title_field: question
public: true
drafts: true
revisions: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, unique: true, searchable: true}
  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
  - {id: asked_on, label: Asked on, kind: date}
`

func allowAll(*http.Request, bool) bool { return true }

func readOnly(_ *http.Request, write bool) bool { return !write }

// adminMux mounts the admin routes without the session middleware; the
// Authorize callback stands in for the permission check.
func adminMux(api AdminAPI) http.Handler {
	handler, err := router.New(api.Routes(), func(*http.Request) bool { return true })
	if err != nil {
		panic(err)
	}
	return handler
}

func data(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	record, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object in %v", body)
	}
	return record
}

func TestAdminContentAPICreatesUpdatesAndDeletes(t *testing.T) {
	service, types := newServiceFor(t, adminFAQ)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	base := AdminAPIPath + "/faq"

	created, body := call(t, admin, http.MethodPost, base, `{"id":"ignored","question":"How?","category":"general","asked_on":"2026-10-14","status":"published"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	record := data(t, body)
	if record["question"] != "How?" || record["version"] != float64(1) || record["asked_on"] != "2026-10-14" {
		t.Fatalf("created record=%v", record)
	}
	if record[StatusField] != StatusDraft {
		t.Fatalf("new record is not a draft: %v", record[StatusField])
	}
	id := record["id"].(string)
	if id == "ignored" {
		t.Fatal("client-supplied id was used")
	}

	if got, body := call(t, admin, http.MethodGet, base+"/"+id, ""); got.Code != http.StatusOK || data(t, body)["id"] != id {
		t.Fatalf("get: %d %s", got.Code, got.Body)
	}
	updated, body := call(t, admin, http.MethodPut, base+"/"+id, `{"version":1,"question":"How exactly?"}`)
	if updated.Code != http.StatusOK || data(t, body)["version"] != float64(2) {
		t.Fatalf("update: %d %s", updated.Code, updated.Body)
	}
	if stale, body := call(t, admin, http.MethodPut, base+"/"+id, `{"version":1,"question":"Stale"}`); stale.Code != http.StatusConflict || body["code"] != "conflict" || body["request_id"] == nil {
		t.Fatalf("stale update: %d %s", stale.Code, stale.Body)
	}

	invalid, body := call(t, admin, http.MethodPost, base, `{"category":"general"}`)
	if invalid.Code != http.StatusUnprocessableEntity || body["code"] != "unprocessable_entity" {
		t.Fatalf("missing required: %d %s", invalid.Code, invalid.Body)
	}
	if fields := body["fields"].(map[string]any); fields["question"] != "required" {
		t.Fatalf("fields=%v", fields)
	}
	if duplicate, body := call(t, admin, http.MethodPost, base, `{"question":"How exactly?"}`); duplicate.Code != http.StatusUnprocessableEntity || body["fields"].(map[string]any)["question"] != "taken" {
		t.Fatalf("duplicate: %d %s", duplicate.Code, duplicate.Body)
	}
	if malformed, _ := call(t, admin, http.MethodPost, base, `{"question":`); malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed body: %d", malformed.Code)
	}

	if missing, _ := call(t, admin, http.MethodGet, AdminAPIPath+"/nope", ""); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown type: %d", missing.Code)
	}
	if missing, _ := call(t, admin, http.MethodGet, base+"/nope", ""); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown record: %d", missing.Code)
	}
	if deleted, _ := call(t, admin, http.MethodDelete, base+"/"+id, ""); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", deleted.Code, deleted.Body)
	}
	if gone, _ := call(t, admin, http.MethodGet, base+"/"+id, ""); gone.Code != http.StatusNotFound {
		t.Fatalf("deleted record still served: %d", gone.Code)
	}
}

func TestAdminContentAPIAuthorizesReadsAndWrites(t *testing.T) {
	service, types := newServiceFor(t, adminFAQ)
	viewer := adminMux(AdminAPI{Service: service, Types: types, Authorize: readOnly})
	if list, _ := call(t, viewer, http.MethodGet, AdminAPIPath+"/faq", ""); list.Code != http.StatusOK {
		t.Fatalf("viewer list: %d", list.Code)
	}
	if created, body := call(t, viewer, http.MethodPost, AdminAPIPath+"/faq", `{"question":"No"}`); created.Code != http.StatusForbidden || body["code"] != "forbidden" {
		t.Fatalf("viewer create: %d %s", created.Code, created.Body)
	}
	denied := adminMux(AdminAPI{Service: service, Types: types})
	if list, _ := call(t, denied, http.MethodGet, AdminAPIPath+"/faq", ""); list.Code != http.StatusForbidden {
		t.Fatalf("no authorizer: %d", list.Code)
	}
}

func TestAdminContentAPIListsFiltersAndPages(t *testing.T) {
	service, types := newServiceFor(t, adminFAQ)
	ctx := context.Background()
	for _, data := range []map[string]any{
		{"question": "B question", "category": "general", StatusField: StatusPublished},
		{"question": "A question", "category": "billing", StatusField: StatusDraft},
		{"question": "C question", "category": "general", StatusField: StatusPublished},
	} {
		if _, err := service.CreateRecord(ctx, "faq", data); err != nil {
			t.Fatal(err)
		}
	}
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	questions := func(target string) ([]string, map[string]any) {
		t.Helper()
		recorder, body := call(t, admin, http.MethodGet, target, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", target, recorder.Code, recorder.Body)
		}
		result := []string{}
		for _, item := range body["data"].([]any) {
			result = append(result, item.(map[string]any)["question"].(string))
		}
		return result, body["metadata"].(map[string]any)
	}
	if got, _ := questions(AdminAPIPath + "/faq?sort=question"); len(got) != 3 || got[0] != "A question" {
		t.Fatalf("sorted list: %v", got)
	}
	if got, _ := questions(AdminAPIPath + "/faq?status=draft"); len(got) != 1 || got[0] != "A question" {
		t.Fatalf("drafts: %v", got)
	}
	if got, _ := questions(AdminAPIPath + "/faq?status=published&filter[category]=general&sort=-question"); len(got) != 2 || got[0] != "C question" {
		t.Fatalf("published general: %v", got)
	}
	if got, _ := questions(AdminAPIPath + "/faq?search=a+quest"); len(got) != 1 {
		t.Fatalf("search: %v", got)
	}
	got, meta := questions(AdminAPIPath + "/faq?limit=2&sort=question")
	if len(got) != 2 || meta["has_more"] != true || meta["limit"] != float64(2) {
		t.Fatalf("page 1: %v %v", got, meta)
	}
	got, meta = questions(AdminAPIPath + "/faq?limit=2&offset=2&sort=question")
	if len(got) != 1 || meta["has_more"] != false || meta["offset"] != float64(2) {
		t.Fatalf("page 2: %v %v", got, meta)
	}
	for _, target := range []string{"/faq?status=archived", "/faq?filter[question]=x", "/faq?sort=nope", "/faq?locale=xx"} {
		if recorder, _ := call(t, admin, http.MethodGet, AdminAPIPath+target, ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", target, recorder.Code)
		}
	}
}

func TestAdminContentAPIWorkflowAndRevisions(t *testing.T) {
	service, types := newServiceFor(t, adminFAQ)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Actor: func(*http.Request) string { return "editor@example.test" }})
	base := AdminAPIPath + "/faq"
	_, body := call(t, admin, http.MethodPost, base, `{"question":"Draft text"}`)
	id := data(t, body)["id"].(string)
	public := publicMux(PublicAPI{Service: service, Types: types})
	if count := publicCount(t, public, "/api/v1/content/faq"); count != 0 {
		t.Fatalf("draft is public: %d", count)
	}

	published, body := call(t, admin, http.MethodPost, base+"/"+id+"/workflow", `{"target":"published","version":1}`)
	if published.Code != http.StatusOK || data(t, body)[StatusField] != StatusPublished || data(t, body)["version"] != float64(2) {
		t.Fatalf("publish: %d %s", published.Code, published.Body)
	}
	if count := publicCount(t, public, "/api/v1/content/faq"); count != 1 {
		t.Fatalf("published record not public: %d", count)
	}
	if stale, _ := call(t, admin, http.MethodPost, base+"/"+id+"/workflow", `{"target":"draft","version":1}`); stale.Code != http.StatusConflict {
		t.Fatalf("stale transition: %d", stale.Code)
	}
	if bad, _ := call(t, admin, http.MethodPost, base+"/"+id+"/workflow", `{"target":"archived","version":2}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("unknown target: %d", bad.Code)
	}

	if updated, _ := call(t, admin, http.MethodPut, base+"/"+id, `{"version":2,"question":"Live text"}`); updated.Code != http.StatusOK {
		t.Fatalf("update: %d %s", updated.Code, updated.Body)
	}
	listed, body := call(t, admin, http.MethodGet, base+"/"+id+"/revisions", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("revisions: %d %s", listed.Code, listed.Body)
	}
	revisions := body["data"].([]any)
	if len(revisions) != 3 || revisions[0].(map[string]any)["version"] != float64(3) || revisions[0].(map[string]any)["actor"] != "editor@example.test" {
		t.Fatalf("revisions=%v", revisions)
	}
	snapshot, body := call(t, admin, http.MethodGet, base+"/"+id+"/revisions/1", "")
	if snapshot.Code != http.StatusOK || data(t, body)["fields"].(map[string]any)["question"] != "Draft text" {
		t.Fatalf("revision 1: %d %s", snapshot.Code, snapshot.Body)
	}
	if missing, _ := call(t, admin, http.MethodGet, base+"/"+id+"/revisions/99", ""); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown revision: %d", missing.Code)
	}
	restored, body := call(t, admin, http.MethodPost, base+"/"+id+"/revisions/1/restore", `{"version":3}`)
	if restored.Code != http.StatusOK {
		t.Fatalf("restore: %d %s", restored.Code, restored.Body)
	}
	if record := data(t, body); record["question"] != "Draft text" || record[StatusField] != StatusPublished || record["version"] != float64(4) {
		t.Fatalf("restored=%v", record)
	}
	if stale, _ := call(t, admin, http.MethodPost, base+"/"+id+"/revisions/1/restore", `{"version":3}`); stale.Code != http.StatusConflict {
		t.Fatalf("stale restore: %d", stale.Code)
	}
}

func TestAdminContentAPILocales(t *testing.T) {
	service, types := newServiceFor(t, localizedGuide)
	locales := NewLocales(true, "tr", []string{"tr", "en"})
	admin := adminMux(AdminAPI{Service: service, Types: types, Locales: locales, Authorize: allowAll})
	base := AdminAPIPath + "/guide"
	_, body := call(t, admin, http.MethodPost, base, `{"title":"Başlık","body":"<p>Metin</p>","rank":1}`)
	id := data(t, body)["id"].(string)

	options, body := call(t, admin, http.MethodGet, base+"/"+id+"/locales", "")
	if options.Code != http.StatusOK || data(t, body)["default"] != "tr" {
		t.Fatalf("locales: %d %s", options.Code, options.Body)
	}
	empty, body := call(t, admin, http.MethodGet, base+"/"+id+"/locale/en", "")
	if empty.Code != http.StatusOK || data(t, body)["version"] != float64(0) || data(t, body)["fields"].(map[string]any)["title"] != "" {
		t.Fatalf("untranslated locale: %d %s", empty.Code, empty.Body)
	}
	saved, body := call(t, admin, http.MethodPut, base+"/"+id+"/locale/en", `{"version":0,"fields":{"title":"Title"}}`)
	if saved.Code != http.StatusOK || data(t, body)["version"] != float64(1) || data(t, body)["fields"].(map[string]any)["title"] != "Title" {
		t.Fatalf("save locale: %d %s", saved.Code, saved.Body)
	}
	if stale, _ := call(t, admin, http.MethodPut, base+"/"+id+"/locale/en", `{"version":0,"fields":{"title":"Again"}}`); stale.Code != http.StatusConflict {
		t.Fatalf("stale locale save: %d", stale.Code)
	}
	if unknown, _ := call(t, admin, http.MethodGet, base+"/"+id+"/locale/de", ""); unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown locale: %d", unknown.Code)
	}

	merged, body := call(t, admin, http.MethodGet, base+"/"+id+"?locale=en", "")
	record := data(t, body)
	if merged.Code != http.StatusOK || record["title"] != "Title" || record["body"] != "<p>Metin</p>" {
		t.Fatalf("merged read: %d %v", merged.Code, record)
	}
	list, body := call(t, admin, http.MethodGet, base+"?locale=en", "")
	if list.Code != http.StatusOK || body["data"].([]any)[0].(map[string]any)["title"] != "Title" {
		t.Fatalf("merged list: %d %s", list.Code, list.Body)
	}

	copied, body := call(t, admin, http.MethodPost, base+"/"+id+"/locale/en/copy", `{"from":"tr","version":1}`)
	if copied.Code != http.StatusOK || data(t, body)["fields"].(map[string]any)["title"] != "Başlık" || data(t, body)["version"] != float64(2) {
		t.Fatalf("copy locale: %d %s", copied.Code, copied.Body)
	}
	if same, _ := call(t, admin, http.MethodPost, base+"/"+id+"/locale/en/copy", `{"from":"en","version":2}`); same.Code != http.StatusNotFound {
		t.Fatalf("copy onto itself: %d", same.Code)
	}
}
