# HydroGo — 8-Round Security Audit & Scan Playbook

Each round covers a specific bounded area. Execute sequentially, verify at the end of each round, and pause if issues are found.

## Ground Rules (Enforced for Every Round)

For every vulnerability finding:
1. **Write the failing test first and run it**, record output.
2. Apply the minimal remediation.
3. Show the test passing.
4. **Commit the test and fix together** — never commit fixes without tests.
5. Unverified findings are not documented as resolved.

---

## ROUND 1 — `modules/members` (Authorization / IDOR)

```text
Audit modules/members ONLY. Do not read outside modules/.

1. For EVERY handler that takes an ID from the path: verify that it checks whether the ID belongs to the caller's tenant, rather than merely that it exists.
2. Password reset and email change flows: verify tokens are single-use, bounded by expiration, and bound to the issuing account. Ensure comparisons are constant-time.
3. Session invalidation: verify session revocation occurs on password change, logout, and account deactivation.
4. Handlers returning counts, lists, or exports: verify they apply the same tenant scoping filters as single-record endpoints.
5. Form bodies: verify wrapped in http.MaxBytesReader before ParseForm.

For every finding: failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 2 — `internal/media` (File Upload & Storage)

```text
Audit internal/media ONLY.

Focus on file handling:
1. Path traversal: verify stored filenames and paths cannot escape storage roots. Verify symlink resolution.
2. Upload size enforcement: verify size is bounded BEFORE streaming into memory or disk.
3. MIME verification: do not trust client-declared Content-Type; sniff content safely. Verify SVG sanitization.
4. Filename generation: ensure user-supplied names are sanitized or replaced with random identifiers.
5. Deletion boundaries: prevent cross-tenant deletion where ID and storage key diverge.

Failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 3 — `internal/auth` (Passwords, Sessions, Lockout)

```text
Audit internal/auth ONLY.

1. Password hashing: verify algorithm and cost parameters (Argon2id). Ensure constant-time comparison on hashes and tokens.
2. Session ID generation: verify crypto/rand source with sufficient entropy.
3. Lockout & rate limiting: verify protections against brute-force login attacks.
4. Token storage: ensure tokens are stored hashed (SHA-256) and expiration is enforced on read.
5. Timing attacks: audit endpoints for account enumeration or timing discrepancies.

Failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 4 — `internal/adminstudio` (Admin UI & Handlers)

```text
Audit internal/adminstudio (excluding already-audited sqlstore).

1. Input sinks: trace inputs reaching filesystem paths, URLs, or HTML attributes. Check href, src, action, and path joins.
2. Handler authorization: map every admin handler to its required permission scope.
3. Template escaping: verify Templ auto-escaping is maintained and raw HTML interpolation is strictly sanitized.
4. File & directory management: audit deletion paths and root boundary validation.

Failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 5 — `internal/database` (SQL & Migration Safety)

```text
Audit internal/database ONLY.

1. Query parameterization: ensure all queries use parameter binding rather than string concatenation. Verify allowlists for table/column names.
2. Migration safety: verify non-blocking operations on non-empty tables and presence of rollback scripts.
3. Transaction scoping: verify multi-statement writes execute within a single transaction.
4. Query efficiency: identify and eliminate N+1 loops.
5. Resource lifecycle: ensure rows, statements, and connections are closed on all execution branches.

Failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 6 — `extension/v1` (Public API Surface)

```text
Audit extension/v1 ONLY.

1. Authentication: ensure all endpoints enforce valid credentials unless explicitly declared public.
2. Authorization: verify role and tenant scoping on every route.
3. Input validation: bound string lengths, validate enums, and reject unrecognized payloads.
4. Mass assignment: reject direct decoding into sensitive entity fields (role, tenant_id, status, etc.).
5. Response leakage: ensure internal models and secrets are not leaked in JSON responses.

Failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 7 — `internal/mailer` & `internal/observability`

```text
Audit internal/mailer and internal/observability ONLY.

1. Template injection: ensure user values in email templates are escaped.
2. Header injection: strip or reject CR/LF characters in email headers (To, From, Subject).
3. Log hygiene: ensure credentials, tokens, session IDs, and card data are redacted from logs and errors.
4. Telemetry PII: verify metrics and traces strip personally identifiable information.
5. Panic recovery: ensure recovery handlers do not log sensitive request bodies.

Failing test -> minimal fix -> passing test -> atomic commit.
```

## ROUND 8 — `internal/backup` (Restore Safety)

```text
Audit internal/backup ONLY.

1. Restore boundaries: ensure restore operations cannot write outside designated targets.
2. Archive extraction: prevent path traversal (../, absolute paths, external symlinks) in archive unpacking.
3. Destructive actions: audit drop, truncate, or overwrite operations for confirmation and access checks.
4. Encryption: ensure secrets and archives are encrypted with managed keys.
5. Concurrency: verify locking during concurrent restore invocations.

Failing test -> minimal fix -> passing test -> atomic commit.
```

---

## Post-Round Verification

```bash
git log -1 --format='%an  %s'
go build ./... && go test ./...
```
