package realtime

import (
	"errors"
	"testing"
	"time"
)

func TestPresenceTenantIsolationAndTTL(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	presence := &Presence{TTL: time.Minute, Now: func() time.Time { return now }}
	if err := presence.Heartbeat(PresenceSession{TenantID: "tenant-a", Resource: "page-1", UserID: "u2", Name: "Editor B"}); err != nil {
		t.Fatal(err)
	}
	if got := presence.Active("tenant-a", "page-1", "u1"); len(got) != 1 || got[0].Name != "Editor B" {
		t.Fatalf("active editors = %#v", got)
	}
	if got := presence.Active("tenant-b", "page-1", "u1"); len(got) != 0 {
		t.Fatalf("cross-tenant editors leaked: %#v", got)
	}
	if got := presence.Active("tenant-a", "page-1", "u2"); len(got) != 0 {
		t.Fatalf("current user was not excluded: %#v", got)
	}
	now = now.Add(time.Minute)
	if got := presence.Active("tenant-a", "page-1", "u1"); len(got) != 0 {
		t.Fatalf("expired editor remained: %#v", got)
	}
}

func TestPresenceRejectsInvalidAndSeparatesResources(t *testing.T) {
	p := &Presence{}
	if err := p.Heartbeat(PresenceSession{TenantID: "t", Resource: "page\n2", UserID: "u", Name: "Editor"}); !errors.Is(err, ErrInvalidPresence) {
		t.Fatalf("invalid resource error = %v", err)
	}
	if err := p.Heartbeat(PresenceSession{TenantID: "t", Resource: "page-1", UserID: "u", Name: "Editor"}); err != nil {
		t.Fatal(err)
	}
	if got := p.Active("t", "news-1", "other"); len(got) != 0 {
		t.Fatalf("presence crossed resources: %#v", got)
	}
}

func TestPresenceLeaveAndSessionFields(t *testing.T) {
	presence := &Presence{TTL: time.Minute}
	if err := presence.Heartbeat(PresenceSession{TenantID: "t1", Resource: "res-1", UserID: "u1", Email: "u1@example.com", Name: "User One"}); err != nil {
		t.Fatal(err)
	}
	active := presence.Active("t1", "res-1", "other")
	if len(active) != 1 || active[0].Email != "u1@example.com" || active[0].UserID != "u1" {
		t.Fatalf("unexpected active editor: %#v", active)
	}

	presence.Leave("t1", "res-1", "u1")
	activeAfter := presence.Active("t1", "res-1", "other")
	if len(activeAfter) != 0 {
		t.Fatalf("expected editor to be gone after Leave, got: %#v", activeAfter)
	}
}
