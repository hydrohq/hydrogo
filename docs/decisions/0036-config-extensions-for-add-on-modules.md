# 0036: Config extensions for add-on modules

- Status: Accepted
- Date: 2026-11-22

## Context

Payment gateways, shipping carriers and e-invoicing providers are optional add-on modules that are distributed separately. Their settings (`payments:`, `carriers:`, `invoicing:`) used to be typed fields of `config.Config`, with their own defaults, `HYDRO_*` overrides, validation and production secret checks inside core. That tied core to provider names and to the shape of each provider's settings.

## Decision

Core owns no provider settings. An add-on module registers one top-level config section with `config.RegisterExtension`:

- `Section`: the YAML key the module owns.
- `Defaults`: a pointer to the section value with defaults applied.
- `Env`: applies `HYDRO_*` environment overrides.
- `Validate`: checks the final value (it also receives the whole `Config`).
- `ProductionIssues`: lists secrets and settings that must not reach production as they are.

`Load` captures top-level keys it does not know, hands the ones an extension owns to that extension (strictly: unknown keys inside the section are rejected) and still rejects every other unknown key. Without the module, a section such as `payments:` is therefore refused like any typo. The decoded value is available as `config.ExtensionValue[*T](cfg, "section")`; modules wrap it in a typed `FromConfig` helper. Hooks fall back to `Defaults` when a `Config` was built without `Load`.

Maintenance-mode bypass for callbacks that an add-on serves (gateway or carrier notifications) is registered the same way with `server.AllowDuringMaintenance`.

## Consequences

- Core builds, validates and runs with no knowledge of any provider; the YAML and environment variable names users see are unchanged when the module is present.
- Secrets validation for add-ons runs through `config.ExtensionProductionIssues`.
- A module is applied with its own build tag and registers itself from `init`, so there is no central list to edit.
