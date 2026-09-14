# ADR-0082: Bounded list query contract

## Status

Accepted

## Context

Public and Studio list endpoints are bounded, but a later detail lookup inside
an item loop could silently reintroduce N+1 database work.

## Decision

- The pages and news SQL repositories must satisfy their list projection with a
  single bounded `QueryContext` call.
- Local SQLite fixtures count query calls while exercising the real repository
  and migration schema. The fixture covers both a public page list and a news
  list surface.
- PostgreSQL planner plans and latency under customer-sized data remain an
  external capacity gate.

## Consequences

List projections cannot grow hidden per-row database work without failing a
local regression. Detail enrichment must be explicit and separately budgeted.

## Verification

See `modules/pages/repository/sqlstore/query_budget_test.go`,
`modules/news/repository/sqlstore/query_budget_test.go`, and
`docs/validation/R5-bounded-list-queries.md`.
