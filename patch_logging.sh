#!/bin/bash
awk '
/slog.String\("session_id", session.ID\)/ { next }
/slog.String\("actor_id", user.ID\), slog.String\("user_id", user.ID\)/ { next }
/if user := auth.UserFromContext\(r.Context\(\)\); user != nil && user.ID != "" {/ {
    skip = 1
    next
}
/if session := auth.SessionFromContext\(r.Context\(\)\); session != nil && session.ID != "" {/ {
    skip = 1
    next
}
{
    if (skip && /}/) {
        skip = 0
        next
    }
    if (!skip) print
}
' internal/observability/logging.go > temp.go && mv temp.go internal/observability/logging.go
