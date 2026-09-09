package observability

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBoundedPathCapsLogField(t *testing.T) {
	path := make([]byte, maxLoggedPathBytes+20)
	for i := range path {
		path[i] = 'a'
	}
	bounded := boundedPath(string(path))
	if len(bounded) != maxLoggedPathBytes+len("…") {
		t.Fatalf("bounded path length = %d, want %d", len(bounded), maxLoggedPathBytes+len("…"))
	}
	if bounded[len(bounded)-len("…"):] != "…" {
		t.Fatalf("bounded path missing truncation marker: %q", bounded)
	}
}

func TestBoundedPathLeavesShortPathUnchanged(t *testing.T) {
	const path = "/admin/pages/1"
	if got := boundedPath(path); got != path {
		t.Fatalf("bounded path = %q, want %q", got, path)
	}
}

func TestBoundedPathDoesNotSplitUTF8(t *testing.T) {
	path := strings.Repeat("a", maxLoggedPathBytes-1) + "ğ" + "bbbb"
	bounded := boundedPath(path)
	if !utf8.ValidString(bounded) {
		t.Fatalf("bounded path is invalid UTF-8: %q", bounded)
	}
}
