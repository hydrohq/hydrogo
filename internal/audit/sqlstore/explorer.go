package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type ExplorerFilter struct {
	Actor, Action, Resource, Target, From, Until string
	Limit, Offset                                int
}

type ExplorerEvent struct {
	ID, ActorID, ActorEmail, Action, TargetID, ResourceType string
	CreatedAt, RequestID, BeforeJSON, AfterJSON             string
}

type ExplorerResult struct {
	Events []ExplorerEvent
	Total  int
}

// Explore returns a tenant-scoped, bounded audit page and total count.
func (s Store) Explore(ctx context.Context, filter ExplorerFilter) (ExplorerResult, error) {
	if s.RequireTenantScope {
		if _, err := tenant.Require(ctx); err != nil {
			return ExplorerResult{}, err
		}
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 || filter.Offset > 1_000_000 {
		return ExplorerResult{}, errors.New("audit page is out of range")
	}
	for _, value := range []string{filter.Actor, filter.Action, filter.Resource, filter.Target, filter.From, filter.Until} {
		if len(value) > 256 || strings.ContainsRune(value, '\x00') {
			return ExplorerResult{}, errors.New("audit filter is invalid")
		}
	}
	where, args, next := s.explorerWhere(ctx, filter)
	var result ExplorerResult
	countSQL := "SELECT COUNT(*) FROM audit_events a" + where
	if err := s.DB.QueryRowContext(ctx, countSQL, args...).Scan(&result.Total); err != nil {
		return ExplorerResult{}, err
	}
	limitArg := next
	offsetArg := next + 1
	query := "SELECT a.id, a.actor_id, COALESCE(u.email, ''), a.action, a.target_id, a.resource_type, a.created_at, COALESCE(a.request_id, ''), a.before_json, a.after_json FROM audit_events a LEFT JOIN users u ON u.id = a.actor_id" + where + " ORDER BY a.created_at DESC, a.id DESC LIMIT " + s.placeholder(limitArg) + " OFFSET " + s.placeholder(offsetArg) // #nosec G202 -- only placeholders and fixed identifiers are concatenated; every value is bound
	queryArgs := append(append([]any(nil), args...), filter.Limit, filter.Offset)
	rows, err := s.DB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return ExplorerResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var event ExplorerEvent
		if err := rows.Scan(&event.ID, &event.ActorID, &event.ActorEmail, &event.Action, &event.TargetID, &event.ResourceType, &event.CreatedAt, &event.RequestID, &event.BeforeJSON, &event.AfterJSON); err != nil {
			return ExplorerResult{}, err
		}
		result.Events = append(result.Events, event)
	}
	return result, rows.Err()
}

func (s Store) explorerWhere(ctx context.Context, filter ExplorerFilter) (string, []any, int) {
	conditions := make([]string, 0, 7)
	args := make([]any, 0, 7)
	appendArg := func(expr string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(expr, s.placeholder(len(args))))
	}
	if s.RequireTenantScope {
		scope, _ := tenant.Require(ctx)
		appendArg("a.tenant_id = %s", scope.TenantID)
	}
	if filter.Actor != "" {
		args = append(args, filter.Actor, filter.Actor)
		conditions = append(conditions, fmt.Sprintf("(a.actor_id = %s OR a.actor_id IN (SELECT id FROM users WHERE lower(email) = lower(%s)))", s.placeholder(len(args)-1), s.placeholder(len(args))))
	}
	if filter.Action != "" {
		appendArg("a.action LIKE %s", "%"+filter.Action+"%")
	}
	if filter.Resource != "" {
		appendArg("a.resource_type = %s", filter.Resource)
	}
	if filter.Target != "" {
		appendArg("a.target_id LIKE %s", "%"+filter.Target+"%")
	}
	if filter.From != "" {
		appendArg("a.created_at >= %s", filter.From)
	}
	if filter.Until != "" {
		appendArg("a.created_at < %s", filter.Until)
	}
	if len(conditions) == 0 {
		return "", args, len(args) + 1
	}
	return " WHERE " + strings.Join(conditions, " AND "), args, len(args) + 1
}
