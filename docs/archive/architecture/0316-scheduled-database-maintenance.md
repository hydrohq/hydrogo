# ADR-0316: Scheduled database maintenance

## Status

Accepted

## Decision

Persistent HydroGo server processes schedule one bounded database maintenance
run each Sunday at 03:00 UTC. CGI mode does not start this worker. SQLite uses
`VACUUM` from a separate, short-lived connection pool so the request pool stays
available; WAL readers retain their snapshots, and a busy post-vacuum
checkpoint is deferred safely. The operation reports the reclaimed database
and WAL footprint. PostgreSQL runs `ANALYZE` outside a transaction to refresh
planner statistics. MySQL is not changed by this capability.

Successful runs emit structured system logs and, when the Pages module's
audit table is installed, append a maintenance audit event (one scoped event
per configured tenant in multi-tenant deployments).
Failures are logged without exposing database diagnostics to public requests.
The worker has no schema, cache, or durable job dependency. It uses the
existing profile-specific DB pool limits only while a scheduled run executes;
requests continue on the normal pool. SQLite retains the single-node
deployment constraint from ADR-0006; PostgreSQL's operation is safe to repeat
across server instances.

No migration is required. The operation is compatible with old binaries and
is cancelled with the server context. SQLite maintenance uses a temporary
database-sized working copy, so operators should reserve corresponding free
disk capacity. The database file and WAL are measured together before and
after vacuum; a reader-held WAL checkpoint may postpone reclamation until a
later run.

## Verification

SQLite tests hold a read transaction across vacuum, verify it remains usable,
and confirm a later run reclaims space. Schedule tests cover the weekly UTC
boundary. PostgreSQL executes the same maintenance operation when
`HYDRO_TEST_POSTGRES_URL` is configured.
