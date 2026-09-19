# R70 documentation reconciliation

**Date:** 2026-09-19  
**Status:** Current status documents reconciled with the implemented mobile and
release-gate boundaries; production certification remains explicitly open.

The README, Project Status, Implementation Status and Roadmap now describe both
audiences accurately: Admin Studio/staff mobile and the separate consumer
member mobile API. The documents distinguish local contract completion from
real threat/device/provider, proxy, restore, capacity and multi-tenant gates.
The member order-detail/payment-status projection is linked to its OpenAPI and
SQLite ownership evidence; no provider webhook or production-ready claim was
added.

Verification:

```text
rg -n "not a public member mobile API|custom mobile API if" README.md docs
go test ./... -p 1 -count=1 -timeout=5m
```

The first search is expected to return no stale current-status claims; dated
historical reports may retain their original wording and date.
