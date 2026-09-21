package adminstudio

import (
	"errors"
	"testing"
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
