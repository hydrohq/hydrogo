# 0008: Business-application capabilities in core

- Status: Proposed
- Date: 2026-10-21

## Context

ADR 0001 made HydroGo a single-binary CMS and ADR 0007 limited extension to YAML content types, API keys and webhooks, and a frontend on the API. HydroGo is now also meant to carry small business applications (a CRM first, then an ERP-lite) built from presets, without forking core. Those need things a CMS rarely does: atomic multi-record writes, server-side rules, numbering and computed fields, state machines, roles and row-level access, aggregation, and events other services can trust.

## Decision

1. Core gains **generic** capabilities that more than one domain (CMS, CRM, ERP, helpdesk) wants. The test for a change is "would a second domain use this?", not "does a blog need it?".
2. Domain logic stays out of core: accounting rules, tax, stock costing, e-invoicing, and any customer-specific behaviour live in preset YAML and in services next to the project (rule 2 of `AGENTS.md` still holds).
3. Capabilities are declared in the content-type YAML. There are no Go plugins. Logic that needs code runs in a side service that talks to core through the API, signed webhooks and synchronous hooks (fail-closed by default, allowlisted hosts, HMAC-signed).
4. A capability that is not used in a project's YAML changes nothing for it (no routes, tables or work), as with modules.
5. Rule 12 stays: nothing here may require a rebuild on publish.
6. Presets (`crm`, `erp-lite`) choose modules and stock content types; their frontends are separate starters.
7. Rules, computed fields and row-level access use one small, side-effect-free expression language. Proposed: CEL (typed, bounded cost, no I/O). The choice is part of the first PR that needs it.
8. PostgreSQL is the recommended database for ERP-lite; every capability still works on SQLite, PostgreSQL and MySQL and is tested on all three.

Rough order: foundations and queries first, then the CRM preset, then data integrity (batch, numbering, rules, state machines), permissions and reports, side-service support, and the ERP-lite preset. The detailed plan is tracked outside the repository (rule 7).

## Consequences

- The scope of "CMS" widens on purpose; this record is the boundary. Anything that fails the second-domain test is declined or moved to a starter.
- Each capability adds a three-dialect test burden and a threat model entry (SSRF in hooks, row-level access on every read path, expression cost limits).
- ADR 0007's extension list gains synchronous hooks and service accounts once they exist.
