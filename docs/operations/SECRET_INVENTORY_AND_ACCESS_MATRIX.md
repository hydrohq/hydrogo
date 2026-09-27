# Secret Inventory & Access Control Matrix

> **Environment Scope:** Production & Staging  
> **Security Requirement:** Zero raw secrets in repositories, unencrypted backups, or application logs.

---

## 1. Secret Taxonomy & Rotation Policy

| Secret Name | Purpose | Format | Storage / Injection Method | Rotation Frequency | Rotation Procedure |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `HYDRO_SESSION_KEY` | Signs session cookies and auth tokens | 64-char Hex (32 bytes entropy) | HashiCorp Vault / AWS Secrets Manager -> Mounted file | 90 days | Dual-key verification window (old key accepts reads, new key signs writes) |
| `HYDRO_ENCRYPTION_KEY` | Encrypts column-level secrets (MFA, keys) | 64-char Hex (AES-256-GCM) | Key Escrow / Cloud KMS | 180 days | Key version header re-encryption background job |
| `HYDRO_DATABASE_URL` | PostgreSQL application user credentials | URI with credentials | Docker / K8s secret volume | 90 days | Rotate password in DB, update secret, restart app pods |
| `HYDRO_REDIS_URL` | Valkey / Redis cluster authentication | URI with password | Docker / K8s secret volume | 90 days | Multi-pass AUTH update without cache eviction |
| `HYDRO_S3_SECRET_KEY` | Media asset bucket IAM access | AWS Secret Key string | IAM Role / Assumed Role (IRSA) | 90 days | IAM credential rotation |
| `HYDRO_MAILER_SMTP_PASS`| Transactional email delivery | API key token | Secrets Manager | 180 days | API key regenerated on provider |
| `HYDRO_FORMS_TURNSTILE_SECRET_KEY` | Verifies optional public-form challenges | Provider secret | Secrets Manager | Provider policy | Rotate alongside site key and redeploy both values together |
| `HYDRO_METRICS_TOKEN` | Prometheus metrics scrape authentication | High-entropy Bearer token | Secret Manager | 180 days | Updated in Prometheus scrape config + App |

---

## 2. Access Control & IAM Role Matrix

| Role | Database Access | Redis Access | S3 / Media Bucket | Secrets Manager | Admin Studio UI |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **HydroGo App Process** | `hydro_app` (DML only, no DDL at runtime) | Read/Write (Namespaced DB 0) | Read/Write (`/media/*` prefix) | Read-only to mounted secrets | N/A (Runs binary) |
| **CI/CD Pipeline** | Migration runner (DDL permissions during release) | None | Asset upload (`/assets/*`) | Ephemeral deploy credentials | None |
| **System Administrator (DevOps)** | Read-only replica via bastion (MFA required) | Read-only replica (CLI via SSH) | Read-only bucket audit | Admin access with break-glass log | Super-admin (MFA enforced) |
| **Content Editor** | No direct access | No direct access | Via presigned API upload only | No access | Role-restricted workbench |

---

## 3. Secret Leak Prevention Checklist

1. **Static Analysis & Guardrails:** `tools/actionguard` and pre-commit hooks verify that no `.env`, `.pem`, `.key`, or secret files are committed to version control.
2. **Log Redaction:** All structured JSON log formatters filter keys containing `password`, `secret`, `token`, `key`, `authorization`, `cookie`, `session`.
3. **Diagnostics & Error Boundaries:** Application panics and HTTP 500 error responses emit opaque correlation IDs (`request_id`) and never dump stack traces or SQL connection strings to client endpoints.
