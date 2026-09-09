package auth

import (
	"testing"
)

func TestContextHelpersAreNilSafe(t *testing.T) {
	user := &User{ID: "user-1"}
	session := &Session{ID: "session-1"}
	if got := WithUser(nil, user); UserFromContext(got) != user {
		t.Fatal("WithUser did not retain the user")
	}
	if got := WithSession(nil, session); SessionFromContext(got) != session {
		t.Fatal("WithSession did not retain the session")
	}
	if UserFromContext(nil) != nil || SessionFromContext(nil) != nil {
		t.Fatal("nil context unexpectedly returned authentication state")
	}
}
