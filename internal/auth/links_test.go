package auth

import "testing"

func TestLinksPointAtTheAdminPanel(t *testing.T) {
	cases := map[string]string{
		PasswordResetLink("https://crm.example.test/", "a b+c"): "https://crm.example.test/admin/reset-password?token=a+b%2Bc",
		InviteAcceptLink("http://127.0.0.1:8080", "tok"):        "http://127.0.0.1:8080/admin/accept-invite?token=tok",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
