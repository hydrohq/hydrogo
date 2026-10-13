# HydroGo — Comprehensive Security Audit Plan (Rounds 2–8)

This document is the structured security audit plan for HydroGo. Each round focuses on **one bounded area** at a time. Round 1 (`modules/members`) is completed and merged into `main`.

## Execution Workflow

1. Open a branch for each round: `git checkout -b audit-<round-num> main`
2. Implement and test within that branch, committing atomically.
3. Once `go build ./...` and `go test ./...` are green: fast-forward merge the branch into `main`, then delete the topic branch.
4. Proceed to the next round.

**Rule:** Do not execute rounds in a single unbounded session. Stop and report findings upon completing each round to guarantee rigorous verification.

---

# COMMON RULES — Enforced Across All Rounds

1. **Test First:** Write the failing test first, RUN IT, SHOW FAILURE OUTPUT. Then apply the fix. Never write code before proving the failure.
2. **Prove Test Authenticity:** Temporarily revert the fix and demonstrate that the test breaks again. A test that never failed is not valid proof.
3. **Test the Real Layer:** If the vulnerability resides in SQL, filesystem, or HTTP middleware, the test must exercise the real implementation. In-memory doubles that bypass the tested behavior prove nothing.
4. **No Grep Tests:** Reading a `.go` source file and checking `strings.Contains` is a grep, not a behavioral test.
5. **Document Side Effects:** When modifying an implementation, document changed behaviors (e.g., edge cases now rejected, distinct error codes returned).
6. **No Unproven Claims:** If a defect cannot be demonstrated with a failing test, report it as unverified with the exact blockers encountered.
7. **Commit Together:** Fix and accompanying regression tests must be delivered in the **same commit**.
8. **Pre-Commit Verification:** Run `go build ./...`, `go test ./... -count=1`, and `go vet ./...` before committing.

---

# ROUND 2 — `internal/auth` (26 files / ~5880 lines)

```text
Audit internal/auth ONLY. Do not read outside internal/auth except for interface definitions.

Focus on credential and session flows (recovery.go, mfa.go, invite.go, password.go, mobile.go, session code):

1. Password Recovery (recovery.go): Verify tokens are single-use, consumed immediately upon use, bounded by expiration, stored hashed (SHA-256), compared in constant-time, and strictly bound to the target account.
2. MFA (mfa.go): Audit TOTP brute-force rate-limiting, server-side attempt counters, replay protection on consumed codes, and single-use recovery codes.
3. Sessions (service.go): Verify crypto/rand entropy, secure session token hashing, expiration enforcement on every read path, and comprehensive revocation on logout/password change.
```

---

# ROUND 3 — `internal/media` (File Upload & Processing)

```text
Audit internal/media ONLY.

1. Storage Path Traversal: Ensure stored file keys cannot escape storage root via relative prefixes or symlinks.
2. Filename Generation: Sanitize filenames; avoid relying on user-provided strings for disk storage.
3. MIME Validation: Do not trust client Content-Type headers; sniff magic bytes. Verify Content-Disposition and SVG script sanitization.
4. Upload Limits: Enforce size boundaries BEFORE streaming file bodies into memory. Bound file counts in multipart requests.
5. Deletion Isolation: Ensure a delete request cannot delete another tenant's files.
```

---

# ROUND 4 — `extension/v1` (External API Surface)

```text
Audit extension/v1 ONLY. Treat every request as hostile.

1. Authentication: Enforce credentials across all non-public endpoints.
2. Authorization: Map routes to required permissions and ensure strict role/tenant verification.
3. Mass Assignment: Prohibit direct request decoding into domain structs; protect internal fields (id, tenant_id, role, status, etc.).
4. Input Validation: Bound string lengths, validate enums, and parse numeric parameters safely.
5. Data Leakage: Ensure sensitive internal models and hashes are never serialized to JSON responses.
6. Idempotency: Verify protection against duplicate mutations (duplicate billing, duplicate submissions).
```

---

# ROUND 5 — `internal/backup` (Backup & Restore Safety)

```text
Audit internal/backup ONLY.

1. Archive Extraction: Validate extracted paths stay within the root directory; reject ../ and symlink traversals.
2. Restore Boundaries: Ensure restore targets cannot write outside the designated directory.
3. Destructive Operations: Audit drop, truncate, or overwrite operations for explicit confirmations.
4. Credentials & Keys: Verify backups do not leak unencrypted credentials and archives use salted key derivation.
5. Concurrency: Prevent concurrent restore operations against the same target.
```

---

# ROUND 6 — `internal/adminstudio` (Admin UI & Handlers)

```text
Audit internal/adminstudio (excluding already-audited sqlstore).

1. Handler Authorization: Ensure every admin handler verifies appropriate permission levels.
2. Sinks & Reflection: Trace inputs to href, src, action, or redirects to eliminate open redirects and injection.
3. Filesystem Actions: Verify path resolution and symlink containment.
4. Templ Rendering: Ensure auto-escaping is maintained and raw HTML interpolation is sanitized.
5. CSRF Protection: Ensure every state-mutating request validates CSRF tokens.
```

---

# ROUND 7 — `internal/database` (SQL & Migrations)

```text
Audit internal/database ONLY.

1. Query Parameterization: Ensure all dynamic queries use bind parameters. Table and column identifiers must be allowlisted.
2. Migration Safety: Verify non-destructive operations and presence of rollback capabilities.
3. Transactions: Ensure multi-statement mutations execute within atomic transactions.
4. N+1 Loops: Eliminate queries within unbounded iteration loops.
5. Connection Lifecycle: Guarantee rows, statements, and connections close on all error branches.
6. Pool Configuration: Ensure MaxOpenConns, MaxIdleConns, and ConnMaxLifetime are configured safely.
```

---

# ROUND 8 — `internal/mailer` & `internal/observability`

```text
Audit internal/mailer AND internal/observability.

1. Header Injection: Reject CR/LF characters in email headers (To, From, Subject, CC, BCC).
2. Template Injection: Ensure email bodies do not concatenate unescaped user input into templates.
3. Log Hygiene: Verify passwords, session tokens, and keys are redacted from logs and errors.
4. Telemetry Privacy: Ensure traces and metrics strip PII and high-cardinality user data.
5. Panic Recovery: Ensure panic recovery middleware does not log sensitive request payloads.
6. Error Sanitization: Strip internal file paths and SQL snippets before returning errors to clients.
```

---

## Post-Round Verification Checklist

```bash
go build ./... && go test ./... -count=1
git log -1 --format='%an  %s'
git log -1 --format='%(trailers)'
git status --short
```
