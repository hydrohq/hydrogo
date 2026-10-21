package contenttypes

import (
	"net/http"
	"testing"
)

const invoicesYAML = `id: invoices
label: Invoices
title_field: number
fields:
  - {id: number, label: Number, kind: text, list: true, sequence: {prefix: "INV-{yyyy}-", pad: 4}}
  - {id: quantity, label: Quantity, kind: number, min: 0}
  - {id: unit_price, label: Unit price, kind: number, decimals: 2, min: 0}
  - {id: subtotal, label: Subtotal, kind: number, decimals: 2, computed: "quantity * unit_price", filterable: true}
  - {id: total, label: Total, kind: number, decimals: 2, computed: "round(subtotal * 1.18, 2)", list: true}
`

func TestInvoiceTypeNumbersAndComputesOnSave(t *testing.T) {
	service, types := newServiceFor(t, invoicesYAML)
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	base := AdminAPIPath + "/invoices"

	created, body := call(t, admin, http.MethodPost, base, `{"quantity":3,"unit_price":19.99,"total":1,"number":"HACK"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	record := data(t, body)
	if record["subtotal"] != 59.97 || record["total"] != 70.76 {
		t.Fatalf("computed fields = %v", record)
	}
	if number, _ := record["number"].(string); len(number) != len("INV-2026-0001") || number[len(number)-4:] != "0001" {
		t.Fatalf("number = %v", record["number"])
	}
	id := record["id"].(string)

	updated, body := call(t, admin, http.MethodPut, base+"/"+id, `{"version":1,"quantity":10,"unit_price":2.5,"number":"HACK2"}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update: %d %s", updated.Code, updated.Body)
	}
	after := data(t, body)
	if after["subtotal"] != 25.0 || after["total"] != 29.5 || after["number"] != record["number"] {
		t.Fatalf("after update = %v", after)
	}

	if response, _ := call(t, admin, http.MethodPost, base, `{"quantity":1,"unit_price":1.005}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("3 decimals on a money field: %d %s", response.Code, response.Body)
	}

	_, filtered := call(t, admin, http.MethodGet, base+"?filter[subtotal][gte]=25", "")
	if rows, _ := filtered["data"].([]any); len(rows) != 1 {
		t.Fatalf("filter on a computed field = %v", filtered)
	}
}
