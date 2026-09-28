package adminstudio

import (
	"strings"
	"testing"
)

func TestParseImportCSVAndRejectsBadShape(t *testing.T) {
	rows, err := ParseImportFile(strings.NewReader("title,slug\nA,a\n"), "content.CSV")
	if err != nil || len(rows) != 2 || rows[1][0] != "A" {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if _, err := ParseImportFile(strings.NewReader("title,slug\nA\n"), "content.csv"); err == nil {
		t.Fatal("accepted inconsistent row")
	}
	if _, err := ParseImportFile(strings.NewReader("a,b"), "content.xls"); err == nil {
		t.Fatal("accepted unsupported format")
	}
}

func TestXLSXColumn(t *testing.T) {
	for input, want := range map[string]int{"A1": 0, "Z2": 25, "AA1": 26, "AZ4": 51} {
		got, err := xlsxColumn(input)
		if err != nil || got != want {
			t.Fatalf("%s=%d,%v want %d", input, got, err, want)
		}
	}
}
