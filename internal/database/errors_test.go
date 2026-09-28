package database

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type sqlStateTestError string

func (e sqlStateTestError) Error() string    { return string(e) }
func (e sqlStateTestError) SQLState() string { return string(e) }

func TestIsUniqueViolationRecognizesPostgresSQLState(t *testing.T) {
	if !IsUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("expected SQLSTATE 23505 to be recognized")
	}
}

func TestIsUniqueViolationUsesAdapterNeutralSQLState(t *testing.T) {
	if !IsUniqueViolation(sqlStateTestError("23505")) {
		t.Fatal("expected adapter-neutral SQLSTATE to be recognized")
	}
}

func TestIsUniqueViolationRecognizesSQLiteMessage(t *testing.T) {
	if !IsUniqueViolation(errors.New("constraint failed: UNIQUE constraint failed: members.email")) {
		t.Fatal("expected SQLite uniqueness message to be recognized")
	}
}

func TestIsUniqueViolationRecognizesMySQLDuplicateEntry(t *testing.T) {
	if !IsUniqueViolation(errors.New("Error 1062 (23000): Duplicate entry 'member@example.test' for key 'members.email_unique'")) {
		t.Fatal("MySQL duplicate entry was not classified as a unique violation")
	}
}

func TestIsUniqueViolationRejectsUnrelatedErrors(t *testing.T) {
	if IsUniqueViolation(errors.New("connection reset by peer")) {
		t.Fatal("unrelated error was classified as unique violation")
	}
}
