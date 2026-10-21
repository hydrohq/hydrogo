package contenttypes

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// CSV transfer limits.
const (
	MaxExportRecords = 50000
	MaxImportRows    = 1000
	maxImportBytes   = 5 << 20
)

var errDryRun = errors.New("dry run")

// exportColumns are the fields written to CSV: everything the caller may
// read. Values of one record are joined with ";" for lists.
func (a AdminAPI) exportColumns(r *http.Request, definition Definition) []Field {
	var columns []Field
	for _, field := range definition.Fields {
		if a.fieldReadable(r, field) {
			columns = append(columns, field)
		}
	}
	return columns
}

// export streams the records matching the list filters as CSV: a header of
// field ids, then one row per record (columns id, then the fields). Row
// access and field permissions apply. Cells that a spreadsheet would run as
// a formula are prefixed with an apostrophe.
//
//	GET /api/v1/admin/content/deals/_export?filter[stage]=won
func (a AdminAPI) export(w http.ResponseWriter, r *http.Request) {
	definition, _, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	query, ok := a.scopedQuery(w, r, definition)
	if !ok {
		return
	}
	columns := a.exportColumns(r, definition)
	header := make([]string, 0, len(columns)+1)
	header = append(header, "id")
	for _, field := range columns {
		header = append(header, field.ID)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+definition.ID+`.csv"`)
	w.Header().Set("Cache-Control", "private, no-store")
	out := csv.NewWriter(w)
	_ = out.Write(header)
	err := a.eachRecord(r, definition.ID, query, MaxExportRecords, func(record collections.Record) error {
		item := a.present(r, definition.ID, Flatten(record))
		row := make([]string, 0, len(header))
		row = append(row, record.ID)
		for _, field := range columns {
			row = append(row, csvCell(item[field.ID]))
		}
		return out.Write(row)
	})
	out.Flush()
	if err != nil {
		// The status line is already out; end the file with a marker row
		// so a truncated export cannot pass for a complete one.
		w.Header().Set("X-Export-Incomplete", "1")
		_ = csv.NewWriter(w).Write([]string{"# export incomplete: " + exportFailure(err)})
	}
}

func exportFailure(err error) string {
	if errors.Is(err, errTooMany) {
		return fmt.Sprintf("more than %d records; narrow the filters", MaxExportRecords)
	}
	return "storage error"
}

// csvCell renders a value for CSV and defuses spreadsheet formulas.
func csvCell(value any) string {
	var text string
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		text = typed
	case bool:
		text = strconv.FormatBool(typed)
	case float64:
		text = strconv.FormatFloat(typed, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, part := range typed {
			parts = append(parts, csvCell(part))
		}
		text = strings.Join(parts, ";")
	default:
		text = fmt.Sprint(typed)
	}
	if text != "" && strings.ContainsRune("=+-@\t\r", rune(text[0])) {
		if _, err := strconv.ParseFloat(text, 64); err != nil {
			return "'" + text
		}
	}
	return text
}

