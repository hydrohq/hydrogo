package sqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// The audit chain makes tampering with the log detectable. Each event gets a
// link: its position per tenant and SHA-256(previous hash, position, the
// event's fields). Changing, removing or reordering an event breaks every
// later hash. Anyone who can write to the database can rebuild the whole
// chain, so keep the head hash (hydro audit head) somewhere they cannot reach
// and check it with hydro audit verify --anchor.
var chainEnabled atomic.Bool

// SetChain turns chaining of new audit events on or off (config audit.chain).
func SetChain(enabled bool) { chainEnabled.Store(enabled) }

// ChainEnabled reports whether new audit events are chained.
func ChainEnabled() bool { return chainEnabled.Load() }

type chainedEvent struct {
	ID, Action, TargetID, RequestID, ActorID, ResourceType string
	CreatedAt, Before, After                               string
}

// chainHash is the link hash of an event after prevHash at position seq.
func chainHash(prevHash string, seq int64, tenantID string, e chainedEvent) string {
	body, _ := json.Marshal([]any{prevHash, seq, tenantID, e.ID, e.Action, e.TargetID, e.RequestID, e.ActorID, e.ResourceType, e.CreatedAt, e.Before, e.After})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// appendChain links the event just written. It runs in the event's
// transaction and locks the tenant's head row, so concurrent writers queue
// instead of forking the chain.
func (s Store) appendChain(ctx context.Context, tx database.Executor, tenantID string, e chainedEvent) error {
	if !ChainEnabled() {
		return nil
	}
	p := s.placeholder
	lock := ""
	if s.Dialect != database.SQLite {
		lock = " FOR UPDATE"
	}
	// PostgreSQL and SQLite look at the head first and insert it only for the
	// first event of a tenant. MySQL always runs the insert, as ON DUPLICATE
	// KEY UPDATE: it takes the row's exclusive lock at once, so racing writers
	// queue, whereas INSERT IGNORE or a plain read first leaves them holding
	// shared or gap locks that they then deadlock upgrading.
	read := func() (int64, string, error) {
		var seq int64
		var last string
		err := tx.QueryRowContext(ctx, "SELECT seq, last_hash FROM audit_chain_heads WHERE tenant_id = "+p(1)+lock, tenantID).Scan(&seq, &last)
		return seq, last, err
	}
	seq, last, err := int64(0), "", error(sql.ErrNoRows)
	if s.Dialect != database.MySQL {
		seq, last, err = read()
	}
	if errors.Is(err, sql.ErrNoRows) {
		var ensure string
		switch s.Dialect {
		case database.Postgres:
			ensure = "INSERT INTO audit_chain_heads (tenant_id, seq, last_hash) VALUES (" + p(1) + ", 0, '') ON CONFLICT (tenant_id) DO NOTHING"
		case database.MySQL:
			ensure = "INSERT INTO audit_chain_heads (tenant_id, seq, last_hash) VALUES (" + p(1) + ", 0, '') ON DUPLICATE KEY UPDATE tenant_id = tenant_id"
		default:
			ensure = "INSERT OR IGNORE INTO audit_chain_heads (tenant_id, seq, last_hash) VALUES (" + p(1) + ", 0, '')"
		}
		if _, err = tx.ExecContext(ctx, ensure, tenantID); err != nil {
			return err
		}
		seq, last, err = read()
	}
	if err != nil {
		return err
	}
	seq++
	hash := chainHash(last, seq, tenantID, e)
	if _, err := tx.ExecContext(ctx, "INSERT INTO audit_chain (tenant_id, seq, event_id, prev_hash, chain_hash) VALUES ("+p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", "+p(5)+")", tenantID, seq, e.ID, last, hash); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE audit_chain_heads SET seq = "+p(1)+", last_hash = "+p(2)+" WHERE tenant_id = "+p(3), seq, hash, tenantID)
	return err
}

// ChainHead is the latest link of one tenant's chain.
type ChainHead struct {
	TenantID string `json:"tenant_id"`
	Seq      int64  `json:"seq"`
	Hash     string `json:"hash"`
}

// ChainHeads lists the head of every chain, for anchoring outside the database.
func (s Store) ChainHeads(ctx context.Context, db *sql.DB) ([]ChainHead, error) {
	rows, err := db.QueryContext(ctx, "SELECT tenant_id, seq, last_hash FROM audit_chain_heads ORDER BY tenant_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var heads []ChainHead
	for rows.Next() {
		var head ChainHead
		if err := rows.Scan(&head.TenantID, &head.Seq, &head.Hash); err != nil {
			return nil, err
		}
		heads = append(heads, head)
	}
	return heads, rows.Err()
}

// ChainReport is the result of verifying one tenant's chain.
type ChainReport struct {
	TenantID string `json:"tenant_id"`
	Links    int64  `json:"links"`
	// Legacy counts events written before chaining began (not covered).
	Legacy int64 `json:"legacy_events"`
	// Problems lists what does not add up; empty means the chain is intact.
	Problems []string  `json:"problems,omitempty"`
	Head     ChainHead `json:"head"`
}

// VerifyChain recomputes every link of a tenant's chain against the audit
// events and the stored head. anchor, when non-nil, must equal the link at
// its position (a head hash kept outside the database): it catches a chain
// that was rebuilt consistently.
func (s Store) VerifyChain(ctx context.Context, db *sql.DB, tenantID string, anchor *ChainHead) (ChainReport, error) {
	report := ChainReport{TenantID: tenantID}
	// #nosec G202 -- only placeholders are concatenated; every value is bound
	rows, err := db.QueryContext(ctx, `SELECT c.seq, c.prev_hash, c.chain_hash, c.event_id, e.id, e.action, e.target_id, e.request_id, e.actor_id, e.resource_type, e.created_at, e.before_json, e.after_json
FROM audit_chain c LEFT JOIN audit_events e ON e.id = c.event_id WHERE c.tenant_id = `+s.placeholder(1)+` ORDER BY c.seq`, tenantID)
	if err != nil {
		return report, err
	}
	defer rows.Close()
	problem := func(format string, args ...any) {
		if len(report.Problems) < 20 {
			report.Problems = append(report.Problems, fmt.Sprintf(format, args...))
		}
	}
	prev, want := "", int64(1)
	var firstCreated string
	for rows.Next() {
		var seq int64
		var prevHash, hash, linkedID string
		var id, action, target, request, actor, resource, created, before, after sql.NullString
		if err := rows.Scan(&seq, &prevHash, &hash, &linkedID, &id, &action, &target, &request, &actor, &resource, &created, &before, &after); err != nil {
			return report, err
		}
		report.Links++
		if seq != want {
			problem("link %d: expected position %d (a link is missing)", seq, want)
			want = seq
		}
		want++
		if prevHash != prev {
			problem("link %d: does not continue the previous link", seq)
		}
		if !id.Valid {
			problem("link %d: audit event %s is gone", seq, linkedID)
			prev = hash
			continue
		}
		if firstCreated == "" {
			firstCreated = created.String
		}
		event := chainedEvent{ID: id.String, Action: action.String, TargetID: target.String, RequestID: request.String, ActorID: actor.String, ResourceType: resource.String, CreatedAt: created.String, Before: before.String, After: after.String}
		if chainHash(prevHash, seq, tenantID, event) != hash {
			problem("link %d: audit event %s was changed", seq, linkedID)
		}
		if anchor != nil && anchor.Seq == seq && anchor.Hash != hash {
			problem("link %d: does not match the anchored hash (the chain was rewritten)", seq)
		}
		prev = hash
	}
	if err := rows.Err(); err != nil {
		return report, err
	}
	_ = rows.Close()
	var headSeq int64
	var headHash string
	err = db.QueryRowContext(ctx, "SELECT seq, last_hash FROM audit_chain_heads WHERE tenant_id = "+s.placeholder(1), tenantID).Scan(&headSeq, &headHash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if report.Links > 0 {
			problem("the chain head is missing")
		}
	case err != nil:
		return report, err
	default:
		report.Head = ChainHead{TenantID: tenantID, Seq: headSeq, Hash: headHash}
		if headSeq != report.Links || headHash != prev {
			problem("the chain head (%d) does not match the last link (%d): links were removed or added", headSeq, report.Links)
		}
	}
	if anchor != nil && anchor.Seq > report.Links {
		problem("the anchored position %d is beyond the chain (%d links): links were removed", anchor.Seq, report.Links)
	}
	if firstCreated != "" {
		var unchained int64
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events e WHERE e.tenant_id = "+s.placeholder(1)+" AND e.created_at > "+s.placeholder(2)+" AND NOT EXISTS (SELECT 1 FROM audit_chain c WHERE c.event_id = e.id)", tenantID, firstCreated).Scan(&unchained); err != nil {
			return report, err
		}
		if unchained > 0 {
			problem("%d audit events written after the chain began are not in it", unchained)
		}
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events e WHERE e.tenant_id = "+s.placeholder(1)+" AND e.created_at <= "+s.placeholder(2)+" AND NOT EXISTS (SELECT 1 FROM audit_chain c WHERE c.event_id = e.id)", tenantID, firstCreated).Scan(&report.Legacy); err != nil {
			return report, err
		}
	}
	return report, nil
}

