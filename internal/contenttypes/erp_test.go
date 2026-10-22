package contenttypes

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestERPLiteFlow walks the erp-lite stock types through an order, its
// invoice and a payment: numbering, exact money, state steps and locking.
func TestERPLiteFlow(t *testing.T) {
	files, err := filepath.Glob("../../content-types/erp/*.yaml")
	if err != nil || len(files) != 9 {
		t.Fatalf("erp types: %v %v", files, err)
	}
	var sources []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, string(content))
	}
	service, types := newServiceFor(t, sources...)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: func(*http.Request, string) bool { return true }})
	root := AdminAPIPath

	_, body := call(t, admin, http.MethodPost, root+"/accounts", `{"name":"Acme","kind":"customer"}`)
	account := data(t, body)["id"].(string)
	_, body = call(t, admin, http.MethodPost, root+"/products", `{"sku":"P-1","name":"Widget","unit":"piece","price":19.99}`)
	product := data(t, body)["id"].(string)

	// The order and its lines are created together, atomically.
	batch := `{"operations":[
	 {"op":"create","type":"sales_orders","fields":{"account":"` + account + `","discount":0.97}},
	 {"op":"create","type":"order_lines","fields":{"order":"$ref:0","product":"` + product + `","quantity":3,"unit_price":19.99,"vat_rate":18}}]}`
	if r, _ := call(t, admin, http.MethodPost, BatchPath, batch); r.Code != http.StatusOK {
		t.Fatalf("batch: %d %s", r.Code, r.Body)
	}
	_, list := call(t, admin, http.MethodGet, root+"/sales_orders", "")
	order := list["data"].([]any)[0].(map[string]any)
	if order["number"] == "" || order["subtotal"].(float64) != 59.97 || order["tax"].(float64) != 10.79 || order["total"].(float64) != 69.79 || order["status"] != "draft" {
		t.Fatalf("order = %v", order)
	}
	orderID := order["id"].(string)
	_, lines := call(t, admin, http.MethodGet, root+"/order_lines", "")
	if lines["data"].([]any)[0].(map[string]any)["line_total"].(float64) != 59.97 {
		t.Fatalf("line = %v", lines["data"])
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/order_lines", `{"order":"`+orderID+`","product":"`+product+`","quantity":0,"unit_price":1}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("zero quantity: %d", r.Code)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/sales_orders/"+orderID+"/state", `{"to":"confirmed"}`); r.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", r.Code, r.Body)
	}

	// Invoice: numbered, posted once, then locked.
	_, body = call(t, admin, http.MethodPost, root+"/invoices", `{"account":"`+account+`","order":"`+orderID+`"}`)
	invoiceID := data(t, body)["id"].(string)
	if r, _ := call(t, admin, http.MethodPost, root+"/invoice_lines", `{"invoice":"`+invoiceID+`","description":"Widget","quantity":3,"unit_price":19.99,"vat_rate":18}`); r.Code != http.StatusCreated {
		t.Fatalf("invoice line: %d %s", r.Code, r.Body)
	}
	_, body = call(t, admin, http.MethodGet, root+"/invoices/"+invoiceID, "")
	if invoice := data(t, body); invoice["number"] == "" || invoice["subtotal"].(float64) != 59.97 || invoice["total"].(float64) != 70.76 {
		t.Fatalf("invoice = %v", invoice)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/invoices/"+invoiceID+"/state", `{"to":"posted"}`); r.Code != http.StatusOK {
		t.Fatalf("post: %d %s", r.Code, r.Body)
	}
	if r, b := call(t, admin, http.MethodPut, root+"/invoices/"+invoiceID, `{"account":"`+account+`","subtotal":1,"tax":0}`); r.Code != http.StatusUnprocessableEntity || b["message"] != "Posted invoices are locked" {
		t.Fatalf("edit posted invoice: %d %s", r.Code, r.Body)
	}
	// Lines of a posted invoice are locked with it, so its total cannot move.
	if r, _ := call(t, admin, http.MethodPost, root+"/invoice_lines", `{"invoice":"`+invoiceID+`","description":"Late","quantity":1,"unit_price":5}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("line on posted invoice: %d %s", r.Code, r.Body)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/payments", `{"invoice":"`+invoiceID+`","amount":70.76,"paid_on":"2026-10-21","method":"bank_transfer"}`); r.Code != http.StatusCreated {
		t.Fatalf("payment: %d %s", r.Code, r.Body)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/invoices/"+invoiceID+"/state", `{"to":"paid"}`); r.Code != http.StatusOK {
		t.Fatalf("paid: %d %s", r.Code, r.Body)
	}
}

const rollupParentYAML = `id: carts
label: Carts
fields:
  - {id: name, label: Name, kind: text}
  - {id: items, label: Items, kind: number, decimals: 0, rollup: {from: cart_items, via: cart}}
  - {id: total, label: Total, kind: number, decimals: 2, rollup: {from: cart_items, via: cart, sum: price}}
`

const rollupChildYAML = `id: cart_items
label: Cart items
fields:
  - {id: cart, label: Cart, kind: relation, to: carts, required: true, filterable: true}
  - {id: price, label: Price, kind: number, decimals: 2}
`

func TestRollupsFollowEveryChangeToTheChildren(t *testing.T) {
	service, types := newServiceFor(t, rollupParentYAML, rollupChildYAML)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	root := AdminAPIPath
	cart := func(id string) map[string]any {
		_, body := call(t, admin, http.MethodGet, root+"/carts/"+id, "")
		return data(t, body)
	}
	_, body := call(t, admin, http.MethodPost, root+"/carts", `{"name":"a","total":999}`)
	a := data(t, body)["id"].(string)
	_, body = call(t, admin, http.MethodPost, root+"/carts", `{"name":"b"}`)
	b := data(t, body)["id"].(string)
	if got := cart(a); got["total"].(float64) != 0 || got["items"].(float64) != 0 {
		t.Fatalf("new cart = %v", got)
	}
	_, body = call(t, admin, http.MethodPost, root+"/cart_items", `{"cart":"`+a+`","price":0.1}`)
	first := data(t, body)["id"].(string)
	call(t, admin, http.MethodPost, root+"/cart_items", `{"cart":"`+a+`","price":0.2}`)
	if got := cart(a); got["total"].(float64) != 0.3 || got["items"].(float64) != 2 {
		t.Fatalf("after two items = %v", got)
	}
	// Moving an item to another cart updates both.
	call(t, admin, http.MethodPut, root+"/cart_items/"+first, `{"cart":"`+b+`","price":0.1}`)
	if got := cart(a); got["total"].(float64) != 0.2 || got["items"].(float64) != 1 {
		t.Fatalf("source cart = %v", got)
	}
	if got := cart(b); got["total"].(float64) != 0.1 || got["items"].(float64) != 1 {
		t.Fatalf("target cart = %v", got)
	}
	// Editing the parent keeps its totals; deleting an item lowers them.
	call(t, admin, http.MethodPut, root+"/carts/"+b, `{"name":"renamed","total":5}`)
	if got := cart(b); got["name"] != "renamed" || got["total"].(float64) != 0.1 {
		t.Fatalf("renamed cart = %v", got)
	}
	call(t, admin, http.MethodDelete, root+"/cart_items/"+first, "")
	if got := cart(b); got["total"].(float64) != 0 || got["items"].(float64) != 0 {
		t.Fatalf("after delete = %v", got)
	}
}

func TestRollupDefinitionsAreChecked(t *testing.T) {
	for name, child := range map[string]string{
		"missing type":   "",
		"wrong via":      "id: cart_items\nlabel: Items\nfields:\n  - {id: cart, label: Cart, kind: text}\n  - {id: price, label: Price, kind: number}\n",
		"sum not number": "id: cart_items\nlabel: Items\nfields:\n  - {id: cart, label: Cart, kind: relation, to: carts}\n  - {id: price, label: Price, kind: text}\n",
	} {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(rollupParentYAML), 0o644)
		if child != "" {
			_ = os.WriteFile(filepath.Join(dir, "b.yaml"), []byte(child), 0o644)
		}
		types, err := LoadDir(dir)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := Schemas(types); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestPeriodClosingLocksDatedRecords(t *testing.T) {
	files, _ := filepath.Glob("../../content-types/erp/*.yaml")
	var sources []string
	for _, file := range files {
		content, _ := os.ReadFile(file)
		sources = append(sources, string(content))
	}
	service, types := newServiceFor(t, sources...)
	allow := func(*http.Request, string) bool { return true }
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: allow})
	root := AdminAPIPath
	_, body := call(t, admin, http.MethodPost, root+"/accounts", `{"name":"Acme","kind":"customer"}`)
	account := data(t, body)["id"].(string)
	invoice := func(date string) *httptestRecorder {
		r, b := call(t, admin, http.MethodPost, root+"/invoices", `{"account":"`+account+`","issue_date":"`+date+`"}`)
		return &httptestRecorder{code: r.Code, body: b}
	}
	_, body = call(t, admin, http.MethodPost, root+"/invoices", `{"account":"`+account+`","issue_date":"2026-09-10"}`)
	september := data(t, body)["id"].(string)

	// Without the permission nobody can close a period; with it, it is final.
	denied := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Can: func(*http.Request, string) bool { return false }})
	if r, _ := call(t, denied, http.MethodPost, root+"/period_closings", `{"closed_through":"2026-09-30"}`); r.Code != http.StatusForbidden {
		t.Fatalf("close without permission: %d", r.Code)
	}
	_, body = call(t, admin, http.MethodPost, root+"/period_closings", `{"closed_through":"2026-09-30"}`)
	closing := data(t, body)["id"].(string)
	if r, _ := call(t, admin, http.MethodDelete, root+"/period_closings/"+closing, ""); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reopen a period: %d", r.Code)
	}

	if got := invoice("2026-09-30"); got.code != http.StatusUnprocessableEntity || got.body["message"] != "The period up to 2026-09-30 is closed" {
		t.Fatalf("create in closed period: %d %v", got.code, got.body)
	}
	if got := invoice("2026-10-01"); got.code != http.StatusCreated {
		t.Fatalf("create after the period: %d %v", got.code, got.body)
	}
	if r, _ := call(t, admin, http.MethodPut, root+"/invoices/"+september, `{"account":"`+account+`","issue_date":"2026-10-05"}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("move a closed record out: %d", r.Code)
	}
	if r, _ := call(t, admin, http.MethodDelete, root+"/invoices/"+september, ""); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("delete in closed period: %d", r.Code)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/invoices/"+september+"/state", `{"to":"posted"}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("step in closed period: %d", r.Code)
	}
}

