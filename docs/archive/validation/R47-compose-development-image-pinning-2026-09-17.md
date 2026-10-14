# R47 — Main development Compose image pinning

**Date:** 2026-09-17  
**Status:** Successful (static configuration verification)

The primary development stack was also pinned to exact versions:

- `postgres:18.6`
- `redis:8.10.1-alpine`

Compose security lint now rejects `latest`, Redis major/major-minor, and PostgreSQL major-only image tags; new services lacking patch/release tags fail local verification. Registry digest/provenance and CVE scanning remain release pipeline gates.