// Relink appends audit events written after the chain began that never got a
// link (the chain was switched off for a while) to the end of the tenant's
// chain, oldest first. It returns how many it linked. Events from before the
// chain began stay outside it, and relinking vouches only for what the
// database holds now: those events were unprotected while they were unlinked.
func (s Store) Relink(ctx context.Context, db *sql.DB, tenantID string) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var first sql.NullString
	if err := tx.QueryRowContext(ctx, "SELECT MIN(e.created_at) FROM audit_chain c JOIN audit_events e ON e.id = c.event_id WHERE c.tenant_id = "+s.placeholder(1), tenantID).Scan(&first); err != nil {
		return 0, err
	}
	if !first.Valid {
		return 0, nil
	}
	rows, err := tx.QueryContext(ctx, "SELECT e.id, e.action, e.target_id, e.request_id, e.actor_id, e.resource_type, e.created_at, e.before_json, e.after_json FROM audit_events e WHERE e.tenant_id = "+s.placeholder(1)+" AND e.created_at > "+s.placeholder(2)+" AND NOT EXISTS (SELECT 1 FROM audit_chain c WHERE c.event_id = e.id) ORDER BY e.created_at, e.id", tenantID, first.String) // #nosec G202 -- only placeholders and fixed identifiers are concatenated; every value is bound
	if err != nil {
		return 0, err
	}
	var events []chainedEvent
	for rows.Next() {
		var e chainedEvent
		var target, request, actor, resource, before, after sql.NullString
		if err := rows.Scan(&e.ID, &e.Action, &target, &request, &actor, &resource, &e.CreatedAt, &before, &after); err != nil {
			_ = rows.Close()
			return 0, err
		}
		e.TargetID, e.RequestID, e.ActorID, e.ResourceType, e.Before, e.After = target.String, request.String, actor.String, resource.String, before.String, after.String
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	_ = rows.Close()
	previous := chainEnabled.Swap(true)
	defer chainEnabled.Store(previous)
	for _, e := range events {
		if err := s.appendChain(ctx, tx, tenantID, e); err != nil {
			return 0, err
		}
	}
	return len(events), tx.Commit()
}