type httptestRecorder struct {
	code int
	body map[string]any
}

func TestPeriodLockAndHooksCoverPublishing(t *testing.T) {
	const posts = `id: posts
label: Posts
drafts: true
period_lock: {field: day, closings: closes, through: upto}
hooks:
  - {on: update, service: gate}
fields:
  - {id: day, label: Day, kind: date}
`
	const closes = `id: closes
label: Closes
fields:
  - {id: upto, label: Upto, kind: date}
`
	service, types := newServiceFor(t, posts, closes)
	veto := false
	hook := func(context.Context, string, HookEvent) (int, string, error) {
		if veto {
			return http.StatusUnprocessableEntity, "vetoed", nil
		}
		return http.StatusOK, "", nil
	}
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Hook: hook})
	root := AdminAPIPath
	_, body := call(t, admin, http.MethodPost, root+"/posts", `{"day":"2026-09-01"}`)
	id := data(t, body)["id"].(string)
	veto = true
	if r, _ := call(t, admin, http.MethodPost, root+"/posts/"+id+"/workflow", `{"target":"published","version":1}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("publish past a vetoing hook: %d %s", r.Code, r.Body)
	}
	veto = false
	call(t, admin, http.MethodPost, root+"/closes", `{"upto":"2026-09-30"}`)
	if r, _ := call(t, admin, http.MethodPost, root+"/posts/"+id+"/workflow", `{"target":"published","version":1}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("publish in a closed period: %d %s", r.Code, r.Body)
	}
}
