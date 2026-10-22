package contenttypes

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const ordersYAML = `id: orders
label: Orders
title_field: status
rules:
  - assert: "quantity > 0"
    message: "Quantity must be positive"
  - when: "status == 'posted'"
    readonly: true
    message: "Posted orders are locked"
states:
  field: status
  initial: draft
  transitions:
    - {from: draft, to: approved, permission: order.approve}
    - {from: approved, to: posted, when: "total > 0", message: "Nothing to post"}
    - {from: posted, to: void}
fields:
  - {id: status, label: Status, kind: select, options: [draft, approved, posted, void], list: true, filterable: true}
  - {id: quantity, label: Quantity, kind: number}
  - {id: price, label: Price, kind: number, decimals: 2}
  - {id: total, label: Total, kind: number, decimals: 2, computed: "quantity * price"}
`

func TestStatesAndRulesThroughTheAdminAPI(t *testing.T) {
	service, types := newServiceFor(t, ordersYAML)
	base := AdminAPIPath + "/orders"

	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: func(_ *http.Request, p string) bool { return p == "order.approve" }})
	if response, body := call(t, admin, http.MethodPost, base, `{"quantity":0,"price":3}`); response.Code != http.StatusUnprocessableEntity || body["message"] != "Quantity must be positive" {
		t.Fatalf("assert: %d %s", response.Code, response.Body)
	}
	created, body := call(t, admin, http.MethodPost, base, `{"quantity":2,"price":3.5,"status":"posted"}`)
	if created.Code != http.StatusCreated || data(t, body)["status"] != "draft" {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	id := data(t, body)["id"].(string)
	record := base + "/" + id

	if response, _ := call(t, admin, http.MethodPost, record+"/state", `{"to":"posted"}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("draft to posted: %d %s", response.Code, response.Body)
	}
	approved, body := call(t, admin, http.MethodPost, record+"/state", `{"to":"approved","version":1}`)
	if approved.Code != http.StatusOK || data(t, body)["status"] != "approved" {
		t.Fatalf("approve: %d %s", approved.Code, approved.Body)
	}
	if response, body := call(t, admin, http.MethodPost, record+"/state", `{"to":"posted"}`); response.Code != http.StatusOK || data(t, body)["status"] != "posted" {
		t.Fatalf("post: %d %s", response.Code, response.Body)
	}
	if response, body := call(t, admin, http.MethodPut, record, `{"quantity":5,"price":1}`); response.Code != http.StatusUnprocessableEntity || body["message"] != "Posted orders are locked" {
		t.Fatalf("edit posted: %d %s", response.Code, response.Body)
	}
	if response, _ := call(t, admin, http.MethodDelete, record, ""); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("delete posted: %d %s", response.Code, response.Body)
	}
	if response, body := call(t, admin, http.MethodPost, record+"/state", `{"to":"void"}`); response.Code != http.StatusOK || data(t, body)["status"] != "void" {
		t.Fatalf("void: %d %s", response.Code, response.Body)
	}

	// Without the permission the approval is refused; without a Can hook too.
	other, body := call(t, admin, http.MethodPost, base, `{"quantity":1,"price":1}`)
	otherID := data(t, body)["id"].(string)
	if other.Code != http.StatusCreated {
		t.Fatal(other.Body)
	}
	for name, can := range map[string]func(*http.Request, string) bool{"denied": func(*http.Request, string) bool { return false }, "no hook": nil} {
		restricted := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: can})
		if response, _ := call(t, restricted, http.MethodPost, fmt.Sprintf("%s/%s/state", base, otherID), `{"to":"approved"}`); response.Code != http.StatusForbidden {
			t.Errorf("%s: %d %s", name, response.Code, response.Body)
		}
	}

	// A type without states answers 404 to a state change.
	plain, plainTypes := newServiceFor(t, adminFAQ)
	faq := adminMux(AdminAPI{Service: plain, Types: plainTypes, Authorize: allowAll})
	_, created2 := call(t, faq, http.MethodPost, AdminAPIPath+"/faq", `{"question":"Q?"}`)
	if response, _ := call(t, faq, http.MethodPost, AdminAPIPath+"/faq/"+data(t, created2)["id"].(string)+"/state", `{"to":"x"}`); response.Code != http.StatusNotFound {
		t.Fatalf("no states: %d", response.Code)
	}
}

func TestRulesAndStatesAreCheckedWhenTheTypeLoads(t *testing.T) {
	bad := `id: bad
label: Bad
states: {field: status, initial: draft, transitions: [{from: draft, to: nowhere}]}
fields:
  - {id: status, label: Status, kind: select, options: [draft, done]}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("a transition to an unknown state was accepted")
	}
}
