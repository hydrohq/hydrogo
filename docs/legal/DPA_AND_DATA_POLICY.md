# Data Processing Agreement (DPA) & Privacy Policy Framework

> **Compliance Targets:** KVKK (6698 Sayılı Kanun) & GDPR (EU 2016/679)  
> **Applicability:** Single-Tenant Customer Deployments & Managed Services

---

## 1. Roles & Data Controller / Processor Boundary

- **Customer:** Data Controller (*Veri Sorumlusu*). Determines the legal basis, purposes of processing, and user categories.
- **HydroGo Platform / Managed Provider:** Data Processor (*Veri İşleyen*). Processes data strictly on documented customer instructions and technical contracts.

---

## 2. Personal Data Categories Processed

1. **System Administrators & Editors:**
   - Identity: Name, email address, password hash (Argon2id), encrypted MFA secret.
   - Access Logs: IP address (truncated/anonymized in logs after audit window), User-Agent, audit event timestamps.
2. **End-Users & Form Inquirers:**
   - Contact form submissions: Name, email, phone (if enabled), message content.
   - Member / Commerce accounts (if enabled): Email, shipping/billing address, transaction history (no raw credit card numbers stored; delegated to tokenized payment gateways).
3. **Session & Cookie Data:**
   - Strict HTTP-only, SameSite cookies for authenticated sessions.
   - No cross-site tracking or third-party behavioral tracking cookies in Core.

---

## 3. Data Subject Rights & Erasure Protocol (*Unutulma Hakkı*)

### Append-Only Erasure Intent Log (ADR-0014)
- Deletion requests are recorded in an external, cryptographic hash-chained erasure log.
- Cascading deletion permanently overwrites/removes user PII from:
  1. Active PostgreSQL tables (`users`, `sessions`, `form_submissions`, `member_profiles`).
  2. Distributed Redis cache (all active tokens and cached session keys).
  3. S3 object storage (any user-uploaded personal files).
- Backup snapshots retain immutable point-in-time archives until their maximum retention threshold (30 days), after which they are irreversibly destroyed by storage lifecycle policies.
- Upon restore from an older backup, HydroGo's replay mechanism re-applies post-backup deletion intents from the cryptographic erasure log to prevent "zombie PII" revival.

---

## 4. Technical & Organizational Security Measures (TOMs)

- **Encryption in Transit:** Strict TLS 1.3 with HSTS (`max-age=63072000; includeSubDomains; preload`).
- **Encryption at Rest:** 
  - Database disk volume encrypted with AES-256 (LUKS / AWS EBS KMS / GCP CMEK).
  - Object storage server-side encryption (SSE-S3 / SSE-KMS).
  - Sensitive application columns (MFA secrets, API tokens, webhook signing keys) encrypted at application layer using AES-256-GCM with versioned keys.
- **Access Control:** Multi-Factor Authentication (TOTP) enforced for all admin accounts; principle of least privilege.
- **Audit Logging:** Tamper-evident, structured audit logs for every state-altering administrative action.
- **Subprocessors:** Customer must be notified at least 30 days prior to introducing or replacing any third-party infrastructure subprocessor (e.g., cloud hosting, transactional email provider).
