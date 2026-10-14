# R40 — Compose port binding security lint

**Date:** 2026-09-17  
**Status:** Successful (local)

`scripts/compose-security-lint.ps1` verifies expected loopback bindings in local Compose fixtures and rejects PostgreSQL, Redis, or MinIO ports exposed via bare, `0.0.0.0`, or non-loopback host bindings to the LAN fail-closed. The script executes directly after container hardening checks in the `scripts/verify-local.ps1` workflow.

This check does not replace production firewall/ingress policies; it maintains secure defaults for the development stack within the repository.
