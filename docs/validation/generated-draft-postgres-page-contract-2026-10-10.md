# Generated draft PostgreSQL page contract — 2026-10-10

Added an environment-gated contract test for the E2E fixture's PostgreSQL
draft adapter. With `HYDRO_TEST_POSTGRES_URL`, it creates an isolated schema,
resets seed state, verifies deterministic `updated_at DESC, resource_id DESC`
ordering across a 25-row page plus lookahead and page two, and checks locale
filtering. Schema cleanup runs through `t.Cleanup`. The shared page fixture
timestamp is now derived from the current time after reset data, avoiding a
hard-coded date that would eventually change test ordering.

Checks:

- The test package compiled and `go vet` passed.
- The test was skipped because `HYDRO_TEST_POSTGRES_URL` is unset. `pg_isready`
  found no server. A throwaway PostgreSQL 18.6 cluster initialized successfully,
  but `pg_ctl` could not bind its Unix socket (`Operation not permitted`); Docker
  daemon access was denied. PostgreSQL behavior therefore remains unverified
  in this environment.
- The separate SQLite generated-handler pagination contract passed in the
  prior validation report, [`generated-draft-inbox-pagination-2026-10-10.md`](generated-draft-inbox-pagination-2026-10-10.md).

This adds an executable PostgreSQL gate but does not count as a PostgreSQL
certification result until the test actually runs against a server.
