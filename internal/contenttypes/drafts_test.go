package contenttypes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

const draftsFAQ = `id: faq
label: FAQ
title_field: question
public: true
drafts: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
`

func adminPost(t *testing.T, handler http.Handler, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	form.Set("_csrf", "token")
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func publicMux(service PublicAPI) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/content/{type}", service.List)
	mux.HandleFunc("GET /api/v1/content/{type}/{id}", service.Get)
	return mux
}

func publicCount(t *testing.T, mux http.Handler, target string) int {
	t.Helper()
	recorder, body := call(t, mux, http.MethodGet, target, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s: %d %s", target, recorder.Code, recorder.Body)
	}
	return len(body["items"].([]any))
}

func TestDraftsRejectAStatusField(t *testing.T) {
	dir := t.TempDir()
	content := "id: x\nlabel: X\ndrafts: true\nfields: [{id: status, label: S, kind: text}]\n"
	if err := os.WriteFile(filepath.Join(dir, "x.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("a drafts type accepted its own status field")
	}
}

func TestDraftsArePublishedThroughTheWorkflowAndNeverPublic(t *testing.T) {
	service, types := newServiceFor(t, draftsFAQ)
	api := AdminAPI{Service: service, Types: types}
	admin := generatedAdmin(t, api, func(options *adminstudio.GeneratedAdminOptions) {
		options.CanTransition, options.Transition = api.CanTransition, api.Transition
	})
	public := publicMux(PublicAPI{Service: service, Types: types})

	created := adminPost(t, admin, "/admin/resources/faq/new", url.Values{"id": {"x"}, "question": {"Secret launch date?"}, "category": {"general"}})
	if created.Code != http.StatusSeeOther {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	records, err := service.ListRecords(context.Background(), "faq", 10, 0)
	if err != nil || len(records) != 1 || records[0].Data[StatusField] != StatusDraft {
		t.Fatalf("new record should be a draft: %+v %v", records, err)
	}
	id := records[0].ID

	assertHidden := func(stage string) {
		t.Helper()
		for _, target := range []string{"/api/v1/content/faq", "/api/v1/content/faq?search=secret", "/api/v1/content/faq?sort=-question", "/api/v1/content/faq?filter[category]=general"} {
			if count := publicCount(t, public, target); count != 0 {
				t.Fatalf("%s: %s returned %d draft(s)", stage, target, count)
			}
		}
		if recorder, _ := call(t, public, http.MethodGet, "/api/v1/content/faq/"+id, ""); recorder.Code != http.StatusNotFound {
			t.Fatalf("%s: draft by id: %d", stage, recorder.Code)
		}
		if recorder, _ := call(t, public, http.MethodGet, "/api/v1/content/faq?filter[status]=draft", ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status filter: %d", stage, recorder.Code)
		}
	}
	assertHidden("new draft")

	status, body := adminGet(t, admin, "/admin/resources/faq/"+id)
	if status != http.StatusOK || !strings.Contains(body, `value="published"`) || !strings.Contains(body, "/admin/resources/faq/"+id+"/workflow") {
		t.Fatalf("edit page lacks a publish action: %d\n%s", status, body)
	}
	if status, body := adminGet(t, admin, "/admin/resources/faq?filter%5Bstatus%5D=draft"); status != http.StatusOK || !strings.Contains(body, "Secret launch date?") {
		t.Fatalf("admin status filter: %d", status)
	}

	// The API ignores a status in the payload: saving cannot publish.
	if recorder, _ := call(t, api, http.MethodPut, "/api/v1/faq/"+id, `{"version":1,"question":"Secret launch date?","status":"published"}`); recorder.Code != http.StatusOK {
		t.Fatalf("save: %d %s", recorder.Code, recorder.Body)
	}
	assertHidden("save with status in payload")

	if stale := adminPost(t, admin, "/admin/resources/faq/"+id+"/workflow", url.Values{"target": {"published"}, "expected_version": {"1"}}); stale.Code != http.StatusConflict {
		t.Fatalf("stale publish: %d", stale.Code)
	}
	if published := adminPost(t, admin, "/admin/resources/faq/"+id+"/workflow", url.Values{"target": {"published"}, "expected_version": {"2"}}); published.Code != http.StatusSeeOther {
		t.Fatalf("publish: %d %s", published.Code, published.Body)
	}
	if count := publicCount(t, public, "/api/v1/content/faq?search=secret"); count != 1 {
		t.Fatalf("published record missing: %d", count)
	}
	if recorder, record := call(t, public, http.MethodGet, "/api/v1/content/faq/"+id, ""); recorder.Code != http.StatusOK || record[StatusField] != StatusPublished {
		t.Fatalf("published by id: %d %v", recorder.Code, record)
	}

	// An edit keeps the record published.
	edited := adminPost(t, admin, "/admin/resources/faq/"+id, url.Values{"version": {"3"}, "question": {"Launch date?"}, "category": {"billing"}, StatusField: {"draft"}})
	if edited.Code != http.StatusSeeOther {
		t.Fatalf("edit: %d %s", edited.Code, edited.Body)
	}
	if count := publicCount(t, public, "/api/v1/content/faq"); count != 1 {
		t.Fatalf("edit unpublished the record: %d", count)
	}

	if unpublished := adminPost(t, admin, "/admin/resources/faq/"+id+"/workflow", url.Values{"target": {"draft"}, "expected_version": {"4"}}); unpublished.Code != http.StatusSeeOther {
		t.Fatalf("unpublish: %d %s", unpublished.Code, unpublished.Body)
	}
	assertHidden("unpublished")
}

func TestRecordsWithoutStatusAreDrafts(t *testing.T) {
	service, types := newServiceFor(t, draftsFAQ)
	record, err := service.CreateRecord(context.Background(), "faq", map[string]any{"question": "Old record"})
	if err != nil {
		t.Fatal(err)
	}
	public := publicMux(PublicAPI{Service: service, Types: types})
	if count := publicCount(t, public, "/api/v1/content/faq"); count != 0 {
		t.Fatalf("record without status is public: %d", count)
	}
	if recorder, body := call(t, AdminAPI{Service: service, Types: types}, http.MethodGet, "/api/v1/faq/"+record.ID, ""); recorder.Code != http.StatusOK || body[StatusField] != StatusDraft {
		t.Fatalf("admin view: %d %v", recorder.Code, body)
	}
}
