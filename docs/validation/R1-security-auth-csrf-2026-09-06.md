# R1 Security, Authentication and CSRF Validation

**Date:** 2026-09-06  
**Roadmap item:** R1 Runtime Security Foundation  
**Status:** In fix cycle; local tests passed, CI and full auth flow not yet closed.

## Validated Behaviors

1. **Argon2id Password Hashing:**
   - Password hashing and constant-time (`subtle.ConstantTimeCompare`) validation implemented with OWASP compliant parameters (`memory=64MB`, `iterations=1`, `parallelism=4`, `saltLength=16`, `keyLength=32`).
   - To prevent timing attacks, invalid user queries trigger a fake validation with the same cost as an actual user hash.

2. **Opaque DB-Backed Session Management:**
   - 32-byte cryptographic random token generation (`crypto/rand`).
   - Only the SHA-256 digest (`token_hash`) of the token is stored in the database; plaintext session tokens cannot be compromised in a database leak.
   - Session cookie is set with `HttpOnly`, `SameSite=Lax`, and `Secure` flags.
   - Session creation, validation, last seen update (`touch`), deletion, and expired cleanup contracts are tested with `tests/contract/auth_contract_test.go` on SQLite and PostgreSQL.

3. **CSRF Protection:**
   - The `X-CSRF-Token` header or `_csrf` form parameter is enforced on state-changing HTTP methods (`POST`, `PUT`, `PATCH`, `DELETE`).
   - Requests with invalid or missing CSRF tokens are blocked with 403 Forbidden.

4. **Strict Security Headers & Dynamic CSP Nonce:**
   - A cryptographic CSP nonce is generated for every HTTP request and injected into the context.
   - The `Content-Security-Policy: default-src 'self'; script-src 'self' 'nonce-...'; style-src 'self' 'nonce-...'; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; form-action 'self'` header is applied.
   - `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Referrer-Policy: strict-origin-when-cross-origin` headers are enforced.

5. **CLI Bootstrap:**
   - Initial admin account creation via `hydro create-admin --email ... --password-stdin` is validated at the CLI level; the password is not written to the command line.

## Executed Commands and Test Output

```text
go test -v ./internal/auth/...
go test -v ./internal/security/...
go test -v ./internal/server/...
go test -v ./tests/contract/...
go test -v ./cmd/hydro/...
```
All tests resulted in success (PASS).
