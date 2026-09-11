# R1 Toolchain, Cross Compilation and Backup Tools Validation

**Date:** 2026-09-06  
**Roadmap item:** R1 Toolchain and Portability  
**Status:** Partial; cross-compilation and encoder selection validated, actual backup/restore execution required separately.

## Validated Behaviors

1. **Cross-Platform Compilation:**
   - The project is configured entirely with pure Go (CGO-free) dependencies:
     - SQLite: `modernc.org/sqlite`
     - PostgreSQL: `github.com/jackc/pgx/v5`
     - Cryptography & Password: `golang.org/x/crypto/argon2`
     - Image: Go standard library (`image/png`, `image/jpeg`, `image/gif`) + `golang.org/x/image/webp`
   - Both Windows amd64 (`hydro.exe`) and Linux amd64 (`bin/hydro-linux-amd64`) targets were successfully compiled without requiring any external C libraries:
     ```powershell
     $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/hydro-linux-amd64 ./cmd/hydro
     $env:GOOS="windows"; $env:GOARCH="amd64"; go build -o bin/hydro.exe ./cmd/hydro
     ```

2. **Backup and Recovery Tools Evaluation:**
   - **SQLite (Compact Profile):** Independent operation can be validated with CGO-free `VACUUM INTO` and SQLite online backup logic, or immediate file copying after a WAL checkpoint.
   - **PostgreSQL (Standard/Distributed):** Compatible with standard `pg_dump` / `pg_restore` CLI tools and S09 logical data export contracts.

3. **Image Processing / Encoder Decision:**
   - Pure Go libraries were chosen because CGO and external C dependencies like `libvips` compromise the single-binary distribution and portability principle (Manifesto Article 2 & 5).
