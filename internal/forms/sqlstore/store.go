package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const MaxStoredForms = 1000

var ErrFormNotFound = sql.ErrNoRows

// Store is the durable registry for validated form schemas. Authorization,
// publication and submission policy remain owned by the Site module.
type Store struct {
	Executor           database.Executor
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func (s Store) validate() error {
	if err := database.RequireExecutor(s.Executor); err != nil {
		return err
	}
	if s.Dialect != database.SQLite && s.Dialect != database.Postgres && s.Dialect != database.MySQL {
		return fmt.Errorf("unsupported forms dialect %q", s.Dialect)
	}
	return nil
}

func (s Store) Upsert(ctx context.Context, form forms.Form) error {
	return s.UpsertWithExecutor(ctx, s.Executor, form)
}

func (s Store) UpsertWithExecutor(ctx context.Context, executor database.Executor, form forms.Form) error {
	if db, ok := executor.(*sql.DB); ok {
		return database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			copy := s
			copy.Executor = tx
			return copy.UpsertWithExecutor(ctx, tx, form)
		})
	}
	s.Executor = executor
	if err := s.validate(); err != nil {
		return err
	}
	if err := form.Validate(); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(form)
	if err != nil {
		return err
	}
	if s.Dialect == database.Postgres {
		if s.RequireTenantScope {
			_, err = s.Executor.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at,tenant_id) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET title=EXCLUDED.title,schema_json=EXCLUDED.schema_json,updated_at=EXCLUDED.updated_at,tenant_id=EXCLUDED.tenant_id`, form.ID, form.Title, string(payload), time.Now().UTC(), tenantID)
		} else {
			_, err = s.Executor.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at) VALUES ($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET title=EXCLUDED.title,schema_json=EXCLUDED.schema_json,updated_at=EXCLUDED.updated_at`, form.ID, form.Title, string(payload), time.Now().UTC())
		}
		return err
	}
	if s.Dialect == database.MySQL {
		now := time.Now().UTC()
		if s.RequireTenantScope {
			_, err = s.Executor.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at,tenant_id) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE title=VALUES(title),schema_json=VALUES(schema_json),updated_at=VALUES(updated_at),tenant_id=VALUES(tenant_id)`, form.ID, form.Title, string(payload), now, tenantID)
		} else {
			_, err = s.Executor.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE title=VALUES(title),schema_json=VALUES(schema_json),updated_at=VALUES(updated_at)`, form.ID, form.Title, string(payload), now)
		}
		return err
	}
	if s.RequireTenantScope {
		_, err = s.Executor.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at,tenant_id) VALUES (?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,schema_json=excluded.schema_json,updated_at=excluded.updated_at,tenant_id=excluded.tenant_id`, form.ID, form.Title, string(payload), time.Now().UTC().Format(time.RFC3339Nano), tenantID)
	} else {
		_, err = s.Executor.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at) VALUES (?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,schema_json=excluded.schema_json,updated_at=excluded.updated_at`, form.ID, form.Title, string(payload), time.Now().UTC().Format(time.RFC3339Nano))
	}
	return err
}

func (s Store) Get(ctx context.Context, id string) (forms.Form, error) {
	if db, ok := s.Executor.(*sql.DB); ok {
		var form forms.Form
		err := database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			copy := s
			copy.Executor = tx
			var err error
			form, err = copy.Get(ctx, id)
			return err
		})
		return form, err
	}
	if err := s.validate(); err != nil {
		return forms.Form{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return forms.Form{}, ErrFormNotFound
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return forms.Form{}, err
	}
	query := "SELECT schema_json FROM hydro_forms WHERE id = " + placeholder(s.Dialect, 1)
	args := []any{id}
	if s.RequireTenantScope {
		query += " AND tenant_id = " + placeholder(s.Dialect, 2)
		args = append(args, tenantID)
	}
	var payload string
	if err := s.Executor.QueryRowContext(ctx, query, args...).Scan(&payload); err != nil {
		return forms.Form{}, err
	}
	var form forms.Form
	if err := json.Unmarshal([]byte(payload), &form); err != nil {
		return forms.Form{}, errors.New("stored form schema is invalid")
	}
	if err := form.Validate(); err != nil {
		return forms.Form{}, errors.New("stored form schema is invalid")
	}
	return form, nil
}

func (s Store) List(ctx context.Context) ([]forms.Form, error) {
	if db, ok := s.Executor.(*sql.DB); ok {
		var result []forms.Form
		err := database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope).WithinQuery(ctx, func(tx database.QueryExecutor) error {
			copy := s
			copy.Executor = tx
			var err error
			result, err = copy.List(ctx)
			return err
		})
		return result, err
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	queryer, ok := s.Executor.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("forms executor does not support queries")
	}
	query := "SELECT schema_json FROM hydro_forms"
	args := []any{}
	if s.RequireTenantScope {
		query += " WHERE tenant_id = " + placeholder(s.Dialect, 1)
		args = append(args, tenantID)
	}
	limitPos := 1
	if s.RequireTenantScope {
		limitPos = 2
	}
	query += " ORDER BY id LIMIT " + placeholder(s.Dialect, limitPos)
	args = append(args, MaxStoredForms)
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]forms.Form, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var form forms.Form
		if err := json.Unmarshal([]byte(payload), &form); err != nil || form.Validate() != nil {
			return nil, errors.New("stored form schema is invalid")
		}
		result = append(result, form)
	}
	return result, rows.Err()
}

func (s Store) Delete(ctx context.Context, id string) error {
	return s.DeleteWithExecutor(ctx, s.Executor, id)
}

func (s Store) DeleteWithExecutor(ctx context.Context, executor database.Executor, id string) error {
	if db, ok := executor.(*sql.DB); ok {
		return database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
			copy := s
			copy.Executor = tx
			return copy.DeleteWithExecutor(ctx, tx, id)
		})
	}
	s.Executor = executor
	if err := s.validate(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrFormNotFound
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := "DELETE FROM hydro_forms WHERE id = " + placeholder(s.Dialect, 1)
	args := []any{id}
	if s.RequireTenantScope {
		query += " AND tenant_id = " + placeholder(s.Dialect, 2)
		args = append(args, tenantID)
	}
	result, err := s.Executor.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrFormNotFound
	}
	return nil
}

func placeholder(dialect database.Dialect, position int) string {
	if dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
