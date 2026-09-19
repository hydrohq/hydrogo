# R69 member order detail and payment status

**Date:** 2026-09-19  
**Status:** SQLite contract and API compatibility checks passed; provider
sandbox/webhook delivery remains open.

The consumer surface now exposes `GET /api/v1/mobile/member/orders/{orderID}`.
The repository query scopes by both order ID and authenticated member ID, and
the projection excludes customer identity/contact fields and payment-provider
secrets. The TypeScript client and OpenAPI contract are updated; webhook
verification remains server-side and is never exposed to the mobile app.

```text
go test ./modules/commerce ./modules/commerce/repository/sqlstore ./modules/members/transport
go run ./tools/openapidiff --baseline docs/api/baselines/member-commerce-v1.openapi.json --current docs/api/member-commerce-v1.openapi.json
```
