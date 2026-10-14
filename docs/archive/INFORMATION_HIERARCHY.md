# HydroGo Information Hierarchy & Respect for the User (Audience Separation)

> **"Respect for the user is not just delivering fast responses; it is shielding them from cognitive clutter, technical noise, and internal architecture they do not need and should not have to understand."**

Every user interface, notification, error message, log line, and API response across HydroGo is designed for a specific target **audience**. Technical noise and internal architectural mechanisms must never leak to end users or business operators.

---

## 🏛️ The 5 Information Tiers

Information across the platform is partitioned into five distinct layers:

```text
+-------------------------------------------------------------+
│ 1. MEMBER & PUBLIC CUSTOMER                                 │  ➔ Clean, courteous, actionable, zero jargon
+-------------------------------------------------------------+
│ 2. BUSINESS OPERATOR & EDITOR (Admin Studio)                │  ➔ Content, orders, catalog, leads, business KPIs
+-------------------------------------------------------------+
│ 3. SYSTEM ADMINISTRATOR & OPERATOR (Sysadmin & SRE)         │  ➔ Host health, DB pool, backups, drift, worker queues
+-------------------------------------------------------------+
│ 4. DEVELOPER & INTEGRATOR (API Consumer)                    │  ➔ API schemas, webhook payloads, events, client SDKs
+-------------------------------------------------------------+
│ 5. AUDITOR & COMPLIANCE OFFICER (Security & DPO)            │  ➔ Immutable audit logs, data erasure, security events
+-------------------------------------------------------------+
```

---

### 1. Member & Public Customer Tier

**Objective:** Enable users to accomplish tasks with minimal friction and maximum confidence.  
**Tone:** Concise, helpful, polite, and solution-focused.

- ✅ **PERMITTED INFORMATION:**
  - Order status, delivery tracking, receipt download links.
  - Account profile details, registration date, membership badges.
  - Clear security summaries ("Secure", "2 Active Devices", "Last login: Today at 14:30").
  - Human-friendly validation guidance ("Incorrect email or password", "This email is already registered", "Please check your input").
  - Clear navigation verbs ("Go to Cart", "Sign Out", "Reset Password").

- ❌ **STRICTLY PROHIBITED (Technical Leakage):**
  - Cryptographic algorithms and library names (e.g., `Argon2id`, `AES-256-GCM`, `Ed25519`, `bcrypt`, `crypto/subtle`).
  - Database engine details (e.g., `SQLite`, `PostgreSQL`, `pgx`, `database/sql`, raw SQL queries, internal table/column names).
  - Runtime exceptions and stack traces (e.g., Go panics, stack traces, `nil pointer dereference`, filesystem paths like `C:\dev\...` or `/var/www/...`).
  - Internal system identifiers and hashes (e.g., session token hashes, salts, internal worker IDs).
  - Raw HTTP / infrastructure telemetry (e.g., response time in ms, pool saturation, memory allocations).

---

### 2. Business Operator & Editor Tier (Admin Studio User)

**Objective:** Facilitate day-to-day editorial, marketing, and business workflows.  
**Surfaces:** `/admin`, `/admin/news`, `/admin/pages`, `/admin/commerce/*`, `/admin/members`.

- ✅ **PERMITTED INFORMATION:**
  - Article drafts, publication schedules, categories, tags, and revisions.
  - Product stock levels, variant prices, order values, customer contact information.
  - Sales charts, form submission counts, newly registered members.
  - Rich Text formatting controls, media upload pickers, and visual image previews.

- ❌ **PROHIBITED:**
  - Raw CPU/RAM hardware utilization counters.
  - Outbox queue retry exponential backoff formulas or raw SQL migration diffs.
  - Sysadmin-level restore fence locks and storage driver parameters.

---

### 3. System Administrator & Operator Tier (Sysadmin & SRE)

**Objective:** Ensure high availability, zero-downtime deployments, and verified disaster recovery.  
**Surfaces:** CLI (`hydro doctor`, `hydro backup`, `hydro release`), `/admin/control`, `/admin/operations`.

- ✅ **PERMITTED INFORMATION:**
  - Database connection pool health (active / idle connections, max pool limit).
  - Blue/Green deployment state, write fences, and rollback readiness.
  - Disk capacity utilization, WAL file sizes, backup integrity hashes (SHA-256).
  - Dead-letter queue depth, outbox retry counters, and background worker status.
  - Rate limiting telemetry, process memory usage, and operational status.

---

### 4. Developer & Integrator Tier (API Consumer)

**Objective:** Support reliable third-party API, webhook, and custom module integrations.  
**Surfaces:** `/admin/developer`, `/api/v1/*`, GraphQL schema, Webhook delivery logs.

- ✅ **PERMITTED INFORMATION:**
  - REST and GraphQL OpenAPI v3 schema definitions.
  - Webhook delivery outcomes, HTTP status codes (200, 429, 500), and transmitted/received JSON payloads.
  - Masked API access tokens, permission scopes, and rate limit headers.
  - Strongly typed client SDK contracts.

---

### 5. Auditor & Compliance Officer Tier (Security & DPO)

**Objective:** Verify compliance with data protection regulations (GDPR/KVKK), security standards, and governance policies.  
**Surfaces:** `/admin/security`, `/admin/permissions`, Audit Trail logs.

- ✅ **PERMITTED INFORMATION:**
  - Append-only, tamper-evident audit logs: who altered or removed what resource and when.
  - Data erasure and anonymization certificates/receipts.
  - Security incident logs: failed authentication attempts, rate-limit bans, and anomalous IP activity.
  - Role-based access control (RBAC) assignments and permission override records.

---

## 🛠️ Software Engineering & UI Design Principles

1. **Fail-Closed & Leak Prevention:** Error responses must never expose internal errors (e.g. `sql syntax error at line 42`). Show end users a graceful message (`"Unable to complete this action right now. Please try again later."`), while logging the structured error to internal observability logs (`logger.Error`).
2. **Audience-Aware Vocabulary:** Cryptographic and database jargon belongs only in administrative and developer tools. Customer screens use plain language: `"Encrypted Security"` instead of `"Argon2id"`, `"Please sign in"` instead of `"HTTP 401 Unauthorized"`.
3. **Minimize Cognitive Load:** Before adding any data point or text to an end-user screen, evaluate: *"Does this information help the user take a meaningful action?"* If not, eliminate it from the user-facing interface.
