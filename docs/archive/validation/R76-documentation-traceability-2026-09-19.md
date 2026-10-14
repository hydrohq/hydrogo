# R76 — Documentation and scope reconciliation (2026-09-19)

## Scope

The product-facing documentation was checked against the current source,
release manifest and accepted ADR index. Unsupported claims were removed or
qualified as pending external evidence.

| Surface | Current truthful boundary | Evidence |
| --- | --- | --- |
| `README.md` | HydroGo is not declared a fully certified commercial v1; profiles and remaining gates are explicit. | README status and profile sections |
| `docs/PROJECT_STATUS.md` | Staff and consumer/member mobile APIs exist; production certification remains open. | Project status mobile and certification sections |
| `docs/ROADMAP.md` | Local contracts are distinguished from real proxy, restore, provider and capacity gates. | S05–S09 and distributed sections |
| `docs/SAAS_CMS_READINESS_PLAN.md` | Single-customer managed deployment is the current boundary; shared tenancy is Phase 6 and opt-in. | Product boundary and Phase 6 |
| `docs/architecture/README.md` | ADR statuses match the implemented boundary, including scoped BFF policy. | ADR-0147 through ADR-0151 |
| `docs/validation/release-gate-manifest.json` | Local development is ready; managed Standard remains candidate pending external gates. | Manifest profiles and gate statuses |

## Remaining issue classification

- **Environment:** PostgreSQL clean restore, reverse-proxy blue/green,
  external backup/key recovery, production key ceremony, capacity/RPO/RTO.
- **Code/product:** deployable BFF session store/login exchange, staging
  browser E2E, provider sandbox certification, and future shared-tenancy
  implementation.
- **Legal/operations:** independent penetration test, DPA/support/SLA and
  commercial pilot evidence.

The reconciliation does not turn any pending item into a production claim.
