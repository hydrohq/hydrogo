# Independent Security Review & Pentest Verification Checklist

> **Target Standard:** OWASP Top 10 & ASVS Level 2  
> **Applicability:** Production Go-Live Gate (Phase 2 Certification)

---

## 1. Authentication & Session Security

- [x] **Password Storage:** Argon2id with cryptographically safe salt and tuned memory/time parameters.
- [x] **MFA Enforcement:** TOTP (RFC 6238) with encrypted secret keys (AES-256-GCM) and single-use backup recovery codes.
- [x] **Brute-Force Protection:** Rate limiting on `/login`, `/mfa`, and password reset endpoints.
- [x] **Session Handling:** Cryptographically secure 256-bit session tokens; strict `HttpOnly`, `Secure`, `SameSite=Lax/Strict` cookie attributes.
- [x] **Session Revocation:** Instant invalidation on password change, role downgrade, or administrative deactivation.
- [x] **Token Family Invalidation:** Single-use rotation for mobile refresh tokens; immediate family revocation upon reuse detection.

---

## 2. Authorization & Access Control (No IDOR)

- [x] **Strict Domain Boundary:** Every handler passes actor identity to the application service layer.
- [x] **Record-Level Authorization:** Object IDs are validated against tenant/owner scope; direct manipulation of IDs yields HTTP 403/404 fail-closed.
- [x] **Privilege Separation:** Role-Based Access Control (Admin, Editor, Member, Auditor) enforced deterministically at service boundaries.

---

## 3. Injection, Escaping & Content Security

- [x] **SQL Injection:** 100% parameterized SQL queries; string concatenation in queries is strictly prevented by lint rules and contract suites.
- [x] **XSS Prevention:** Type-safe Templ templating engine auto-escapes all dynamic variables; `templ.Raw` usage requires explicit security audit.
- [x] **Content-Security-Policy (CSP):** Strict CSP headers with cryptographic nonces for inline scripts/styles; `eval()` and inline unsafe code forbidden.
- [x] **MIME Sniffing & Framing:** `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`.

---

## 4. Media & Upload Security

- [x] **Content-Type & Magic Byte Validation:** File payloads inspected for true magic numbers; filename extensions alone are never trusted.
- [x] **File Execution Prevention:** Uploaded files stored outside executable paths, delivered with `Content-Disposition: attachment` or immutable CDN headers.
- [x] **SVG & Image Sanitization:** Strict XML/SVG parsing and tag stripping to prevent Stored XSS via graphic vectors.

---

## 5. Network & API Boundary Protection

- [x] **CORS Policy:** Strict origin allowlists; credentialed cross-origin requests (`Access-Control-Allow-Credentials: true` with wildcard `*`) forbidden.
- [x] **Idempotency Safeguard:** Idempotency keys required on financial, payment, and order state mutations.
- [x] **Sensitive Data Redaction:** Raw tokens, passwords, and private encryption keys are automatically scrubbed from JSON logs, metrics, and trace spans.
