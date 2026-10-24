package security

import (
	"errors"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

type IPBan struct {
	Prefix  netip.Prefix
	Expires time.Time
}

// IPBlocklist is a bounded, process-local fast path. Durable rules are refreshed
// from the security store by the runtime and administrative changes update it
// immediately in the current process.
type IPBlocklist struct {
	mu        sync.RWMutex
	entries   map[string]IPBan
	max       int
	now       func() time.Time
	Extractor interface {
		Extract(r *http.Request) string
	}
}

func NewIPBlocklist(maxEntries int) *IPBlocklist {
	if maxEntries < 1 {
		maxEntries = 1
	}
	return &IPBlocklist{entries: make(map[string]IPBan), max: maxEntries, now: time.Now}
}

func (b *IPBlocklist) Ban(value string, expires time.Time) error {
	if b == nil {
		return ErrBlocklistUnavailable
	}
	prefix, err := parseBanPrefix(value)
	if err != nil {
		return err
	}
	key := prefix.String()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.evictExpired()
	if _, exists := b.entries[key]; !exists && len(b.entries) >= b.max {
		return ErrBlocklistFull
	}
	b.entries[key] = IPBan{Prefix: prefix, Expires: expires.UTC()}
	return nil
}

func (b *IPBlocklist) Unban(value string) error {
	if b == nil {
		return ErrBlocklistUnavailable
	}
	prefix, err := parseBanPrefix(value)
	if err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.entries, prefix.String())
	b.mu.Unlock()
	return nil
}

func (b *IPBlocklist) Blocked(address string) bool {
	if b == nil {
		return false
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(address))
	if err != nil {
		return false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	now := time.Now()
	if b.now != nil {
		now = b.now()
	}
	for _, entry := range b.entries {
		if !entry.Expires.IsZero() && !now.Before(entry.Expires) {
			continue
		}
		if entry.Prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (b *IPBlocklist) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var address string
		if b.Extractor != nil {
			address = b.Extractor.Extract(r)
		} else {
			address = requestmeta.ClientIP(r)
		}
		if b.Blocked(address) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (b *IPBlocklist) Replace(entries []IPBan) error {
	if b == nil {
		return ErrBlocklistUnavailable
	}
	if len(entries) > b.max {
		return ErrBlocklistFull
	}
	next := make(map[string]IPBan, len(entries))
	for _, entry := range entries {
		prefix, err := parseBanPrefix(entry.Prefix.String())
		if err != nil {
			return err
		}
		next[prefix.String()] = IPBan{Prefix: prefix, Expires: entry.Expires.UTC()}
	}
	b.mu.Lock()
	b.entries = next
	b.mu.Unlock()
	return nil
}

func (b *IPBlocklist) evictExpired() {
	now := time.Now()
	if b.now != nil {
		now = b.now()
	}
	for key, entry := range b.entries {
		if !entry.Expires.IsZero() && !now.Before(entry.Expires) {
			delete(b.entries, key)
		}
	}
}

var (
	ErrBlocklistUnavailable = errors.New("IP blocklist is unavailable")
	ErrBlocklistFull        = errors.New("IP blocklist capacity reached")
)

func parseBanPrefix(value string) (netip.Prefix, error) {
	value = strings.TrimSpace(value)
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), nil
	}
	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, errors.New("invalid IP address or CIDR")
	}
	bits := 128
	if address.Is4() {
		bits = 32
	}
	return netip.PrefixFrom(address, bits), nil
}
