package sqlstore

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type BanStore struct{ Dialect database.Dialect }

func (s BanStore) Put(ctx context.Context, db database.Executor, prefix netip.Prefix, expires time.Time) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	var expiry any
	if !expires.IsZero() {
		expiry = expires.UTC().Format(time.RFC3339Nano)
	}
	query := `INSERT INTO security_ip_bans(address,expires_at,created_at) VALUES($1,$2,$3) ON CONFLICT(address) DO UPDATE SET expires_at=EXCLUDED.expires_at,created_at=EXCLUDED.created_at`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO security_ip_bans(address,expires_at,created_at) VALUES(?,?,?) ON CONFLICT(address) DO UPDATE SET expires_at=excluded.expires_at,created_at=excluded.created_at`
	}
	_, err := db.ExecContext(ctx, query, prefix.Masked().String(), expiry, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s BanStore) Remove(ctx context.Context, db database.Executor, prefix netip.Prefix) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	query := `DELETE FROM security_ip_bans WHERE address=$1`
	if s.Dialect == database.SQLite {
		query = `DELETE FROM security_ip_bans WHERE address=?`
	}
	_, err := db.ExecContext(ctx, query, prefix.Masked().String())
	return err
}

func (s BanStore) ListActive(ctx context.Context, db database.QueryExecutor, now time.Time, limit int) ([]security.IPBan, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 10000 {
		return nil, fmt.Errorf("ban list limit must be 1-10000")
	}
	query := `SELECT address,expires_at FROM security_ip_bans WHERE expires_at IS NULL OR expires_at>$1 ORDER BY address LIMIT $2`
	if s.Dialect == database.SQLite {
		query = `SELECT address,expires_at FROM security_ip_bans WHERE expires_at IS NULL OR expires_at='' OR expires_at>? ORDER BY address LIMIT ?`
	}
	rows, err := db.QueryContext(ctx, query, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]security.IPBan, 0)
	for rows.Next() {
		var address string
		var expires string
		if err := rows.Scan(&address, &expires); err != nil {
			return nil, err
		}
		prefix, err := netip.ParsePrefix(address)
		if err != nil {
			return nil, fmt.Errorf("invalid stored ban prefix: %w", err)
		}
		entry := security.IPBan{Prefix: prefix.Masked()}
		if expires != "" {
			entry.Expires, err = time.Parse(time.RFC3339Nano, expires)
			if err != nil {
				return nil, fmt.Errorf("invalid stored ban expiry: %w", err)
			}
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}
