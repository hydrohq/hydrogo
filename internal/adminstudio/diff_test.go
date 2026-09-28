package adminstudio

import (
	"errors"
	"testing"
)

func TestCompareLinesAndFields(t *testing.T) {
	rows, err := CompareLines([]string{"title", "old", "tail"}, []string{"title", "new", "tail"})
	if err != nil || len(rows) != 4 || rows[1].Kind != DiffRemoved || rows[2].Kind != DiffAdded {
		t.Fatalf("diff = %#v, %v", rows, err)
	}
	fields := CompareFields(map[string]string{"title": "A", "body": "old"}, map[string]string{"title": "B", "body": "old", "status": "draft"})
	if len(fields) != 2 || fields[0].Field != "status" || fields[1].Field != "title" {
		t.Fatalf("field diff = %#v", fields)
	}
}

func TestCompareLinesIsBounded(t *testing.T) {
	_, err := CompareLines(make([]string, 401), nil)
	if !errors.Is(err, ErrDiffTooLarge) {
		t.Fatalf("oversized diff error = %v", err)
	}
}
