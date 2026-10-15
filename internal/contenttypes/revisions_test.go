package contenttypes

import (
	"context"
	"net/http"
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
