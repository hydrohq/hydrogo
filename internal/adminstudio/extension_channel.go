package adminstudio

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidExtensionChannel = errors.New("invalid extension channel")
	ErrExtensionChannelDenied  = errors.New("extension channel denied")
)

// ChannelPrincipal is server-derived only. Browser-provided actor, tenant, or
// capability fields are never accepted as authority.
type ChannelPrincipal struct {
	SessionID string
	UserID    string
	TenantID  string
}

type ExtensionChannel struct {
	ID          string
	Challenge   string
	ExtensionID string
	SurfaceID   string
	Origin      string
	Principal   ChannelPrincipal
	Permissions map[string]bool
	ExpiresAt   time.Time
	Bound       bool
	requests    map[string]struct{}
}

// ChannelManager holds short-lived, per-process channel state. Losing this
// state fails closed; a browser must re-open the protected host shell.
type ChannelManager struct {
	mu       sync.Mutex
	channels map[string]*ExtensionChannel
	now      func() time.Time
	ttl      time.Duration
	max      int
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{channels: make(map[string]*ExtensionChannel), now: time.Now, ttl: 5 * time.Minute, max: 1024}
}

func (m *ChannelManager) Issue(principal ChannelPrincipal, extensionID, surfaceID, origin string, permissions []string) (ExtensionChannel, error) {
	if m == nil || principal.SessionID == "" || principal.UserID == "" || extensionID == "" || surfaceID == "" || origin == "" {
		return ExtensionChannel{}, ErrInvalidExtensionChannel
	}
	id, err := extensionChannelToken()
	if err != nil {
		return ExtensionChannel{}, err
	}
	challenge, err := extensionChannelToken()
	if err != nil {
		return ExtensionChannel{}, err
	}
	allowed := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		if permission != "" {
			allowed[permission] = true
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(m.now())
	if len(m.channels) >= m.max {
		return ExtensionChannel{}, ErrExtensionChannelDenied
	}
	channel := &ExtensionChannel{ID: id, Challenge: challenge, ExtensionID: extensionID, SurfaceID: surfaceID, Origin: origin, Principal: principal, Permissions: allowed, ExpiresAt: m.now().Add(m.ttl), requests: make(map[string]struct{})}
	m.channels[id] = channel
	return cloneChannel(channel), nil
}

func (m *ChannelManager) Bind(id, challenge string, principal ChannelPrincipal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	channel, ok := m.channels[id]
	if !ok || !m.now().Before(channel.ExpiresAt) || channel.Bound || !channelChallengeEqual(channel.Challenge, challenge) || channel.Principal != principal {
		return ErrExtensionChannelDenied
	}
	channel.Bound = true
	return nil
}

func (m *ChannelManager) Authorize(id, requestID, permission string, principal ChannelPrincipal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	channel, ok := m.channels[id]
	if !ok || !channel.Bound || !m.now().Before(channel.ExpiresAt) || channel.Principal != principal || !channel.Permissions[permission] || requestID == "" {
		return ErrExtensionChannelDenied
	}
	if _, replay := channel.requests[requestID]; replay {
		return ErrExtensionChannelDenied
	}
	if len(channel.requests) >= 256 {
		return ErrExtensionChannelDenied
	}
	channel.requests[requestID] = struct{}{}
	return nil
}

func (m *ChannelManager) Revoke(id string, principal ChannelPrincipal) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if channel, ok := m.channels[id]; ok && channel.Principal == principal {
		delete(m.channels, id)
	}
}

func (m *ChannelManager) pruneLocked(now time.Time) {
	for id, channel := range m.channels {
		if !now.Before(channel.ExpiresAt) {
			delete(m.channels, id)
		}
	}
}

func cloneChannel(c *ExtensionChannel) ExtensionChannel {
	result := *c
	result.Permissions = make(map[string]bool, len(c.Permissions))
	for k, v := range c.Permissions {
		result.Permissions[k] = v
	}
	result.requests = nil
	return result
}

func extensionChannelToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func channelChallengeEqual(expected, actual string) bool {
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
