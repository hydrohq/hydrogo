# Built-in Admin mutation CSRF route matrix — 2026-10-09

`TestBuiltInMutationRoutesAreProtectedAndCSRFChecked` now wraps every
registered mutation handler in the Core site, site-with-forms, news, page,
media, and identity/account route compositions with a dispatch counter. For
each non-GET `/admin/` and `/api/media` mutation, it submits both a request
without CSRF and a request whose CSRF cookie and header disagree. Both must
return `403`, and neither request may reach the route handler.

Verification passed:

```sh
GOCACHE=/tmp/hydrogo-go-cache GOTMPDIR=/tmp/hydrogo-go-tmp \
TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off \
  go test ./cmd/hydro -run '^TestBuiltInMutationRoutesAreProtectedAndCSRFChecked$' -count=1 -p 1 -timeout=4m
GOCACHE=/tmp/hydrogo-go-cache GOTMPDIR=/tmp/hydrogo-go-tmp \
TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off \
  go test ./cmd/hydro -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-go-cache GOTMPDIR=/tmp/hydrogo-go-tmp \
TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off go vet ./cmd/hydro
git diff --check
```

This proves route protection and early CSRF rejection for these in-process
compositions. It does not prove role-specific persisted state, production
session middleware, the separately composed backup/security/audit surfaces,
browser behavior, or PostgreSQL; those remain covered by their own tests or
open acceptance gates.
