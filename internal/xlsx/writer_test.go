package xlsx

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestWriteProducesUTF8WorkbookWithStyledHeader(t *testing.T) {
	var out bytes.Buffer
	if err := Write(&out, [][]string{{"Türkçe Başlık", "備考"}, {"İstanbul, Bursa", "emoji 🚀 & <xml>"}}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read %s: %v / %v", file.Name, readErr, closeErr)
		}
		files[file.Name] = string(body)
	}
	for _, name := range []string{"[Content_Types].xml", "xl/workbook.xml", "xl/styles.xml", "xl/worksheets/sheet1.xml"} {
		if files[name] == "" {
			t.Fatalf("missing workbook part %s", name)
		}
	}
	for _, want := range []string{"Türkçe Başlık", "備考", "İstanbul, Bursa", "emoji 🚀 &amp; &lt;xml&gt;", `s="1"`, `<is><t xml:space="preserve">`} {
		if !strings.Contains(files["xl/worksheets/sheet1.xml"], want) {
			t.Fatalf("worksheet missing %q: %s", want, files["xl/worksheets/sheet1.xml"])
		}
	}
}

func TestWriteRejectsExcelLimitOverflow(t *testing.T) {
	rows := make([][]string, 1_048_577)
	if err := Write(io.Discard, rows); !errors.Is(err, ErrDimensions) {
		t.Fatalf("Write overflow = %v", err)
	}
}
