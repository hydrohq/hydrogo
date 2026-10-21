package contenttypes

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

const invoiceYAML = `id: invoices
label: Invoices
title_field: status
states:
  field: status
  initial: draft
  transitions:
    - {from: draft, to: posted}
hooks:
  - {on: transition, to: posted, service: ledger}
  - {on: create, service: credit-check, fail_open: true}
  - {on: delete, service: ledger}
fields:
  - {id: status, label: Status, kind: select, options: [draft, posted]}
  - {id: total, label: Total, kind: number}
`

func TestHooksGateChanges(t *testing.T) {
	service, types := newServiceFor(t, invoiceYAML)
	base := AdminAPIPath + "/invoices"
	var seen []HookEvent
	answer := func(service string) (int, error) { return http.StatusOK, nil }
	hook := func(_ context.Context, service string, event HookEvent) (int, string, error) {
		seen = append(seen, event)
		status, err := answer(service)
		if status == http.StatusUnprocessableEntity {
			return status, "credit limit exceeded", nil
		}
		return status, "", err
	}
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll, Hook: hook})

	created, body := call(t, admin, http.MethodPost, base, `{"total":5}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	id := data(t, body)["id"].(string)
	if len(seen) != 1 || seen[0].On != "create" || seen[0].Fields["total"] != 5.0 || seen[0].Type != "invoices" {
		t.Fatalf("create event = %+v", seen)
	}
	// A fail_open hook that is down does not block creating.
	answer = func(string) (int, error) { return 0, errors.New("down") }
	if r, _ := call(t, admin, http.MethodPost, base, `{"total":6}`); r.Code != http.StatusCreated {
		t.Fatalf("fail-open create: %d %s", r.Code, r.Body)
	}
	// A required hook that is down blocks the step (fail closed).
	if r, _ := call(t, admin, http.MethodPost, base+"/"+id+"/state", `{"to":"posted"}`); r.Code != http.StatusServiceUnavailable {
		t.Fatalf("down hook: %d %s", r.Code, r.Body)
	}
	// A 4xx answer rejects with the service's message.
	answer = func(string) (int, error) { return http.StatusUnprocessableEntity, nil }
	if r, b := call(t, admin, http.MethodPost, base+"/"+id+"/state", `{"to":"posted"}`); r.Code != http.StatusUnprocessableEntity || b["message"] != "credit limit exceeded" {
		t.Fatalf("rejected step: %d %s", r.Code, r.Body)
	}
	if r, _ := call(t, admin, http.MethodDelete, base+"/"+id, ""); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("rejected delete: %d", r.Code)
	}
	if r, _ := call(t, admin, http.MethodPost, BatchPath, `{"operations":[{"op":"delete","type":"invoices","id":"`+id+`"}]}`); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("rejected batch delete: %d %s", r.Code, r.Body)
	}
	// Allowed: the step goes through and the event names from and to.
	answer = func(string) (int, error) { return http.StatusOK, nil }
	seen = nil
	if r, _ := call(t, admin, http.MethodPost, base+"/"+id+"/state", `{"to":"posted"}`); r.Code != http.StatusOK {
		t.Fatalf("allowed step: %d %s", r.Code, r.Body)
	}
	if len(seen) != 1 || seen[0].From != "draft" || seen[0].To != "posted" || seen[0].ID != id {
		t.Fatalf("transition event = %+v", seen)
	}
	// No caller configured: the required hook fails closed.
	bare := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	if r, _ := call(t, bare, http.MethodDelete, base+"/"+id, ""); r.Code != http.StatusServiceUnavailable {
		t.Fatalf("no caller: %d", r.Code)
	}
}

func TestHookDefinitionsAreChecked(t *testing.T) {
	fields := []Field{{ID: "a", Label: "A", Kind: "text"}}
	for name, hook := range map[string]Hook{
		"bad event":        {On: "explode", Service: "x"},
		"transition":       {On: "transition", Service: "x"},
		"to without steps": {On: "create", To: "a", Service: "x"},
		"bad service":      {On: "create", Service: "Bad Name"},
	} {
		d := Definition{ID: "x", Label: "X", Hooks: []Hook{hook}, Fields: fields}
		if d.Validate() == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}
