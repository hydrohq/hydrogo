# hydrogo — 8 Tur Kapsamlı Tarama

Her turun kendi alanı var. Sırayla at, her turun sonunda kontrol et, gerekirse dur.
Maliyet: luna 0.10/0.50. Bir tur ~0.03–0.08 $. Sekiz tur ~0.50 $.

## Kural (her prompt'ta var, tekrarlama)

Her bulgu için: önce **başarısız testi yaz ve çalıştır**, çıktıyı göster,
sonra minimum düzeltme, sonra testin geçtiğini göster.
Testi **commit et** — çalıştırıp atma.
Doğrulanamayan bulgu raporlanmaz.
Düzeltme kendisi commit edilsin, sadece test commitlenmeden düzeltme yapma.

---

## TUR 1 — modules/members (yetki / IDOR)

```
Audit modules/members ONLY. Do not read anything outside modules/.

This is the highest-value unaudited surface in the repo. Every defect found so
far in this codebase has been at the authorization or tenant-isolation
boundary, so look there hardest.

1. For EVERY handler that takes an id from the path: does it verify that id
   belongs to the caller's tenant, or only that it exists? A logged-in member
   reading or mutating another member's record is the bug we want.
2. Password reset and email change flows: is the token single-use, does it
   expire, and is it bound to the account it was issued for? Look for any
   comparison that is not constant-time.
3. Session invalidation: is the session destroyed on password change, on
   logout, and on account deactivation? Is it destroyed on ALL of them, or
   only the easy one?
4. Any handler that returns a count, a list, or an export: does it apply the
   same tenant filter that the single-record handler does?
5. Every form body: is it wrapped in http.MaxBytesReader before ParseForm?

For every finding: write a failing test, run it, show the failure, apply the
minimal fix, show it passing, then COMMIT the fix and its test together.
Plain commit message, no Co-Authored-By trailer.
```

## TUR 2 — internal/media (dosya yükleme)

```
Audit internal/media ONLY.

Focus on file handling, in this order:

1. Path traversal: for every place a stored filename or path is used to read,
   write, or delete a file, prove it cannot escape the storage root. Check
   symlink handling too - a stored name that is a symlink pointing outside the
   root is the same bug with a different spelling.
2. Size enforcement: is the upload size bounded BEFORE the body is read into
   memory, or only checked after? Reading then rejecting is a memory DoS.
3. MIME validation: is the declared Content-Type trusted, or is the content
   sniffed? If declared type is trusted, is that exploitable given what the
   files are served as later (inline vs attachment, Content-Disposition)?
4. Filename generation: is the stored name derived from user input, or
   generated? If derived, is it sanitized for the filesystem it lands on?
5. Deletion: can one call delete a file another tenant owns, if the record id
   and the storage key diverge?

Write a failing test, show it fail, fix minimally, show it pass, commit both
together. No Co-Authored-By trailer.
```

## TUR 3 — internal/auth (parola, session, lockout)

```
Audit internal/auth ONLY.

1. Password hashing: find the hash function. Report the algorithm and the cost
   parameter. Then read every call site - is any code path hashing with a
   weaker algorithm, or comparing with == instead of the constant-time helper?
   A single == on a token or hash is a high-severity finding.
2. Session id generation: trace it to its source. Is it crypto/rand with enough
   entropy, or derived from something predictable (time, counter, user id)?
3. Lockout and rate limit: can login or password-reset be brute-forced? Is the
   limiter keyed on something the attacker controls (user-supplied email), and
   does that let an attacker lock out arbitrary accounts?
4. Token storage: are reset/verify tokens stored hashed or plaintext? Do they
   expire, and is expiry actually checked on the read path?
5. Timing: any endpoint whose response time reveals whether an account exists.

Write a failing test, show it fail, fix minimally, show it pass, commit both.
No Co-Authored-By trailer.
```

## TUR 4 — internal/adminstudio (25 dosya, sadece sqlstore bakıldı)

```
Audit internal/adminstudio EXCEPT internal/adminstudio/sqlstore, which has
already been audited and fixed.

The sqlstore audit found safeTarget accepting backslash-prefixed paths. Assume
the same class of bug exists elsewhere in this package:

1. Every input that reaches a filesystem path, a URL, or an HTML attribute.
   Trace it from the HTTP handler to its sink. Report anything that reaches
   href, src, action, or a path join without validation.
2. Every handler in the studio UI: who is authorized to call it? Build a table
   of handler -> required permission, and flag any handler with no check or a
   weaker check than its siblings.
3. Any template that interpolates a value. Templ (admin_ux.templ) escapes by
   default - find the places that bypass it, and any use of raw HTML insertion
   with user data.
4. File and directory management handlers: deletion paths, path construction,
   and whether an id is re-resolved to a path inside the root.

Write a failing test, show it fail, fix minimally, show it pass, commit both.
No Co-Authored-By trailer.
```

