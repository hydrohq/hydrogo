package webhooks

import "testing"

func FuzzValidateWebhookURL(f *testing.F) {
	for _, seed := range []string{
		"https://example.test/webhook",
		"http://127.0.0.1:8080/events",
		"file:///tmp/event",
		"https://user:pass@example.test/hook",
		"https://[::1]:8443/hook",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_ = validateURL(raw, false)
	})
}

func FuzzWebhookSignature(f *testing.F) {
	f.Add([]byte(`{"id":"event-1"}`), "secret")
	f.Add([]byte{}, "")
	f.Fuzz(func(t *testing.T, body []byte, secret string) {
		got := signature(body, secret)
		if len(got) != len("sha256=")+64 || got[:len("sha256=")] != "sha256=" {
			t.Fatalf("invalid signature shape %q", got)
		}
	})
}
