package contenttypes

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestERPLiteFlow walks the erp-lite stock types through an order, its
// invoice and a payment: numbering, exact money, state steps and locking.
func TestERPLiteFlow(t *testing.T) {
	files, err := filepath.Glob("../../content-types/erp/*.yaml")
	if err != nil || len(files) != 8 {
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
	 {"op":"create","type":"sales_orders","fields":{"account":"` + account + `","subtotal":59.97,"discount":0.97,"tax":10.8}},
	 {"op":"create","type":"order_lines","fields":{"order":"$ref:0","product":"` + product + `","quantity":3,"unit_price":19.99}}]}`
	if r, _ := call(t, admin, http.MethodPost, BatchPath, batch); r.Code != http.StatusOK {
		t.Fatalf("batch: %d %s", r.Code, r.Body)
	}
	_, list := call(t, admin, http.MethodGet, root+"/sales_orders", "")
	order := list["data"].([]any)[0].(map[string]any)
	if order["number"] == "" || order["total"].(float64) != 69.8 || order["status"] != "draft" {
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
	_, body = call(t, admin, http.MethodPost, root+"/invoices", `{"account":"`+account+`","order":"`+orderID+`","subtotal":59.97,"tax":10.8}`)
	invoice := data(t, body)
	invoiceID := invoice["id"].(string)
	if invoice["number"] == "" || invoice["total"].(float64) != 70.77 {
		t.Fatalf("invoice = %v", invoice)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/invoices/"+invoiceID+"/state", `{"to":"posted"}`); r.Code != http.StatusOK {
		t.Fatalf("post: %d %s", r.Code, r.Body)
	}
	if r, b := call(t, admin, http.MethodPut, root+"/invoices/"+invoiceID, `{"account":"`+account+`","subtotal":1,"tax":0}`); r.Code != http.StatusUnprocessableEntity || b["message"] != "Posted invoices are locked" {
		t.Fatalf("edit posted invoice: %d %s", r.Code, r.Body)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/payments", `{"invoice":"`+invoiceID+`","amount":70.77,"paid_on":"2026-10-21","method":"bank_transfer"}`); r.Code != http.StatusCreated {
		t.Fatalf("payment: %d %s", r.Code, r.Body)
	}
	if r, _ := call(t, admin, http.MethodPost, root+"/invoices/"+invoiceID+"/state", `{"to":"paid"}`); r.Code != http.StatusOK {
		t.Fatalf("paid: %d %s", r.Code, r.Body)
	}
}
