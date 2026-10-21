package collections

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Cursor marks a position in a list sorted by a system column (updated_at,
// the default, or created_at), so the next page starts after it without an
// offset. Value is the column's stored text; records with the same value are
// ordered by ID, as in the list's ORDER BY.
type Cursor struct {
	Field string `json:"f"`
	Desc  bool   `json:"d,omitempty"`
	Value string `json:"v"`
	ID    string `json:"i"`
}

const maxCursorBytes = 512

// EncodeCursor returns the opaque string clients pass back as ?after=.
func EncodeCursor(cursor Cursor) string {
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

// DecodeCursor parses a string produced by EncodeCursor.
func DecodeCursor(raw string) (Cursor, error) {
	if len(raw) == 0 || len(raw) > maxCursorBytes {
		return Cursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidQuery)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	var cursor Cursor
	if err != nil || json.Unmarshal(decoded, &cursor) != nil {
		return Cursor{}, fmt.Errorf("%w: invalid cursor", ErrInvalidQuery)
	}
	return cursor, nil
}

// effectiveSort is the system column and direction a query without a field
// sort lists by: updated_at newest first unless the sort says otherwise.
func effectiveSort(sort *Sort) (field string, desc, ok bool) {
	if sort == nil {
		return SortUpdatedAt, true, true
	}
	if sort.Kind == "" && (sort.Field == SortCreatedAt || sort.Field == SortUpdatedAt) {
		return sort.Field, sort.Desc, true
	}
	return "", false, false
}

// NextCursor returns the cursor that continues a list after record, or ""
// when the query's sort cannot be continued by cursor (a field sort).
func NextCursor(query Query, record Record) string {
	field, desc, ok := effectiveSort(query.Sort)
	if !ok {
		return ""
	}
	value := record.UpdatedAt
	if field == SortCreatedAt {
		value = record.CreatedAt
	}
	return EncodeCursor(Cursor{Field: field, Desc: desc, Value: value.UTC().Format(time.RFC3339Nano), ID: record.ID})
}

// validateCursor checks that the cursor belongs to this query's sort.
func validateCursor(cursor *Cursor, q Query) error {
	field, desc, ok := effectiveSort(q.Sort)
	if !ok || q.Offset != 0 {
		return fmt.Errorf("%w: after cannot be used with this sort or an offset", ErrInvalidQuery)
	}
	if cursor.Field != field || cursor.Desc != desc || len(cursor.Value) == 0 || len(cursor.Value) > 64 || !identifier.MatchString(cursor.ID) {
		return fmt.Errorf("%w: cursor does not match this list", ErrInvalidQuery)
	}
	return nil
}
