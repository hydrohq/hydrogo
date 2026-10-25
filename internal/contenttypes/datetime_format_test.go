package contenttypes

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

func TestDatetimeReadsBackAsItWasSubmitted(t *testing.T) {
	schema := collections.Schema{Fields: []collections.Field{{ID: "starts_at", Kind: collections.DateTime}}}
	for stored, wantPresented := range map[string]string{
		"2026-11-02T10:00:00Z":      "2026-11-02T10:00",
		"2026-11-02T10:00:30Z":      "2026-11-02T10:00:30",
		"2026-11-02T13:00:00+03:00": "2026-11-02T10:00",
	} {
		presented := presentDateTime(stored)
		if presented != wantPresented {
			t.Errorf("present(%s) = %q, want %q", stored, presented, wantPresented)
		}
		// Writing the presented value back must not change the instant.
		back := toStorage(schema, map[string]any{"starts_at": presented})["starts_at"]
		again := presentDateTime(back.(string))
		if again != presented {
			t.Errorf("round trip of %s: %q -> %v -> %q", stored, presented, back, again)
		}
	}
	if got := presentDateTime("not a date"); got != "not a date" {
		t.Errorf("unparsable values are left alone, got %q", got)
	}
}
