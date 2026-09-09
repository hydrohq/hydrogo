package security

import "testing"

func TestContextHelpersAcceptNilContexts(t *testing.T) {
	if got := RequestIDFromContext(nil); got != "" {
		t.Fatalf("nil request ID context = %q", got)
	}
	if got := RequestIDFromContext(WithRequestID(nil, "request-1")); got != "request-1" {
		t.Fatalf("request ID from nil context = %q", got)
	}
	if got := CSRFTokenFromContext(nil); got != "" {
		t.Fatalf("nil CSRF context = %q", got)
	}
}
