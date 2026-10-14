# R49 — Generated client timeout parity

**Date:** 2026-09-17  
**Status:** Successful (local generator contract)

Generated `HydroClient` now validates `timeoutMs` within the same fail-closed `1..120000` ms range as the starter. Invalid values are not silently clamped; they throw a visible `RangeError` during the constructor phase. Valid timeout values apply directly to the request controller.
