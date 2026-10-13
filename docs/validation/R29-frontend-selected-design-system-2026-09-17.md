# R29 — Frontend Starter Selected Design-System Metadata

**Date:** 2026-09-17  
**Status:** Success (local contract)

## Purpose

To verify that the `hydro project new --design-system ...` selection is propagated
into the generated frontend starter manifest. The previous manifest only wrote the supported
adapter list without conveying the selected adapter to the consumer.

## Changes

`frontend/starter.json` now carries two distinct fields:

- `design_system`: the default provider selected for this project;
- `design_systems`: the list of provider-agnostic adapters that the Client Edition
  can choose (`native`, `shadcn`, `material`, `carbon`, `headless`, `vanilla`).

Core does not import any provider runtime; these fields represent manifest
metadata only. The same selection as the `design_system` value in the existing theme
manifest is used.

## Validation

Command:

```powershell
go test ./internal/projectgen -count=1
```

Result: **PASS**. Tests verify the default `native` selection, the adapter list, and
preservation of the selection in manifests generated with `DesignSystem: "shadcn"`.

## Boundaries

This evidence does not cover installation of provider CSS/runtime packages or authentic
browser visual compatibility. Those dependencies belong to the Client Edition and are verified
under separate browser E2E/production gates.
