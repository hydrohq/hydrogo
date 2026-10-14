package contenttypes

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

const revisedFAQ = `id: faq
label: FAQ
title_field: question
public: true
drafts: true
revisions: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true}
  - {id: sort_order, label: Order, kind: number}
`

func TestRevisionsCanBeViewedAndRestoredInTheAdmin(t *testing.T) {
	service, types := newServiceFor(t, revisedFAQ)
	api := AdminAPI{Service: service, Types: types, Actor: func(*http.Request) string { return "editor@example.test" }}
	admin := generatedAdmin(t, api, func(options *adminstudio.GeneratedAdminOptions) {
		options.CanTransition, options.Transition = api.CanTransition, api.Transition
		options.ListRevisions, options.GetRevision = api.ListRevisions, api.GetRevision
		options.CanRestore, options.RestoreRevision = api.CanRestore, api.RestoreRevision
	})
	public := publicMux(PublicAPI{Service: service, Types: types})

	if created := adminPost(t, admin, "/admin/resources/faq/new", url.Values{"id": {"x"}, "question": {"First wording"}, "sort_order": {"1"}}); created.Code != http.StatusSeeOther {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	records, err := service.ListRecords(context.Background(), "faq", 1, 0)
	if err != nil || len(records) != 1 {
		t.Fatal(records, err)
	}
	id := records[0].ID
	base := "/admin/resources/faq/" + id
	if edited := adminPost(t, admin, base, url.Values{"version": {"1"}, "question": {"Second wording"}, "sort_order": {"2"}}); edited.Code != http.StatusSeeOther {
		t.Fatalf("edit: %d %s", edited.Code, edited.Body)
	}
	if published := adminPost(t, admin, base+"/workflow", url.Values{"target": {"published"}, "expected_version": {"2"}}); published.Code != http.StatusSeeOther {
		t.Fatalf("publish: %d %s", published.Code, published.Body)
	}

	status, body := adminGet(t, admin, base)
	for _, version := range []string{"1", "2", "3"} {
		if !strings.Contains(body, base+"/revisions/"+version) {
			t.Fatalf("history lacks version %s: %d\n%s", version, status, body)
		}
	}
	if !strings.Contains(body, "editor@example.test") {
		t.Fatal("history does not name the editor")
	}

	status, body = adminGet(t, admin, base+"/revisions/1")
	if status != http.StatusOK || !strings.Contains(body, "<del>First wording</del>") || !strings.Contains(body, "<ins>Second wording</ins>") || !strings.Contains(body, base+"/revisions/1/restore") {
		t.Fatalf("revision page: %d\n%s", status, body)
	}
	if !strings.Contains(body, "<dt>Status</dt>") {
		t.Fatal("the status change from draft to published is not shown")
	}
	if status, _ := adminGet(t, admin, base+"/revisions/99"); status != http.StatusNotFound {
		t.Fatalf("missing revision: %d", status)
	}

	if stale := adminPost(t, admin, base+"/revisions/1/restore", url.Values{"expected_version": {"2"}}); stale.Code != http.StatusConflict {
		t.Fatalf("stale restore: %d", stale.Code)
	}
	if restored := adminPost(t, admin, base+"/revisions/1/restore", url.Values{"expected_version": {"3"}}); restored.Code != http.StatusSeeOther {
		t.Fatalf("restore: %d %s", restored.Code, restored.Body)
	}
	record, err := service.GetRecord(context.Background(), "faq", id)
	if err != nil || record.Version != 4 || record.Data["question"] != "First wording" || record.Data["sort_order"] != 1.0 || record.Data[StatusField] != StatusPublished {
		t.Fatalf("restored record=%+v err=%v", record, err)
	}
	if recorder, item := call(t, public, http.MethodGet, "/api/v1/content/faq/"+id, ""); recorder.Code != http.StatusOK || item["question"] != "First wording" {
		t.Fatalf("public after restore: %d %v", recorder.Code, item)
	}
	summaries, err := api.ListRevisions(httptestRequest(), adminCollection(t, types), id, 50)
	if err != nil || len(summaries) != 4 || summaries[0].Version != 4 || summaries[0].State != StatusPublished || summaries[0].Actor != "editor@example.test" {
		t.Fatalf("summaries=%+v err=%v", summaries, err)
	}
}

func TestRestoringADraftRevisionDoesNotUnpublish(t *testing.T) {
	service, types := newServiceFor(t, revisedFAQ)
	api := AdminAPI{Service: service, Types: types}
	ctx := context.Background()
	record, err := service.CreateRecord(ctx, "faq", map[string]any{"question": "Draft text", StatusField: StatusDraft})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateRecordVersion(ctx, "faq", record.ID, map[string]any{"question": "Live text", StatusField: StatusPublished}, 1); err != nil {
		t.Fatal(err)
	}
	collection := adminCollection(t, types)
	if err := api.RestoreRevision(httptestRequest(), collection, record.ID, 1, 2); err != nil {
		t.Fatal(err)
	}
	restored, _ := service.GetRecord(ctx, "faq", record.ID)
	if restored.Data["question"] != "Draft text" || restored.Data[StatusField] != StatusPublished {
		t.Fatalf("restored=%+v", restored.Data)
	}
	types[0].Revisions = false
	if api.CanRestore(httptestRequest(), collection, record.ID, 1) {
		t.Fatal("restore offered for a type without revisions")
	}
}

func httptestRequest() *http.Request {
	request, _ := http.NewRequest(http.MethodGet, "/", nil)
	return request
}

func adminCollection(t *testing.T, types []Definition) adminstudio.Collection {
	t.Helper()
	return types[0].AdminCollection()
}
