package adminstudio

import (
	"errors"
	"testing"
	"time"
)

func TestChannelManagerBindsPrincipalPermissionAndReplay(t *testing.T) {
	m := NewChannelManager()
	p := ChannelPrincipal{SessionID: "s1", UserID: "u1", TenantID: "t1"}
	c, err := m.Issue(p, "reports", "home", "https://reports.example", []string{"reports.read"})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Bind(c.ID, c.Challenge, p); err != nil {
		t.Fatal(err)
	}
	if err := m.Authorize(c.ID, "r1", "reports.read", p); err != nil {
		t.Fatal(err)
	}
	if err := m.Authorize(c.ID, "r1", "reports.read", p); !errors.Is(err, ErrExtensionChannelDenied) {
		t.Fatalf("replay = %v", err)
	}
	if err := m.Authorize(c.ID, "r2", "reports.write", p); !errors.Is(err, ErrExtensionChannelDenied) {
		t.Fatalf("permission = %v", err)
	}
	m.Revoke(c.ID, p)
	if err := m.Authorize(c.ID, "r3", "reports.read", p); !errors.Is(err, ErrExtensionChannelDenied) {
		t.Fatalf("revoked = %v", err)
	}
}

func TestChannelManagerFailsClosedAfterExpiry(t *testing.T) {
	m := NewChannelManager()
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	m.ttl = time.Minute
	p := ChannelPrincipal{SessionID: "s1", UserID: "u1"}
	c, err := m.Issue(p, "reports", "home", "https://reports.example", []string{"reports.read"})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := m.Bind(c.ID, c.Challenge, p); !errors.Is(err, ErrExtensionChannelDenied) {
		t.Fatalf("expired channel bind = %v", err)
	}
	if err := m.Authorize(c.ID, "r1", "reports.read", p); !errors.Is(err, ErrExtensionChannelDenied) {
		t.Fatalf("expired channel authorize = %v", err)
	}
}

func TestChannelChallengeEqualRejectsDifferentLengthAndContent(t *testing.T) {
	if !channelChallengeEqual("0123456789", "0123456789") {
		t.Fatal("equal challenge rejected")
	}
	if channelChallengeEqual("0123456789", "012345678") || channelChallengeEqual("0123456789", "0123456780") {
		t.Fatal("different challenge accepted")
	}
}

