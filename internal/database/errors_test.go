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

func TestIsForeignKeyViolationRecognizesEachDatabase(t *testing.T) {
	for name, err := range map[string]error{
		"PostgreSQL SQLSTATE": &pgconn.PgError{Code: "23503"},
		"PostgreSQL RESTRICT": &pgconn.PgError{Code: "23001"},
		"PostgreSQL message":  errors.New(`ERROR: update or delete on table "t" violates foreign key constraint "fk" on table "u"`),
		"MySQL parent row":    errors.New("Error 1451 (23000): Cannot delete or update a parent row: a foreign key constraint fails (`db`.`t`, CONSTRAINT `fk`)"),
		"MySQL child row":     errors.New("Error 1452 (23000): Cannot add or update a child row: a foreign key constraint fails"),
		"SQLite":              errors.New("constraint failed: FOREIGN KEY constraint failed (787)"),
	} {
		if !IsForeignKeyViolation(err) {
			t.Errorf("%s was not recognized as a foreign key violation", name)
		}
		if IsUniqueViolation(err) {
			t.Errorf("%s was taken for a unique violation", name)
		}
	}
	if IsForeignKeyViolation(errors.New("UNIQUE constraint failed: members.email")) || IsForeignKeyViolation(nil) {
		t.Fatal("a unique failure or nil is not a foreign key violation")
	}
}
