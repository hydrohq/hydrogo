# ADR-0301: Curated Page Layout Template Registry

- Status: Accepted
- Date: 2026-09-29

## Context

Editors benefit from reusable block scaffolds when creating pages. Editions also
need a typed way to provide additional generic starting layouts.

## Decision

The pages module provides a built-in registry for a blank canvas, service
showcase, landing page, and case study. Client Editions can append typed
`pages.LayoutTemplate` values to the page handler. Template IDs are unique,
bounded slugs; all supplied blocks pass the existing page block validator
before the registry is rendered or used to create a draft. The Admin Studio
offers a typed Design System radio-card picker. Choosing a template creates a
draft with its blocks already initialized.

Templates remain starting content only: they do not execute migrations,
activate modules, or add routes and workers. No storage schema changes are
required.

## Compatibility and operations

The registry is additive and uses the existing pages block and draft contracts.
SQLite, PostgreSQL, and MySQL behavior is unchanged. Removing an Edition
template does not modify pages that were already created from it.
