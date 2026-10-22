package database

import (
	"errors"
	"strings"
)

// IsUniqueViolation identifies a uniqueness failure at the SQL adapter
// boundary without exposing a driver or SQL dialect to application code.
func IsUniqueViolation(err error) bool {
	if err == nil || IsForeignKeyViolation(err) {
		return false
	}
	var sqlState interface{ SQLState() string }
	if errors.As(err, &sqlState) && sqlState.SQLState() == "23505" {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "unique failed") ||
		strings.Contains(message, "duplicate entry") ||
		strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "constraint failed")
}

// IsForeignKeyViolation identifies a foreign key failure (deleting a row other
// rows point to, or pointing at a row that is not there) at the SQL adapter
// boundary: PostgreSQL SQLSTATE 23503, MySQL errors 1451 and 1452, and SQLite's
// "FOREIGN KEY constraint failed". PostgreSQL reports a RESTRICT rule (what the
// collections tables use) as 23001 and a plain reference as 23503.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	var sqlState interface{ SQLState() string }
	if errors.As(err, &sqlState) && (sqlState.SQLState() == "23503" || sqlState.SQLState() == "23001") {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "foreign key constraint failed") ||
		strings.Contains(message, "violates restrict setting of foreign key") ||
		strings.Contains(message, "foreign key constraint fails") ||
		strings.Contains(message, "violates foreign key constraint")
}
