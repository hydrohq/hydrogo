package contenttypes

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const ledgerCSVYAML = `id: items
label: Items
fields:
  - {id: name, label: Name, kind: text, required: true}
  - {id: qty, label: Qty, kind: number}
  - {id: active, label: Active, kind: boolean}
`

func rawCall(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "text/csv")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCSVImportIsAtomicAndExportDefusesFormulas(t *testing.T) {
	service, types := newServiceFor(t, ledgerCSVYAML)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	base := AdminAPIPath + "/items"

	// One bad row (empty required name) rolls the whole import back.
	bad := "name,qty,active\nA,1,true\n,2,false\n"
	rec := rawCall(t, admin, http.MethodPost, base+"/_import", bad)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "rows[1].name") {
		t.Fatalf("bad import: %d %s", rec.Code, rec.Body)
	}
	if _, body := call(t, admin, http.MethodGet, base, ""); len(body["data"].([]any)) != 0 {
		t.Fatalf("failed import left records: %v", body["data"])
	}
	// Dry run saves nothing; a real run saves all.
	good := "name,qty,active\n=cmd|' /C calc'!A0,1.5,true\nB,2,no\n"
	if rec := rawCall(t, admin, http.MethodPost, base+"/_import?dry_run=1", good); rec.Code != http.StatusOK {
		t.Fatalf("dry run: %d %s", rec.Code, rec.Body)
	}
	if _, body := call(t, admin, http.MethodGet, base, ""); len(body["data"].([]any)) != 0 {
		t.Fatal("dry run saved records")
	}
	if rec := rawCall(t, admin, http.MethodPost, base+"/_import", good); rec.Code != http.StatusOK {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	if rec := rawCall(t, admin, http.MethodPost, base+"/_import", "id,name\n1,x\n"); rec.Code != http.StatusBadRequest {
		t.Fatalf("id column: %d", rec.Code)
	}

	rec = rawCall(t, admin, http.MethodGet, base+"/_export", "")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("export: %d %s", rec.Code, rec.Body)
	}
	rows, err := csv.NewReader(rec.Body).ReadAll()
	if err != nil || len(rows) != 3 || strings.Join(rows[0], ",") != "id,name,qty,active" {
		t.Fatalf("export rows = %v err=%v", rows, err)
	}
	for _, row := range rows[1:] {
		if strings.HasPrefix(row[1], "=") {
			t.Fatalf("formula not defused: %q", row[1])
		}
	}
}
