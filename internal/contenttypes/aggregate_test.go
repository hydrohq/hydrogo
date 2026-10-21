package contenttypes

import (
	"net/http"
	"testing"
)

const salesYAML = `id: sales
label: Sales
access: {owner: owner, all: sales.all}
fields:
  - {id: stage, label: Stage, kind: select, options: [lead, won], filterable: true}
  - {id: owner, label: Owner, kind: text, filterable: true}
  - {id: amount, label: Amount, kind: number, decimals: 2}
  - {id: margin, label: Margin, kind: number, decimals: 2, permission: sales.margin, filterable: true}
`

func TestAggregateAndFieldPermissions(t *testing.T) {
	service, types := newServiceFor(t, salesYAML)
	base := AdminAPIPath + "/sales"
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
	boss := as("boss@x.test", "sales.all", "sales.margin")
	ann := as("ann@x.test")
	for _, body := range []string{
		`{"stage":"lead","owner":"ann@x.test","amount":0.1,"margin":1}`,
		`{"stage":"lead","owner":"ann@x.test","amount":0.2,"margin":2}`,
		`{"stage":"won","owner":"bob@x.test","amount":10.5,"margin":3}`,
	} {
		if r, _ := call(t, boss, http.MethodPost, base, body); r.Code != http.StatusCreated {
			t.Fatalf("seed: %d %s", r.Code, r.Body)
		}
	}
	_, body := call(t, boss, http.MethodGet, base+"/_aggregate?group_by=stage&sum=amount,margin", "")
	d := data(t, body)
	groups := d["groups"].([]any)
	lead := groups[0].(map[string]any)
	if lead["key"] != "lead" || lead["count"].(float64) != 2 || lead["sums"].(map[string]any)["amount"] != "0.30" {
		t.Fatalf("lead group = %v", lead)
	}
	if d["total"].(map[string]any)["sums"].(map[string]any)["amount"] != "10.80" {
		t.Fatalf("total = %v", d["total"])
	}
	// Ann only sees her own rows and never the margin.
	_, body = call(t, ann, http.MethodGet, base+"/_aggregate?sum=amount", "")
	if got := data(t, body)["total"].(map[string]any); got["count"].(float64) != 2 || got["sums"].(map[string]any)["amount"] != "0.30" {
		t.Fatalf("ann total = %v", got)
	}
	for _, path := range []string{"/_aggregate?sum=margin", "/_aggregate?group_by=margin", "?filter[margin]=1", "?sort=margin", "/_aggregate?sum=stage", "/_aggregate?group_by=amount"} {
		if r, _ := call(t, ann, http.MethodGet, base+path, ""); r.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", path, r.Code, r.Body)
		}
	}
	_, list := call(t, ann, http.MethodGet, base, "")
	for _, item := range list["data"].([]any) {
		if _, leaked := item.(map[string]any)["margin"]; leaked {
			t.Fatalf("margin leaked: %v", item)
		}
	}
	// Saving without the permission keeps the stored margin.
	id := list["data"].([]any)[0].(map[string]any)["id"].(string)
	if r, _ := call(t, ann, http.MethodPut, base+"/"+id, `{"stage":"won","owner":"x","amount":1,"margin":999}`); r.Code != http.StatusOK {
		t.Fatalf("save: %d %s", r.Code, r.Body)
	}
	_, one := call(t, boss, http.MethodGet, base+"/"+id, "")
	margin := data(t, one)["margin"].(float64)
	if margin == 999 || margin == 0 {
		t.Fatalf("margin after hidden write = %v", margin)
	}
}

func TestFieldPermissionIsChecked(t *testing.T) {
	d := Definition{ID: "x", Label: "X", Fields: []Field{{ID: "a", Label: "A", Kind: "text", Required: true, Permission: "x.a"}}}
	if d.Validate() == nil {
		t.Fatal("required field with permission accepted")
	}
}
