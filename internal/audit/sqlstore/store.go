package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) Record(ctx context.Context, tx database.Executor, id, action, targetID string) error {
	return s.RecordChange(ctx, tx, id, action, targetID, []byte("{}"), []byte("{}"))
}

// RecordChange writes bounded, non-sensitive before/after JSON alongside an immutable event.
func (s Store) RecordChange(ctx context.Context, tx database.Executor, id, action, targetID string, before, after []byte) error {
	if ChainEnabled() {
		// The event and its chain link must land together and the head row
		// stay locked until then: give a bare connection its own transaction.
		if db, ok := tx.(*sql.DB); ok && db != nil {
			dbTx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			if err := s.recordChange(ctx, dbTx, id, action, targetID, before, after); err != nil {
				_ = dbTx.Rollback()
				return err
			}
			return dbTx.Commit()
		}
	}
	return s.recordChange(ctx, tx, id, action, targetID, before, after)
}

func (s Store) recordChange(ctx context.Context, tx database.Executor, id, action, targetID string, before, after []byte) error {
	if len(before) == 0 {
		before = []byte("{}")
	}
	if len(after) == 0 {
		after = []byte("{}")
	}
	if len(before) > 4096 || len(after) > 4096 || !json.Valid(before) || !json.Valid(after) {
		return errors.New("audit change details must be valid JSON under 4 KiB")
	}
	if _, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		return s.recordScoped(ctx, tx, id, action, targetID, before, after)
	} else if s.RequireTenantScope {
		return scopeErr
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	var requestID any
	if value := security.RequestIDFromContext(ctx); value != "" {
		requestID = value
	}
	event := chainedEvent{ID: id, Action: action, TargetID: targetID, ActorID: actorID(ctx, action, targetID), ResourceType: resourceType(action), CreatedAt: nowUTC().Format("2006-01-02T15:04:05.000000000Z"), Before: string(before), After: string(after)}
	if requestID != nil {
		event.RequestID = requestID.(string)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events (id, action, target_id, request_id, actor_id, resource_type, created_at, before_json, after_json) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+", "+s.placeholder(6)+", "+s.placeholder(7)+", "+s.placeholder(8)+", "+s.placeholder(9)+")", id, action, targetID, requestID, event.ActorID, event.ResourceType, event.CreatedAt, event.Before, event.After); err != nil {
		return err
	}
	return s.appendChain(ctx, tx, "", event)
}

// RecordScoped writes an audit event with an explicit tenant discriminator.
func (s Store) RecordScoped(ctx context.Context, tx database.Executor, id, action, targetID string) error {
	return s.recordScoped(ctx, tx, id, action, targetID, []byte("{}"), []byte("{}"))
}

func (s Store) recordScoped(ctx context.Context, tx database.Executor, id, action, targetID string, before, after []byte) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	var requestID any
	if value := security.RequestIDFromContext(ctx); value != "" {
		requestID = value
	}
	event := chainedEvent{ID: id, Action: action, TargetID: targetID, ActorID: actorID(ctx, action, targetID), ResourceType: resourceType(action), CreatedAt: nowUTC().Format("2006-01-02T15:04:05.000000000Z"), Before: string(before), After: string(after)}
	if requestID != nil {
		event.RequestID = requestID.(string)
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events (id, tenant_id, action, target_id, request_id, actor_id, resource_type, created_at, before_json, after_json) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+", "+s.placeholder(5)+", "+s.placeholder(6)+", "+s.placeholder(7)+", "+s.placeholder(8)+", "+s.placeholder(9)+", "+s.placeholder(10)+")", id, scope.TenantID, action, targetID, requestID, event.ActorID, event.ResourceType, event.CreatedAt, event.Before, event.After); err != nil {
		return err
	}
	return s.appendChain(ctx, tx, scope.TenantID, event)
}

func actorID(ctx context.Context, action, targetID string) string {
	if user := auth.UserFromContext(ctx); user != nil && user.Active {
		return user.ID
	}
	if strings.HasPrefix(action, "auth.") {
		return targetID
	}
	return ""
}

func resourceType(action string) string {
	resource, _, found := strings.Cut(action, ".")
	if !found {
		return "system"
	}
	return resource
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

// nowUTC is the clock for audit timestamps. It is a variable so a test can
// freeze it: on Windows consecutive events can share a timestamp.
var nowUTC = func() time.Time { return time.Now().UTC() }
