package contenttypes

import (
	"net/http"
	"testing"
)

// Expanding a relation into a type the caller may not read leaves it out.
func TestExpandFollowsThePermissionsOfTheRelatedType(t *testing.T) {
	const authors = `id: authors
label: Authors
permissions: {read: authors.read, write: authors.write}
fields:
  - {id: name, label: Name, kind: text}
`
	const books = `id: books
label: Books
fields:
  - {id: title, label: Title, kind: text}
  - {id: author, label: Author, kind: relation, to: authors}
`
	service, types := newServiceFor(t, authors, books)
	all := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: func(*http.Request, string) bool { return true }})
	_, body := call(t, all, http.MethodPost, AdminAPIPath+"/authors", `{"name":"Ada"}`)
	author := data(t, body)["id"].(string)
	_, body = call(t, all, http.MethodPost, AdminAPIPath+"/books", `{"title":"B","author":"`+author+`"}`)
	book := data(t, body)["id"].(string)

	url := AdminAPIPath + "/books/" + book + "?expand=author"
	if _, body := call(t, all, http.MethodGet, url, ""); data(t, body)["expanded"] == nil {
		t.Fatalf("permitted caller got no expansion: %v", body)
	}
	limited := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: func(*http.Request, string) bool { return false }})
	r, body := call(t, limited, http.MethodGet, url, "")
	if r.Code != http.StatusOK {
		t.Fatalf("book read: %d %s", r.Code, r.Body)
	}
	if expanded, _ := data(t, body)["expanded"].(map[string]any); expanded["author"] != nil {
		t.Fatalf("expansion leaked a type the caller may not read: %v", body)
	}
}

// Restoring an old version must not move a record into a closed period.
func TestRestoreIntoClosedPeriodIsRefused(t *testing.T) {
	const posts = `id: ledger
label: Ledger
revisions: true
period_lock: {field: day, closings: closes, through: upto}
fields:
  - {id: day, label: Day, kind: date}
  - {id: note, label: Note, kind: text}
`
	const closes = `id: closes
label: Closes
fields:
  - {id: upto, label: Upto, kind: date}
`
	service, types := newServiceFor(t, posts, closes)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	root := AdminAPIPath
	// Version 1 is dated in September; version 2 moves it to November.
	_, body := call(t, admin, http.MethodPost, root+"/ledger", `{"day":"2026-09-10","note":"a"}`)
	id := data(t, body)["id"].(string)
	if r, _ := call(t, admin, http.MethodPut, root+"/ledger/"+id, `{"version":1,"day":"2026-11-10","note":"b"}`); r.Code != http.StatusOK {
		t.Fatalf("move to november: %d %s", r.Code, r.Body)
	}
	call(t, admin, http.MethodPost, root+"/closes", `{"upto":"2026-09-30"}`)
	if r, _ := call(t, admin, http.MethodPost, root+"/ledger/"+id+"/revisions/1/restore", `{"version":2}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("restore into a closed period: %d %s", r.Code, r.Body)
	}
}
