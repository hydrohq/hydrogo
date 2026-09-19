# Security Diff Report

**Branch:** `main`  
**Base:** `e9dfcdc`  
**Head:** `742f34a`  
**Date:** 2026-09-19  
**Scope:** Last commit only; source-only incremental review

## Summary

| Category | New | Existing | Total |
| --- | ---: | ---: | ---: |
| Critical | 0 | 0 | 0 |
| High | 0 | 0 | 0 |
| Medium | 0 | 0 | 0 |
| Low | 0 | 0 | 0 |

## Verdict

**PASS — no new Critical or High findings.**

The changed API adds a member-owned order detail. The bearer principal is
validated before dispatch, the repository method requires both member and order
identifiers, and a mismatched owner returns `sql.ErrNoRows` which becomes the
same generic 404 as a missing order. The response uses the existing customer
projection and does not expose checkout contact fields, provider secrets, or
webhook payloads. SQLite tests cover both an authorized lookup and a cross-member
lookup; the full Go test/vet suite and OpenAPI compatibility preflight passed.

The TypeScript client sends `credentials: "omit"` and an explicit bearer only;
the bounded order ID prevents an unbounded path input. No dependency or runtime
configuration changed in this commit.

## Coverage and verification

- Go authorization/data-flow review: `modules/members/transport/http.go`,
  `modules/commerce/customerorders.go`, and
  `modules/commerce/repository/sqlstore/store.go`.
- API contract review: `docs/api/member-commerce-v1.openapi.json` and
  `go run ./tools/openapidiff --baseline docs/api/baselines/member-commerce-v1.openapi.json --current docs/api/member-commerce-v1.openapi.json`.
- Tests: `go test ./... -p 1 -count=1 -timeout=5m`, `go vet ./...`, and
  `npm test` in `examples/headless-starter`.
- Independent challenge: attempted IDOR with another member ID; the scoped
  repository contract rejects it before projection.

## Needs validation

- PostgreSQL execution of the new ownership query requires the configured
  `HYDRO_TEST_POSTGRES_URL` contract environment.
- Provider sandbox payment confirmation and webhook delivery remain external
  gates; this client-facing status projection does not certify them.
- Reverse-proxy/BFF and real device-loss/browser E2E remain outside diff scope.

## Coverage limits

This is an incremental report, not a whole-repository audit. Untouched routes,
dependencies, deployment/IaC, provider credentials, live services, and existing
findings are not reclassified by this report. The prior whole-repository source
audit remains in `security-report/SECURITY-REPORT.md`.
