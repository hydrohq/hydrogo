# R59 — Risk-weighted coverage floors

Date: 2026-09-17

CI now runs `scripts/risk-coverage.sh` for security- and payment-sensitive
packages instead of relying on a repository-wide average. The command creates
temporary coverage profiles, checks package-specific floors, and removes them
on exit. Current explicit floors are:

| Package | Floor |
| --- | ---: |
| `internal/auth/sqlstore` | 4% |
| `modules/members/transport` | 45% |
| `modules/commerce/providers` | 43% |
| `modules/api/graphql` | 65% |

The auth-store floor is a baseline while its driver matrix is expanded; each
floor is intended to be ratcheted upward with new tests.

`bash scripts/risk-coverage.sh` passed on 2026-09-17. Broader branch coverage
and real provider/database matrices remain separate production follow-up work.
