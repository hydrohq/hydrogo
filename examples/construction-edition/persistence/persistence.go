package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Dialect string

const (
	SQLite     Dialect = "sqlite"
	PostgreSQL Dialect = "postgres"
)

var (
	ErrNotFound = errors.New("construction record not found")
	ErrConflict = errors.New("construction revision conflict")
	ErrDenied   = errors.New("construction approval denied")
)

type Store struct{ Dialect Dialect }

func (s Store) p(n int) string {
	if s.Dialect == PostgreSQL {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s Store) Schema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS construction_organizations (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL DEFAULT '', name TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS construction_projects (id TEXT PRIMARY KEY, organization_id TEXT NOT NULL, name TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS construction_memberships (project_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL, PRIMARY KEY(project_id,user_id))`,
		`CREATE TABLE IF NOT EXISTS construction_documents (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, object_key TEXT NOT NULL, checksum TEXT NOT NULL, status TEXT NOT NULL, revision INTEGER NOT NULL, created_by TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS construction_daily_reports (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, report_day TEXT NOT NULL, author_id TEXT NOT NULL, summary TEXT NOT NULL, status TEXT NOT NULL, revision INTEGER NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS construction_audit_events (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, actor_id TEXT NOT NULL, action TEXT NOT NULL, target_id TEXT NOT NULL, created_at TEXT NOT NULL)`,
	}
}

func (s Store) ApplySchema(ctx context.Context, db *sql.DB) error {
	for _, stmt := range s.Schema() {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) AddMembership(ctx context.Context, db *sql.DB, projectID, userID, role string) error {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(userID) == "" {
		return ErrDenied
	}
	_, err := db.ExecContext(ctx, "INSERT INTO construction_memberships(project_id,user_id,role) VALUES ("+s.p(1)+","+s.p(2)+","+s.p(3)+") ON CONFLICT(project_id,user_id) DO UPDATE SET role=excluded.role", projectID, userID, role)
	return err
}

func (s Store) CreateDocument(ctx context.Context, db *sql.DB, id, projectID, objectKey, checksum, actor string) error {
	if id == "" || projectID == "" || objectKey == "" || checksum == "" || actor == "" {
		return ErrDenied
	}
	_, err := db.ExecContext(ctx, "INSERT INTO construction_documents(id,project_id,object_key,checksum,status,revision,created_by,updated_at) VALUES ("+s.p(1)+","+s.p(2)+","+s.p(3)+","+s.p(4)+",'pending',1,"+s.p(5)+","+s.p(6)+")", id, projectID, objectKey, checksum, actor, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// ApproveDocument performs authorization and revision compare-and-swap in one transaction.
func (s Store) ApproveDocument(ctx context.Context, db *sql.DB, projectID, documentID, actor string, expectedRevision int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var role, status, storedProject string
	var revision int
	q := "SELECT m.role,d.status,d.project_id,d.revision FROM construction_memberships m JOIN construction_documents d ON d.project_id=m.project_id WHERE m.project_id=" + s.p(1) + " AND m.user_id=" + s.p(2) + " AND d.id=" + s.p(3)
	if err = tx.QueryRowContext(ctx, q, projectID, actor, documentID).Scan(&role, &status, &storedProject, &revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if storedProject != projectID || (role != "owner" && role != "manager") {
		return ErrDenied
	}
	if status != "pending" || revision != expectedRevision {
		return ErrConflict
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, "UPDATE construction_documents SET status='approved',revision=revision+1,updated_at="+s.p(1)+" WHERE id="+s.p(2)+" AND project_id="+s.p(3)+" AND status='pending' AND revision="+s.p(4), now, documentID, projectID, expectedRevision)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO construction_audit_events(id,project_id,actor_id,action,target_id,created_at) VALUES ("+s.p(1)+","+s.p(2)+","+s.p(3)+",'document.approved',"+s.p(4)+","+s.p(5)+")", fmt.Sprintf("audit-%d", time.Now().UnixNano()), projectID, actor, documentID, now)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}
