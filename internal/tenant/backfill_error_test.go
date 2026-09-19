package tenant

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsMissingTableErrorIsNarrow(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "sqlite missing table", err: errors.New("SQL logic error: no such table: optional_module"), want: true},
		{name: "postgres missing relation", err: &pgconn.PgError{Code: "42P01", Message: "relation optional_module does not exist"}, want: true},
		{name: "wrapped postgres missing relation", err: fmt.Errorf("query failed: %w", &pgconn.PgError{Code: "42P01"}), want: true},
		{name: "missing column is fatal", err: errors.New("column optional_module does not exist"), want: false},
		{name: "missing schema is fatal", err: errors.New("schema optional_module does not exist"), want: false},
		{name: "constraint is fatal", err: errors.New("constraint failed"), want: false},
		{name: "connection is fatal", err: errors.New("connection refused"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMissingTableError(tt.err); got != tt.want {
				t.Fatalf("isMissingTableError() = %v, want %v", got, tt.want)
			}
		})
	}
}
