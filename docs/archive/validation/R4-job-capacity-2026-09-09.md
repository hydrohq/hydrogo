# R4 durable job queue capacity validation

## Scope

The `MaxEntries` control of the durable SQL queue is now serialized by updating the singleton capacity row within the producer transaction. A row lock in PostgreSQL and writer serialization in SQLite preserve the same capacity control. The new `202609090020_job_capacity_lock` migration creates and idempotently seeds the same lock row for each dialect.

## Evidence

- Commit: `c8f4104`
- Environment: Windows development environment
- Commands:
  - `go test ./internal/jobs/sqlstore -count=5`
  - `go test ./... -p 1`
  - `go vet ./...`
  - `git diff --check`
- Result: all local tests and static checks succeeded.
- Contract: `TestSQLiteQueueCapacityIsBoundedForConcurrentProducers` validates that with two concurrent producers and capacity set to `1`, only one job can be committed and the claimable job count does not exceed the limit.

## Limits

Since `HYDRO_TEST_POSTGRES_URL` is not defined in this run, the PostgreSQL concurrency contract was not executed locally; it must be run by the CI/production gate. This evidence also does not certify multiple customer hardware or external broker capacity.
