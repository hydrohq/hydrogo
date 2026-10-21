package contenttypes

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const batchCompanies = `id: companies
label: Companies
title_field: name
fields:
  - {id: name, label: Name, kind: text, required: true, unique: true, list: true}
`

const batchContacts = `id: contacts
label: Contacts
title_field: full_name
fields:
  - {id: full_name, label: Name, kind: text, required: true, list: true}
  - {id: company, label: Company, kind: relation, to: companies, list: true}
`

func batchAdmin(t *testing.T) (http.Handler, AdminAPI) {
	t.Helper()
	service, types := newServiceFor(t, batchCompanies, batchContacts)
	api := AdminAPI{Service: service, Types: types, Authorize: allowAll}
	return adminMux(api), api
}

func count(t *testing.T, admin http.Handler, typeID string) int {
	t.Helper()
	_, body := call(t, admin, http.MethodGet, AdminAPIPath+"/"+typeID, "")
	return len(body["data"].([]any))
}

func TestBatchCreatesRelatedRecordsAtomically(t *testing.T) {
	admin, _ := batchAdmin(t)
	response, body := call(t, admin, http.MethodPost, BatchPath, `{"operations":[
		{"op":"create","type":"companies","fields":{"name":"Acme"}},
		{"op":"create","type":"contacts","fields":{"full_name":"Al","company":"$ref:0"}}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("batch: %d %s", response.Code, response.Body)
	}
	results := data(t, body)["results"].([]any)
	company := results[0].(map[string]any)["id"].(string)
	contact := results[1].(map[string]any)["record"].(map[string]any)
	if contact["company"] != company {
		t.Fatalf("$ref not resolved: contact company=%v want %s", contact["company"], company)
	}
	if count(t, admin, "companies") != 1 || count(t, admin, "contacts") != 1 {
		t.Fatal("records were not saved")
	}
}

func TestBatchRollsBackEverythingOnFailure(t *testing.T) {
	admin, _ := batchAdmin(t)
	response, body := call(t, admin, http.MethodPost, BatchPath, `{"operations":[
		{"op":"create","type":"companies","fields":{"name":"Acme"}},
		{"op":"create","type":"contacts","fields":{"company":"$ref:0"}}]}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d %s", response.Code, response.Body)
	}
	fields := body["fields"].(map[string]any)
	if fields["operations[1].full_name"] != "required" {
		t.Fatalf("error does not name the failing operation: %v", body)
	}
	if got := count(t, admin, "companies"); got != 0 {
		t.Fatalf("first operation was not rolled back: %d companies", got)
	}
}

func TestBatchUpdatesAndDeletesWithVersionChecks(t *testing.T) {
	admin, _ := batchAdmin(t)
	_, created := call(t, admin, http.MethodPost, AdminAPIPath+"/companies", `{"name":"Old"}`)
	id := data(t, created)["id"].(string)

	stale := fmt.Sprintf(`{"operations":[{"op":"update","type":"companies","id":%q,"version":9,"fields":{"name":"New"}}]}`, id)
	if response, _ := call(t, admin, http.MethodPost, BatchPath, stale); response.Code != http.StatusConflict {
		t.Fatalf("stale version: %d %s", response.Code, response.Body)
	}
	good := fmt.Sprintf(`{"operations":[{"op":"update","type":"companies","id":%q,"version":1,"fields":{"name":"New"}},{"op":"delete","type":"companies","id":%q}]}`, id, id)
	if response, _ := call(t, admin, http.MethodPost, BatchPath, good); response.Code != http.StatusOK {
		t.Fatalf("update+delete: %d %s", response.Code, response.Body)
	}
	if count(t, admin, "companies") != 0 {
		t.Fatal("record not deleted")
	}
}

func TestBatchRejectsBadRequests(t *testing.T) {
	admin, _ := batchAdmin(t)
	cases := map[string]string{
		"empty":          `{"operations":[]}`,
		"unknown type":   `{"operations":[{"op":"create","type":"nope","fields":{}}]}`,
		"unknown op":     `{"operations":[{"op":"merge","type":"companies","fields":{}}]}`,
		"forward ref":    `{"operations":[{"op":"create","type":"contacts","fields":{"full_name":"A","company":"$ref:1"}}]}`,
		"create with id": `{"operations":[{"op":"create","type":"companies","id":"x","fields":{"name":"A"}}]}`,
		"update no id":   `{"operations":[{"op":"update","type":"companies","fields":{"name":"A"}}]}`,
		"too many":       `{"operations":[` + strings.Repeat(`{"op":"create","type":"companies","fields":{"name":"A"}},`, MaxBatchOperations) + `{"op":"create","type":"companies","fields":{"name":"B"}}]}`,
	}
	for name, body := range cases {
		if response, _ := call(t, admin, http.MethodPost, BatchPath, body); response.Code < 400 || response.Code >= 500 {
			t.Errorf("%s: status %d %s", name, response.Code, response.Body)
		}
	}
	if count(t, admin, "companies") != 0 {
		t.Fatal("a rejected batch saved records")
	}
}

func TestBatchRequiresWritePermission(t *testing.T) {
	service, types := newServiceFor(t, batchCompanies)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: readOnly})
	if response, _ := call(t, admin, http.MethodPost, BatchPath, `{"operations":[{"op":"create","type":"companies","fields":{"name":"A"}}]}`); response.Code != http.StatusForbidden {
		t.Fatalf("status %d", response.Code)
	}
}
