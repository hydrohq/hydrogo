package adminstudio

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ImportMaxBytes = 5 << 20
	ImportMaxRows  = 5000
)

var ErrInvalidImportFile = errors.New("invalid import file")

// ParseImportFile reads bounded CSV or the first worksheet of a standard XLSX
// workbook. It has no runtime dependencies and never expands archive entries
// beyond the total input limit.
func ParseImportFile(r io.Reader, name string) ([][]string, error) {
	if r == nil {
		return nil, ErrInvalidImportFile
	}
	raw, err := io.ReadAll(io.LimitReader(r, ImportMaxBytes+1))
	if err != nil || len(raw) > ImportMaxBytes {
		return nil, ErrInvalidImportFile
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".csv":
		reader := csv.NewReader(bytes.NewReader(raw))
		reader.FieldsPerRecord = -1
		reader.ReuseRecord = false
		reader.LazyQuotes = false
		rows, err := reader.ReadAll()
		if err != nil {
			return nil, ErrInvalidImportFile
		}
		return validateImportRows(rows)
	case ".xlsx":
		return parseXLSX(raw)
	default:
		return nil, ErrInvalidImportFile
	}
}

func validateImportRows(rows [][]string) ([][]string, error) {
	if len(rows) == 0 || len(rows) > ImportMaxRows+1 || len(rows[0]) == 0 || len(rows[0]) > 100 {
		return nil, ErrInvalidImportFile
	}
	width := len(rows[0])
	for i, row := range rows {
		if len(row) != width {
			return nil, ErrInvalidImportFile
		}
		for _, value := range row {
			if len(value) > 16<<10 {
				return nil, ErrInvalidImportFile
			}
		}
		if i == 0 {
			continue
		}
	}
	return rows, nil
}

type xlsxSharedStrings struct {
	Items []struct {
		Text string `xml:"t"`
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}
type xlsxSheet struct {
	Rows []struct {
		Cells []struct {
			Ref    string `xml:"r,attr"`
			Type   string `xml:"t,attr"`
			Value  string `xml:"v"`
			Inline struct {
				Text string `xml:"t"`
			} `xml:"is"`
		} `xml:"c"`
	} `xml:"row"`
}

func parseXLSX(raw []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, ErrInvalidImportFile
	}
	files := map[string]*zip.File{}
	totalExpanded := uint64(0)
	for _, f := range zr.File {
		totalExpanded += f.UncompressedSize64
		if totalExpanded > ImportMaxBytes || f.UncompressedSize64 > ImportMaxBytes {
			return nil, ErrInvalidImportFile
		}
		files[f.Name] = f
	}
	read := func(name string, dst any) error {
		f := files[name]
		if f == nil {
			return ErrInvalidImportFile
		}
		rc, e := f.Open()
		if e != nil {
			return e
		}
		defer rc.Close()
		data, e := io.ReadAll(io.LimitReader(rc, ImportMaxBytes+1))
		if e != nil || len(data) > ImportMaxBytes {
			return ErrInvalidImportFile
		}
		return xml.Unmarshal(data, dst)
	}
	var shared xlsxSharedStrings
	if files["xl/sharedStrings.xml"] != nil {
		if err := read("xl/sharedStrings.xml", &shared); err != nil {
			return nil, ErrInvalidImportFile
		}
	}
	var sheet xlsxSheet
	if err := read("xl/worksheets/sheet1.xml", &sheet); err != nil {
		return nil, ErrInvalidImportFile
	}
	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		if len(rows) > ImportMaxRows {
			return nil, ErrInvalidImportFile
		}
		values := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			col, e := xlsxColumn(cell.Ref)
			if e != nil || col >= 100 {
				return nil, ErrInvalidImportFile
			}
			for len(values) <= col {
				values = append(values, "")
			}
			value := cell.Value
			switch cell.Type {
			case "s":
				index, e := strconv.Atoi(value)
				if e != nil || index < 0 || index >= len(shared.Items) {
					return nil, ErrInvalidImportFile
				}
				item := shared.Items[index]
				value = item.Text
				if len(item.Runs) > 0 {
					value = ""
					for _, run := range item.Runs {
						value += run.Text
					}
				}
			case "inlineStr":
				value = cell.Inline.Text
			case "", "str", "b":
			default:
				return nil, ErrInvalidImportFile
			}
			if len(value) > 16<<10 {
				return nil, ErrInvalidImportFile
			}
			values[col] = value
		}
		rows = append(rows, values)
	}
	if len(rows) == 0 {
		return nil, ErrInvalidImportFile
	}
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	if width == 0 {
		return nil, ErrInvalidImportFile
	}
	for i := range rows {
		for len(rows[i]) < width {
			rows[i] = append(rows[i], "")
		}
	}
	return validateImportRows(rows)
}

func xlsxColumn(ref string) (int, error) {
	n := 0
	seen := false
	for _, r := range strings.ToUpper(ref) {
		if r >= 'A' && r <= 'Z' {
			n = n*26 + int(r-'A'+1)
			seen = true
			continue
		}
		break
	}
	if !seen {
		return 0, fmt.Errorf("invalid cell")
	}
	return n - 1, nil
}