## TUR 5 — internal/database (SQL ve migration güvenliği)

```
Audit internal/database ONLY.

1. Query construction: find every place a SQL string is built by
   concatenation rather than parameter binding. Trace the concatenated value
   back to its source. If any part of it can be influenced by a request, that
   is critical. Note: table and column names cannot be parameterised, so check
   whether they are validated against an allowlist instead.
2. Migration safety: for each migration, can it run on a non-empty table
   without locking or losing data? Flag destructive operations with no
   rollback path - DROP COLUMN, NOT NULL without default, type narrowing.
3. Transactions: find multi-statement writes that are not in a transaction,
   and transactions that commit before all steps succeed.
4. N+1: any loop that issues a query per iteration. Report the call site and
   the call depth.
5. Connection lifecycle: any path where a connection, row, or prepared
   statement is not closed on the error branch.

Write a failing test, show it fail, fix minimally, show it pass, commit both.
No Co-Authored-By trailer.
```

## TUR 6 — extension/v1 (dış API yüzeyi)

```
Audit extension/v1 ONLY. This is the externally reachable API surface, so
treat every handler as attacker-controlled input.

1. Authentication on every route: is there any handler reachable without a
   valid credential? List them.
2. Authorization: for each route, what does it check beyond "logged in"?
3. Input validation: every JSON body field, every query parameter, every path
   segment. Report unbounded string lengths and unvalidated enums.
4. Mass assignment: any handler that decodes a request body straight into a
   domain struct, so a client can set a field they should not (role, tenant_id,
   id, status, version).
5. Response leakage: any handler returning a struct with unexported-but-
   serialized fields, or a SELECT * projected into a public response.

For mass assignment, the failing test is easy and worth doing for every case:
POST with a role or tenant_id field set and assert the stored value did not
change. Write it, show it fail, fix minimally, show it pass, commit both.
No Co-Authored-By trailer.
```

## TUR 7 — internal/mailer + internal/observability

```
Audit internal/mailer and internal/observability ONLY.

1. Template injection: every email body is built somewhere. If any part is
   concatenated from user data into a template that is then rendered, that is a
   finding. Report what the attacker controls in the output.
2. Header injection: any user value placed into a mail header (To, From,
   Reply-To, Subject, custom headers) without rejecting CR and LF.
3. Logs: any request that logs a password, token, session id, API key, or full
   card number, including inside error messages. Check the redacting helpers
   actually exist and are used.
4. PII in telemetry: any metric or trace that carries an email address, a
   user id, or a URL with a token in the query string.
5. Panic recovery: does an observability middleware recover a panic, and if so
   does the recovery log the request body?

Write a failing test, show it fail, fix minimally, show it pass, commit both.
No Co-Authored-By trailer.
```

## TUR 8 — internal/backup (geri yükleme güvenliği)

```
Audit internal/backup ONLY. This code writes and overwrites user data, so
treat it as the most destructive surface in the repo.

1. Restore: what does it accept? If a restore path is built from a name or id
   that came from a request or a manifest, can it write outside the target
   directory? Prove it either way with a test.
2. Archive extraction: for every tar/zip extraction, check for path traversal
   entries (../, absolute paths, and symlinks pointing outside). Report whether
   the extraction validates the resolved path stays inside the root.
3. Destructive operations: list every function that drops, truncates, or
   overwrites. For each, is the target path validated first? Is there a
   confirmation, and can the confirmation be bypassed?
4. Credentials: does a backup contain secrets, and if so are they encrypted? Is
   the restore side able to read an archive encrypted with a key from config?
5. Concurrency: two restores of the same target at once.

Write a failing test, show it fail, fix minimally, show it pass, commit both.
No Co-Authored-By trailer.
```

---

## Her turdan sonra

```fish
cd ~/hydrogo
git log -1 --format='%an  %s'    # yazar sen olmalısın
git log -1 --format='%(trailers)' # boş olmalı
go build ./... && go test ./...  # yeşil olmalı
```

## Bir turda takılırsan

`/autonomy stop` — sonra bana çıktıyı yapıştır, doğrularım.
