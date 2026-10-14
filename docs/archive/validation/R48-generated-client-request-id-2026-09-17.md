# R48 — Generated client request ID parity

**Date:** 2026-09-17  
**Status:** Successful (local generator contract)

Generated `HydroAPIError` now carries a bounded `requestId` field. Precedence favors `request_id` within the JSON body, falling back to the response `x-request-id` header. Oversize response errors retain the same header correlation. This behavior aligns with the typed error contract in headless starter and provides support/observability correlation without leaking raw infrastructure details to users.
