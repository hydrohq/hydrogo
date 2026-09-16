# ADR-0119: Bound frontend manifest inputs

- **Status:** Accepted
- **Date:** 2026-09-16

The standalone frontend manifest validator and project generator reject files
larger than 1 MiB, non-regular files, and route names/paths/test IDs exceeding
their contract limits. This keeps metadata-driven E2E and starter generation
safe on untrusted build output while preserving deterministic normalization.
