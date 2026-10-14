# R5 — Security abuse corpus baseline

**Status:** Partial local contract.

The shared test corpus now covers open redirects, encoded slash confusion,
header injection, external URLs, and canonical-host bypass attempts. Upload,
SSRF, decompression-bomb, and malformed-multipart vectors remain open until
their respective validators are wired into the same corpus.

Evidence: `go test ./internal/security` passed on 2026-09-14.
