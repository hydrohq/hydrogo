package contenttypes

import (
	"net/http"
	"testing"
)

const dealsYAML = `id: mydeals
label: Deals
access: {owner: owner, all: deals.read_all}
fields:
  - {id: name, label: Name, kind: text}
  - {id: owner, label: Owner, kind: text, filterable: true}
`

func TestRowAccessLimitsUsersToTheirOwnRecords(t *testing.T) {
	service, types := newServiceFor(t, dealsYAML)
	base := AdminAPIPath + "/mydeals"
	as := func(email string, perms ...string) http.Handler {
		return adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll,
			Subject: func(*http.Request) string { return email },
			Can: func(_ *http.Request, p string) bool {
				for _, x := range perms {
					if x == p {
						return true
					}
				}
				return false
			}})
	}
	ann, bob, boss := as("ann@x.test"), as("bob@x.test"), as("boss@x.test", "deals.read_all")

	// Creating records out of someone else's name is not possible.
	created, body := call(t, ann, http.MethodPost, base, `{"name":"A1","owner":"bob@x.test"}`)
	if created.Code != http.StatusCreated || data(t, body)["owner"] != "ann@x.test" {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	annID := data(t, body)["id"].(string)
	if r, _ := call(t, bob, http.MethodPost, base, `{"name":"B1"}`); r.Code != http.StatusCreated {
		t.Fatalf("bob create: %d", r.Code)
	}

	if _, body := call(t, ann, http.MethodGet, base, ""); len(body["data"].([]any)) != 1 {
		t.Fatalf("ann sees %v", body["data"])
	}
	if _, body := call(t, boss, http.MethodGet, base, ""); len(body["data"].([]any)) != 2 {
		t.Fatalf("boss sees %v", body["data"])
	}
	record := base + "/" + annID
	if r, _ := call(t, bob, http.MethodGet, record, ""); r.Code != http.StatusNotFound {
		t.Fatalf("bob reads ann's record: %d", r.Code)
	}
	if r, _ := call(t, bob, http.MethodPut, record, `{"name":"stolen"}`); r.Code != http.StatusNotFound {
		t.Fatalf("bob edits ann's record: %d", r.Code)
	}
	if r, _ := call(t, bob, http.MethodDelete, record, ""); r.Code != http.StatusNotFound {
		t.Fatalf("bob deletes ann's record: %d", r.Code)
	}
	if r, _ := call(t, bob, http.MethodPost, BatchPath, `{"operations":[{"op":"delete","type":"mydeals","id":"`+annID+`"}]}`); r.Code != http.StatusNotFound {
		t.Fatalf("bob batch-deletes ann's record: %d %s", r.Code, r.Body)
	}
	if r, body := call(t, ann, http.MethodPut, record, `{"name":"A2","owner":"bob@x.test"}`); r.Code != http.StatusOK || data(t, body)["owner"] != "ann@x.test" {
		t.Fatalf("ann edits own record: %d %s", r.Code, r.Body)
	}
	if r, _ := call(t, boss, http.MethodGet, record, ""); r.Code != http.StatusOK {
		t.Fatalf("boss reads: %d", r.Code)
	}
	if r, _ := call(t, as(""), http.MethodGet, base, ""); r.Code != http.StatusForbidden {
		t.Fatalf("anonymous subject: %d", r.Code)
	}
}

func TestAccessDefinitionIsChecked(t *testing.T) {
	d := Definition{ID: "x", Label: "X", Access: &Access{Owner: "missing", All: "x.all"}, Fields: []Field{{ID: "a", Label: "A", Kind: "text"}}}
	if d.Validate() == nil {
		t.Fatal("unknown owner field accepted")
	}
	d.Access = &Access{Owner: "a"}
	if d.Validate() == nil {
		t.Fatal("missing all permission accepted")
	}
}
