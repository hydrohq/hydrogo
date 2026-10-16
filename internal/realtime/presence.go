package realtime

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrInvalidPresence = errors.New("invalid editor presence")

const (
	DefaultPresenceTTL = 20 * time.Second
	maxPresenceEntries = 10000
)

type PresenceSession struct {
	TenantID string
	Resource string
	UserID   string
	Email    string
	Name     string
	Updated  time.Time
}

type ActiveEditor struct {
	UserID     string    `json:"user_id,omitempty"`
	Email      string    `json:"email,omitempty"`
	Name       string    `json:"name"`
	UpdatedAt  time.Time `json:"updated_at"`
	ActiveSecs int       `json:"active_seconds"`
}

// Presence stores short-lived editor heartbeats in bounded process memory.
// It contains no document contents and expires idle sessions automatically.
type Presence struct {
	mu      sync.Mutex
	entries map[string]PresenceSession
	TTL     time.Duration
	Now     func() time.Time
}

func (p *Presence) Heartbeat(session PresenceSession) error {
	if !validPresenceValue(session.TenantID, 128) || !validPresenceValue(session.Resource, 128) || !validPresenceValue(session.UserID, 128) || !validPresenceValue(session.Name, 160) {
		return ErrInvalidPresence
	}
	if session.Email != "" && !validPresenceValue(session.Email, 160) {
		return ErrInvalidPresence
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	p.pruneLocked(now)
	key := presenceKey(session.TenantID, session.Resource, session.UserID)
	if _, exists := p.entries[key]; !exists && len(p.entries) >= maxPresenceEntries {
		return errors.New("editor presence capacity reached")
	}
	session.Updated = now
	if p.entries == nil {
		p.entries = make(map[string]PresenceSession)
	}
	p.entries[key] = session
	return nil
}

func (p *Presence) Leave(tenantID, resource, userID string) {
	if !validPresenceValue(tenantID, 128) || !validPresenceValue(resource, 128) || !validPresenceValue(userID, 128) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	key := presenceKey(tenantID, resource, userID)
	delete(p.entries, key)
}

func (p *Presence) Active(tenantID, resource, excludeUserID string) []ActiveEditor {
	if !validPresenceValue(tenantID, 128) || !validPresenceValue(resource, 128) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	p.pruneLocked(now)
	result := make([]ActiveEditor, 0, 4)
	for _, session := range p.entries {
		if session.TenantID != tenantID || session.Resource != resource || session.UserID == excludeUserID {
			continue
		}
		result = append(result, ActiveEditor{
			UserID:     session.UserID,
			Email:      session.Email,
			Name:       session.Name,
			UpdatedAt:  session.Updated,
			ActiveSecs: int(now.Sub(session.Updated).Seconds()),
		})
	}
	return result
}

func (p *Presence) now() time.Time {
	if p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}

func (p *Presence) pruneLocked(now time.Time) {
	ttl := p.TTL
	if ttl <= 0 || ttl > 5*time.Minute {
		ttl = DefaultPresenceTTL
	}
	for key, session := range p.entries {
		if now.Sub(session.Updated) >= ttl {
			delete(p.entries, key)
		}
	}
}

func presenceKey(tenantID, resource, userID string) string {
	return tenantID + "\x00" + resource + "\x00" + userID
}

func validPresenceValue(value string, max int) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= max && !strings.ContainsAny(value, "\x00\r\n")
}
