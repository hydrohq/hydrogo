package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type adminIPBan struct {
	Address   string     `json:"address"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (a *serveApp) adminIPBanListHandler(w http.ResponseWriter, r *http.Request) {
	if !activePlatformAdmin(auth.UserFromContext(r.Context())) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	entries, err := a.banStore.ListActive(r.Context(), a.db, time.Now().UTC(), 4096)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "IP rules are temporarily unavailable", nil)
		return
	}
	bans := make([]adminIPBan, 0, len(entries))
	for _, entry := range entries {
		ban := adminIPBan{Address: entry.Prefix.String()}
		if !entry.Expires.IsZero() {
			expires := entry.Expires.UTC()
			ban.ExpiresAt = &expires
		}
		bans = append(bans, ban)
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, bans, nil)
}

// adminIPBanMutationHandler adds (remove=false) or removes an IP ban rule and
// updates the in-memory blocklist.
func (a *serveApp) adminIPBanMutationHandler(remove bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		var input struct {
			Address  string `json:"address"`
			Duration string `json:"duration"`
		}
		if !api.DecodeJSON(w, r, &input, 8<<10) {
			return
		}
		prefix, err := parseIPBanPrefix(input.Address)
		if err != nil {
			api.WriteValidation(w, r, map[string]string{"address": "invalid"})
			return
		}
		db, banStore, ipBlocklist := a.db, a.banStore, a.ipBlocklist
		var expires time.Time
		if remove {
			err = banStore.Remove(r.Context(), db, prefix)
			if err == nil {
				err = ipBlocklist.Unban(prefix.String())
			}
		} else {
			if raw := strings.TrimSpace(input.Duration); raw != "" {
				duration, parseErr := time.ParseDuration(raw)
				if parseErr != nil || duration < 0 || duration > 365*24*time.Hour {
					api.WriteValidation(w, r, map[string]string{"duration": "invalid"})
					return
				}
				if duration > 0 {
					expires = time.Now().UTC().Add(duration)
				}
			}
			err = banStore.Put(r.Context(), db, prefix, expires)
			if err == nil {
				err = ipBlocklist.Ban(prefix.String(), expires)
			}
		}
		if err != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "IP rule could not be updated", nil)
			return
		}
		action := "security.ip_ban.created"
		if remove {
			action = "security.ip_ban.removed"
		}
		if a.auditRecorder != nil {
			_ = a.auditRecorder.Record(r.Context(), action, prefix.String())
		}
		if remove {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ban := adminIPBan{Address: prefix.String()}
		if !expires.IsZero() {
			ban.ExpiresAt = &expires
		}
		api.WriteData(w, http.StatusCreated, ban, nil)
	}
}
