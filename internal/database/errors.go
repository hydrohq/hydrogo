package database

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation identifies a uniqueness failure at the SQL adapter
// boundary without exposing a driver or SQL dialect to application code.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "unique failed") ||
		strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "constraint failed")
}
