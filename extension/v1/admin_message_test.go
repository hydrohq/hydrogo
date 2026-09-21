package v1

import (
	"errors"
	"strings"
	"testing"
)

func TestAdminMessageContractAcceptsBoundedTypedEnvelope(t *testing.T) {
	message, err := ParseAdminMessage([]byte(`{"version":1,"type":"ready","request_id":"req-1","payload":{"surface":"sales"}}`))
	if err != nil || message.Type != "ready" {
		t.Fatalf("message = %#v, err = %v", message, err)
	}
	payload := AdminBootstrapPayload{ExtensionID: "acme-reports", SurfaceID: "sales", Locale: "tr-TR", Permissions: []string{"reports.read"}}
	if err := payload.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (AdminChannelInitPayload{ExtensionID: "acme-reports", SurfaceID: "sales", Challenge: "challenge-123456789"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminMessageContractRejectsUnknownTypesAndOversizedPayload(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"type":"set-cookie"}`,
		`{"version":2,"type":"ready"}`,
		`{"version":1,"type":"ready","request_id":"bad/id"}`,
	} {
		if _, err := ParseAdminMessage([]byte(raw)); !errors.Is(err, ErrInvalidAdminMessage) {
			t.Fatalf("raw %s error = %v", raw, err)
		}
	}
	if _, err := ParseAdminMessage([]byte(`{"version":1,"type":"ready","payload":"` + strings.Repeat("x", MaxAdminMessageSize) + `"}`)); !errors.Is(err, ErrInvalidAdminMessage) {
		t.Fatalf("oversized payload error = %v", err)
	}
	if err := (AdminBootstrapPayload{ExtensionID: "x", SurfaceID: "y", Locale: "en\nUS"}).Validate(); !errors.Is(err, ErrInvalidAdminMessage) {
		t.Fatalf("invalid bootstrap error = %v", err)
	}
	if err := (AdminChannelInitPayload{ExtensionID: "x", SurfaceID: "y", Challenge: "short"}).Validate(); !errors.Is(err, ErrInvalidAdminMessage) {
		t.Fatalf("short challenge error = %v", err)
	}
}
