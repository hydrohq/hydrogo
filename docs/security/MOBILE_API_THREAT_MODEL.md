# Mobile API Threat Model & Security Controls (ADR-0139)

> **Standard:** STRIDE & OWASP Mobile Top 10  
> **Applicable Surface:** `/api/v1/mobile/*`

---

## 1. Threat Matrix & Mitigations

| Threat Category | Attack Vector | HydroGo Mitigation |
| :--- | :--- | :--- |
| **Spoofing & Impersonation** | Stolen / Intercepted Refresh Token | **Single-Use Token Rotation & Token Family Invalidation:** Any attempt to use a previously rotated refresh token immediately revokes all sessions in that family. |
| **Tampering** | Man-in-the-Middle (MitM) & Parameter Modification | Mandatory TLS 1.3, pinned HTTPS certificates, HMAC-SHA256 hashed storage of tokens in DB. |
| **Repudiation & Device Tracking** | Rogue device actions without audit trail | Every token issuance registers device ID, device model, platform (iOS/Android), IP, and timestamp in tamper-evident database. |
| **Information Disclosure** | Token leakage into access logs, metrics, or cache | Raw tokens are stripped/redacted from all logging formatters; only opaque token hashes (`sha256(token)`) are used for internal lookup. |
| **Denial of Service & Brute-Force** | Credential stuffing on login/token refresh | IP and device-rate limiting with exponential backoff; maximum 5 failed attempts per 15 minutes. |
| **Elevation of Privilege** | Session persistence after password reset or admin ban | All mobile device sessions are cascadingly tied to `users(id)`; user deactivation or password reset immediately invalidates all active tokens. |

---

## 2. Device Revocation Workflows

1. **User-Initiated Device Logout (`/api/v1/mobile/auth/revoke`):**
   - Marks target `device_id` as `revoked_at = NOW()`.
   - Access token and refresh token rejected immediately.
2. **Global Account Logout (`/api/v1/mobile/auth/revoke-all`):**
   - Marks all active device sessions for `user_id` as revoked.
   - Requires re-authentication across all mobile and tablet clients.
3. **Suspicious Reuse Alert (Automated Defense):**
   - If a duplicate refresh token request is detected, the entire `family_id` is revoked and an audit alert `ALERT_SUSPICIOUS_TOKEN_REUSE` is dispatched.
