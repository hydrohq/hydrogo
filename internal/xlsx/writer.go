// Package xlsx writes bounded UTF-8 spreadsheets without external runtimes.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrDimensions = errors.New("spreadsheet exceeds Excel worksheet limits")

// Write emits a single-sheet XLSX workbook. The caller owns row-count limits.
func Write(dst io.Writer, rows [][]string) error {
	if len(rows) > 1_048_576 {
		return ErrDimensions
	}
	for _, row := range rows {
		if len(row) > 16_384 {
			return ErrDimensions
		}
	}
	w := zip.NewWriter(dst)
	files := []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="HydroGo" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`},
	}
	for _, file := range files {
		if err := writeZipFile(w, file.name, strings.NewReader(file.body)); err != nil {
			_ = w.Close()
			return err
		}
	}
	if err := writeSheet(w, rows); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func writeZipFile(w *zip.Writer, name string, src io.Reader) error {
	file, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, src)
	return err
}

func writeSheet(w *zip.Writer, rows [][]string) error {
	file, err := w.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	enc := xml.NewEncoder(file)
	if _, err := io.WriteString(file, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`); err != nil {
		return err
	}
	start := xml.StartElement{Name: xml.Name{Local: "worksheet"}, Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}}}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	sheetData := xml.StartElement{Name: xml.Name{Local: "sheetData"}}
	if err := enc.EncodeToken(sheetData); err != nil {
		return err
	}
	for rowIndex, row := range rows {
		r := xml.StartElement{Name: xml.Name{Local: "row"}, Attr: []xml.Attr{{Name: xml.Name{Local: "r"}, Value: fmt.Sprint(rowIndex + 1)}}}
		if err := enc.EncodeToken(r); err != nil {
			return err
		}
		for colIndex, value := range row {
			cell := xml.StartElement{Name: xml.Name{Local: "c"}, Attr: []xml.Attr{{Name: xml.Name{Local: "r"}, Value: columnName(colIndex+1) + fmt.Sprint(rowIndex+1)}, {Name: xml.Name{Local: "t"}, Value: "inlineStr"}}}
			if rowIndex == 0 {
				cell.Attr = append(cell.Attr, xml.Attr{Name: xml.Name{Local: "s"}, Value: "1"})
			}
			if err := enc.EncodeToken(cell); err != nil {
				return err
			}
			inline := xml.StartElement{Name: xml.Name{Local: "is"}}
			if err := enc.EncodeToken(inline); err != nil {
				return err
			}
			if err := enc.EncodeElement(value, xml.StartElement{Name: xml.Name{Local: "t"}, Attr: []xml.Attr{{Name: xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "space"}, Value: "preserve"}}}); err != nil {
				return err
			}
			if err := enc.EncodeToken(xml.EndElement{Name: inline.Name}); err != nil {
				return err
			}
			if err := enc.EncodeToken(xml.EndElement{Name: cell.Name}); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(xml.EndElement{Name: r.Name}); err != nil {
			return err
		}
	}
	if err := enc.EncodeToken(xml.EndElement{Name: sheetData.Name}); err != nil {
		return err
	}
	if err := enc.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return err
	}
	return enc.Flush()
}

func columnName(n int) string {
	var out [3]byte
	index := len(out)
	for n > 0 {
		n--
		index--
		out[index] = byte('A' + n%26)
		n /= 26
	}
	return string(out[index:])
}
