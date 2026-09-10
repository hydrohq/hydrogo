package transport

import "testing"

func TestNewIDIsNonEmpty(t *testing.T) {
	id, err := newID()
	if err != nil || id == "" {
		t.Fatalf("id = %q/%v", id, err)
	}
}
