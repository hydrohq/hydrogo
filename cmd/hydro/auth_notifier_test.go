package main

import "testing"

func TestAuthNotifierBuildsAdminPanelLinks(t *testing.T) {
	n := newAuthNotifier(nil, "https://crm.example.test/")
	if n.ResetLink == nil || n.InviteLink == nil {
		t.Fatal("a public origin must enable links")
	}
	if got := n.ResetLink("t"); got != "https://crm.example.test/admin/reset-password?token=t" {
		t.Errorf("reset link %q", got)
	}
	if got := n.InviteLink("t"); got != "https://crm.example.test/admin/accept-invite?token=t" {
		t.Errorf("invite link %q", got)
	}
	if bare := newAuthNotifier(nil, "  "); bare.ResetLink != nil || bare.InviteLink != nil {
		t.Error("without an origin there is nothing to link to")
	}
}
