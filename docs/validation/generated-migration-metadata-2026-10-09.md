# Generated migration metadata contract — 2026-10-09

The generator regression now verifies that generated migration metadata is
dialect-aware, transactional, additive, and recoverable. It computes SHA-256
from the generated `migration.sql` and asserts both the SQLite and PostgreSQL
checksum entries match that exact content. It also checks that generated
metadata preserves the explicit forward-only/restore recovery guidance.

Verification passed:

```sh
GOCACHE=/tmp/hydrogo-go-cache GOTMPDIR=/tmp/hydrogo-go-tmp \
TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off \
  go test ./internal/generator -run '^TestGeneratedMigrationMetadataIsDialectAwareAndNonDestructive$' -count=1 -p 1 -timeout=4m
GOCACHE=/tmp/hydrogo-go-cache GOTMPDIR=/tmp/hydrogo-go-tmp \
TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off \
  go test ./internal/generator -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-go-cache GOTMPDIR=/tmp/hydrogo-go-tmp \
TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off go vet ./internal/generator
git diff --check
```

This verifies generator output metadata against its generated SQL in a unit
test. It does not replace applying each migration through a live SQLite and
PostgreSQL adapter, nor certify tenant policy, concurrency, production rollback,
or Blue/Green deployment.
