# R39 — Local Compose services restricted to loopback bindings

**Date:** 2026-09-17  
**Status:** Successful (static configuration verification)

MinIO (API and console) and Redis ports in `deploy/local/docker-compose.storage.yml` are now published only on `127.0.0.1`. The main `compose.dev.yaml` already used the same loopback boundary for PostgreSQL and Redis. This ensures the local development stack does not accidentally expose database, cache, or MinIO management ports to devices on the same local network.

This change does not constitute proof of production network policy or real firewall/ingress rules; it is a secure default for local Compose fixtures. Inter-container access continues over the Compose network.

Verification: YAML diff check and existing Go/local verification hooks passed.
