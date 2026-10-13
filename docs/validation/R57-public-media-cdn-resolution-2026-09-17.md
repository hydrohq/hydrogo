# R57 — Public media CDN resolution

**Date:** 2026-09-17  
**Status:** Successful (local contract)

Public Pages hero blocks now use the configured blob store's optional
`PublicURL` capability when available. Without a configured public provider, or
when URL resolution fails, the existing same-origin `/media/{id}` route remains
the safe fallback. Private or non-ready assets are never exposed.

The media service exposes this capability without leaking S3/CDN details into
the Pages module boundary. Contract tests cover both fallback and configured
CDN URL behavior.

Verification: `go test ./internal/media ./modules/pages/transport` and full local
verification hook.