// importCSV creates one record per CSV row, all in one transaction: the
// first failing row rolls everything back and is named as rows[i].<field>.
// The first row names the columns (field ids; "id" is not accepted). Add
// ?dry_run=1 to validate without saving. Needs write access to the type and,
// with row access, the rows become the caller's.
//
//	POST /api/v1/admin/content/deals/_import   (Content-Type: text/csv)
func (a AdminAPI) importCSV(w http.ResponseWriter, r *http.Request) {
	definition, _, ok := a.resolve(w, r, true)
	if !ok {
		return
	}
	rows, columns, err := a.readImport(r, w, definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}
	dryRun := r.URL.Query().Get("dry_run") == "1"
	r = a.withActor(r)
	var created []string
	err = a.Service.InTransaction(r.Context(), func(tx collections.Service) error {
		inTx := a
		inTx.Service = tx
		schema, err := tx.GetSchema(r.Context(), definition.ID)
		if err != nil {
			return err
		}
		for index, row := range rows {
			payload := make(map[string]any, len(columns))
			for position, field := range columns {
				if value, present, err := csvValue(field, row[position]); err != nil {
					return &batchError{index: index, err: err, schema: schema}
				} else if present {
					payload[field.ID] = value
				}
			}
			record, err := inTx.save(r, schema, "", payload)
			if err != nil {
				return &batchError{index: index, err: err, schema: schema}
			}
			id, _ := record["id"].(string)
			created = append(created, id)
		}
		if dryRun {
			return errDryRun
		}
		return nil
	})
	var failed *batchError
	switch {
	case err == nil || errors.Is(err, errDryRun):
		api.WriteData(w, http.StatusOK, map[string]any{"rows": len(rows), "created": len(created), "dry_run": dryRun}, nil)
	case errors.As(err, &failed):
		status, message, fields := serviceFailure(failed.err, failed.schema)
		prefixed := map[string]string{}
		for field, code := range fields {
			prefixed[fmt.Sprintf("rows[%d].%s", failed.index, field)] = code
		}
		if len(prefixed) == 0 {
			prefixed[fmt.Sprintf("rows[%d]", failed.index)] = message
		}
		api.WriteError(w, r, status, fmt.Sprintf("row %d failed: %s", failed.index+1, message), prefixed)
	case errors.Is(err, collections.ErrNoTransactions):
		api.WriteError(w, r, http.StatusNotImplemented, "import is not available with this storage", nil)
	default:
		writeServiceError(w, r, err, collections.Schema{})
	}
}

// readImport parses the body into rows aligned with the returned columns.
func (a AdminAPI) readImport(r *http.Request, w http.ResponseWriter, definition Definition) ([][]string, []Field, error) {
	reader := csv.NewReader(http.MaxBytesReader(w, r.Body, maxImportBytes))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, nil, errors.New("the CSV needs a header row of field ids")
	}
	columns := make([]Field, 0, len(header))
	seen := map[string]bool{}
	for _, name := range header {
		name = strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))
		field, found := fieldByID(definition, name)
		switch {
		case !found || field.Computed != "" || field.Sequence != nil || (definition.States != nil && definition.States.Field == field.ID) || !a.fieldReadable(r, field):
			return nil, nil, fmt.Errorf("column %q is not a field that can be imported", name)
		case seen[name]:
			return nil, nil, fmt.Errorf("column %q appears twice", name)
		}
		seen[name] = true
		columns = append(columns, field)
	}
	var rows [][]string
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, errors.New("the CSV could not be read (too large or malformed)")
		}
		if len(row) != len(columns) {
			return nil, nil, fmt.Errorf("row %d has %d cells, the header has %d", len(rows)+1, len(row), len(columns))
		}
		if rows = append(rows, row); len(rows) > MaxImportRows {
			return nil, nil, fmt.Errorf("at most %d rows per import", MaxImportRows)
		}
	}
	if len(rows) == 0 {
		return nil, nil, errors.New("the CSV has no rows")
	}
	return rows, columns, nil
}

// csvValue converts a cell to the field's value; empty cells are left out.
func csvValue(field Field, cell string) (any, bool, error) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return nil, false, nil
	}
	invalid := func() (any, bool, error) {
		return nil, false, &saveError{err: fmt.Errorf("%w: %s", collections.ErrInvalidRecord, field.ID), data: map[string]any{field.ID: cell}}
	}
	switch field.Kind {
	case "number":
		number, err := strconv.ParseFloat(cell, 64)
		if err != nil {
			return invalid()
		}
		return number, true, nil
	case "boolean":
		switch strings.ToLower(cell) {
		case "true", "1", "yes":
			return true, true, nil
		case "false", "0", "no":
			return false, true, nil
		}
		return invalid()
	case "relation":
		if field.Many {
			var ids []any
			for _, id := range strings.Split(cell, ";") {
				if id = strings.TrimSpace(id); id != "" {
					ids = append(ids, id)
				}
			}
			return ids, true, nil
		}
	}
	return cell, true, nil
}
