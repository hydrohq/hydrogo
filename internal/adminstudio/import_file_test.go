package adminstudio

import (
	"archive/zip"
	"bytes"
	"io"
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

func TestParseImportXLSXSharedAndInlineStrings(t *testing.T) {
	workbook := xlsxFixture(t, map[string]string{
		"xl/sharedStrings.xml":     `<sst><si><t>Başlık</t></si><si><t>urun-1</t></si><si><t>Ürün metni</t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="inlineStr"><is><t>slug</t></is></c><c r="C1" t="inlineStr"><is><t>body</t></is></c></row><row r="2"><c r="A2" t="s"><v>0</v></c><c r="B2" t="s"><v>1</v></c><c r="C2" t="s"><v>2</v></c></row></sheetData></worksheet>`,
	})

	rows, err := ParseImportFile(bytes.NewReader(workbook), "pages.XLSX")
	if err != nil {
		t.Fatalf("ParseImportFile() error = %v", err)
	}
	want := [][]string{{"Başlık", "slug", "body"}, {"Başlık", "urun-1", "Ürün metni"}}
	if len(rows) != len(want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
	for i := range want {
		if strings.Join(rows[i], "\x00") != strings.Join(want[i], "\x00") {
			t.Fatalf("row %d = %#v, want %#v", i, rows[i], want[i])
		}
	}
}

func xlsxFixture(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, body := range files {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatalf("archive.Create(%q): %v", name, err)
		}
		if _, err := io.WriteString(file, body); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("archive.Close(): %v", err)
	}
	return buffer.Bytes()
}
