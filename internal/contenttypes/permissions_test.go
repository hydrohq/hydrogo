package contenttypes

import (
	"net/http"
	"testing"
)

const ledgerYAML = `id: ledger
label: Ledger
permissions: {read: ledger.read, write: ledger.write}
fields:
  - {id: note, label: Note, kind: text}
`

func TestTypePermissionsReplaceTheGenericOnes(t *testing.T) {
	service, types := newServiceFor(t, ledgerYAML)
	base := AdminAPIPath + "/ledger"
	grant := func(names ...string) func(*http.Request, string) bool {
		return func(_ *http.Request, p string) bool {
			for _, n := range names {
				if n == p {
					return true
				}
			}
			return false
		}
	}
	// Generic content access alone is not enough for a guarded type.
	generic := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: grant()})
	if response, _ := call(t, generic, http.MethodGet, base, ""); response.Code != http.StatusForbidden {
		t.Fatalf("read without ledger.read: %d", response.Code)
	}
	reader := adminMux(AdminAPI{Service: service, Types: types, Authorize: func(*http.Request, bool) bool { return false }, Can: grant("ledger.read")})
	if response, _ := call(t, reader, http.MethodGet, base, ""); response.Code != http.StatusOK {
		t.Fatalf("read with ledger.read: %d %s", response.Code, response.Body)
	}
	if response, _ := call(t, reader, http.MethodPost, base, `{"note":"x"}`); response.Code != http.StatusForbidden {
		t.Fatalf("write with only ledger.read: %d", response.Code)
	}
	if response, _ := call(t, reader, http.MethodPost, BatchPath, `{"operations":[{"op":"create","type":"ledger","fields":{"note":"x"}}]}`); response.Code != http.StatusForbidden {
		t.Fatalf("batch with only ledger.read: %d %s", response.Code, response.Body)
	}
	writer := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: grant("ledger.write")})
	if response, _ := call(t, writer, http.MethodPost, base, `{"note":"x"}`); response.Code != http.StatusCreated {
		t.Fatalf("write with ledger.write: %d %s", response.Code, response.Body)
	}
	if got := PermissionNames(types); len(got) != 2 {
		t.Fatalf("catalog names = %v", got)
	}
}

func TestTypePermissionNamesAreChecked(t *testing.T) {
	for _, name := range []string{"admin.content.write", "Bad Name", "nodot"} {
		d := Definition{ID: "x", Label: "X", Permissions: Permissions{Read: name}, Fields: []Field{{ID: "a", Label: "A", Kind: "text"}}}
		if err := d.Validate(); err == nil {
			t.Fatalf("permission %q accepted", name)
		}
	}
}
