# R62 — Product scope contracts (2026-09-17)

Five previously open product decisions now have explicit, provider-neutral boundaries: OIDC, mobile API, multi-tenant, AI, and Distributed. These documents intentionally do not claim external provider or production certification.

| Contract | ADR | Current state | Required before product claim |
|---|---|---|---|
| OIDC | 0138 | Core boundary defined | Provider/key-rotation/replay and auth DB tests |
| Mobile API | 0139 | Version/token boundary defined | Threat model, revoke, abuse and SDK contract |
| Multi-tenant | 0140 | Opt-in isolation rules defined | IDOR/property/restore/capacity certification |
| AI | 0141 | Opt-in data boundary defined | Provider DPA, redaction, injection and cost tests |
| Distributed | 0142 | Topology/certification gate defined | PostgreSQL + Redis/Valkey + S3/CDN + soak drill |

The remaining items are operator/provider gates, not hidden implementation debt. They require credentials, a staging environment, traffic data, or legal/customer decisions and must be recorded with evidence when executed.
