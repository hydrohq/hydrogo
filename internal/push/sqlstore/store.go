package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type Store struct {
	DB                 database.Executor
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) p(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
func (s Store) scope(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	sc, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return sc.TenantID, nil
}
func (s Store) within(ctx context.Context, fn func(Store) error) error {
	if db, ok := s.DB.(*sql.DB); ok {
		return database.NewTenantTransactor(db, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error { next := s; next.DB = tx; return fn(next) })
	}
	return fn(s)
}
func (s Store) Save(ctx context.Context, sub push.Subscription) error {
	return s.within(ctx, func(tx Store) error {
		scope, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		keys := string(sub.Keys)
		if len(sub.Keys) == 0 {
			keys = "{}"
		}
		_, err = tx.DB.ExecContext(ctx, "DELETE FROM push_subscriptions WHERE tenant_id="+tx.p(1)+" AND user_id="+tx.p(2)+" AND platform="+tx.p(3)+" AND endpoint="+tx.p(4), scope, sub.UserID, sub.Platform, sub.Endpoint)
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, err = tx.DB.ExecContext(ctx, "INSERT INTO push_subscriptions (tenant_id,id,user_id,platform,endpoint,keys_json,active,created_at,updated_at) VALUES ("+tx.p(1)+","+tx.p(2)+","+tx.p(3)+","+tx.p(4)+","+tx.p(5)+","+tx.p(6)+",1,"+tx.p(7)+","+tx.p(8)+")", scope, sub.ID, sub.UserID, sub.Platform, sub.Endpoint, keys, now, now)
		return err
	})
}
func (s Store) Delete(ctx context.Context, id, userID string, platform push.Platform, endpoint string) error {
	return s.within(ctx, func(tx Store) error {
		scope, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		_, err = tx.DB.ExecContext(ctx, "DELETE FROM push_subscriptions WHERE tenant_id="+tx.p(1)+" AND id="+tx.p(2)+" AND user_id="+tx.p(3)+" AND platform="+tx.p(4)+" AND endpoint="+tx.p(5), scope, id, userID, platform, endpoint)
		return err
	})
}
func (s Store) Disable(ctx context.Context, id string) error {
	return s.within(ctx, func(tx Store) error {
		scope, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		_, err = tx.DB.ExecContext(ctx, "UPDATE push_subscriptions SET active=0,updated_at="+tx.p(1)+" WHERE tenant_id="+tx.p(2)+" AND id="+tx.p(3), time.Now().UTC().Format(time.RFC3339Nano), scope, id)
		return err
	})
}
func (s Store) List(ctx context.Context, userID string, platform push.Platform) ([]push.Subscription, error) {
	items := make([]push.Subscription, 0)
	err := s.within(ctx, func(tx Store) error {
		queryer, ok := tx.DB.(database.QueryExecutor)
		if !ok {
			return errors.New("push subscriptions are unavailable")
		}
		scope, err := tx.scope(ctx)
		if err != nil {
			return err
		}
		query := "SELECT id,user_id,platform,endpoint,keys_json,created_at FROM push_subscriptions WHERE tenant_id=" + tx.p(1) + " AND active=1"
		args := []any{scope}
		if userID != "" {
			args = append(args, userID)
			query += " AND user_id=" + tx.p(len(args))
		}
		args = append(args, platform)
		query += " AND platform=" + tx.p(len(args)) + " ORDER BY created_at ASC LIMIT 10000"
		rows, err := queryer.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item push.Subscription
			var p, keys, created string
			if err := rows.Scan(&item.ID, &item.UserID, &p, &item.Endpoint, &keys, &created); err != nil {
				return err
			}
			item.Platform = push.Platform(p)
			if json.Valid([]byte(keys)) {
				item.Keys = json.RawMessage(keys)
			}
			item.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
			if err != nil {
				return errors.New("invalid subscription timestamp")
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}
