# ADR-0010: SDK maturity, module lifecycle and Edition manifest

- **Status:** Accepted
- **Date:** 2026-09-05
- **Supersedes ADR:** [ADR-0008](0008-module-and-extension-contract.md)
- **Implementation Status:** SDK v0/v1 candidate, Edition manifest, generator, and release provenance contracts implemented and verified with contract tests. Official v1 release and production key/provenance evidence open in R5; current status is in [roadmap](../ROADMAP.md).

## Context

ADR-0008 set the boundaries correctly but presented a broad Go interface as a mandatory API before validating two modules. The planned root `core/` directory also didn't explicitly show the application boundary closed to the outside. The new decision preserves the previous extension, generated/custom, and compatibility principles; it clarifies API maturity and packaging behavior.

## SDK and dependency boundary

- HydroCore becomes a versioned Go module. The client project is a separate Go module and its own repository; Core source is not copied. The development workspace can open them together.
- Core's `internal/` scope is an implementation detail. `extension/` is the client/module API. Client Edition only imports extension and declared module APIs.
- The `internal` rule depends on the Go source tree boundary. It may not provide isolation on its own for an example client residing under the same root; CI also checks the allowed import list and a separate external consumer fixture.
- Go package separation is not a sandbox. Client/module code compiled into the binary is considered trusted code; SDK cannot prevent malicious code's file/network/DB access. Untrusted runtime plugins are not loaded.
- Domain and application service do not know SQL/cache dialects. The module's infrastructure adapter layer can use the controlled DB executor contract bound to the transaction by the platform. This capability does not leak into the public handler/view SDK. The "No extension can use SQL" rule should not make DB adapter development impossible.

## Maturity stages

1. Boundaries are accepted now; package names and Go signatures are experimental v0.
2. Pages and News are developed manually; unnecessary interfaces are narrowed down. Registration is made through the explicit composition root; `init()` side effects and global registry are not used.
3. Module identity/dependencies are mandatory. Contributions like route, job, admin, migration are small contracts based on need; every module does not have to write a ten-method empty implementation.
4. The SDK surface to be published is fixed as v1 when two modules and two Client Editions pass contract tests. The generator becomes stable after this.

v0 versions are also locked with a full version and carry a changelog/migration guide; silent breakage is not accepted. Published v1 contracts are preserved within the same major version. If behavior must change for security reasons, the impact and upgrade path are explicitly announced.

## Extension formats and hook behavior

Order of preference: config/theme tokens → typed validation/hook → view/component slot → custom service/route → full view override.

- A validation hook can reject the write or normalize allowed input fields. Final authorization, data validation, and state control are then reapplied by the service; a hook does not relax the core policy.
- In-transaction hooks are short and deterministic; they do not make network/email calls. Their failure rolls back the transaction. Since retry can occur, external side effects are prohibited.
- Post-commit events are delivered at least once with a durable outbox. The event schema is versioned, the consumer is idempotent; one consumer's failure does not erase the other's delivery state.
- Hook order is deterministic with explicit priority and static identity; a loop/double registration is a startup error. A timeout/cancellation contract is applied; this is not a guarantee to forcefully stop untrusted Go code.
- View slots are typed and bound at compile time. A dynamic registry is used only if the need is proven in two themes. A client view cannot bypass the authorization or publication filter directly; it receives a safe view model.

## Module lifecycle

| Action | Behavior |
|---|---|
| Add to build | Dependency resolution, DB capability, license/version check; only selected modules are imported/embedded |
| Activation | Migration plan within the inspected release is applied; then the new binary is ready. HTTP startup does not run DDL by itself |
| Execution | Only routes, assets, jobs, and resources of active modules are exposed |
| Deactivation | Dependent modules are checked; new jobs stop; old workers are drained; queued jobs are parked; the new build does not include the module |
| Reactivation | The stored schema/migration ledger is audited; a new release is prepared through a supported upgrade path |
| Uninstallation | No automatic table/file deletion. A separate purge plan, backup, authorization, and audit are required for data and relations |
| Downgrade | Reverting only to an old release compatible with current schema, event/job, and content formats; no general automatic down migration |

Arbitrary Go modules are not loaded via admin click at runtime. Module selection creates a release. Temporary operational shutdown can stop new production; it does not remove bytes from the build. The unused module payload promise is verified via the Edition build where the module is not imported/embedded. Stored old client data is not an "active module resource".

The platform orchestrator can see the ledger/data record of a passive module; it does not run the passive module's migration automatically. Deactivation is rejected if a dependent active module requires the passive module's data.

## Manifest and release

There are three separate documents; names are pre-implementation drafts:

- `edition.yaml`: Edition identity, selected modules, presentation properties, and content definitions. Contains no secrets.
- `edition.lock`: Resolved exact Core/module/toolchain/generator versions and content hashes. Must be consistent with the Go module graph. The first official modules can be versioned together with Core; independent versioning is not mandatory.
- `release.json`: Actually built dependencies, source revisions, asset inventory, binary digest, DB/migration compatibility, job/event/content formats, config schema, and build provenance. The package signature binds the binary with this; trust in the signing key is established at installation.

Deployment profile, DB connection, domain, and secret references are environment configs. Secret values do not enter the manifest/lock. Priority: explicit CLI flag → environment/secret reference → config file → documented default. Conflicting secret sources and unknown config keys cause an error. The config cannot silently select another DB if the driver is not included.

If the actual dependency graph resolved by Go does not meet the lock expectation, the build stops. The "Core 1.x + module 2.x" tag alone is not proof of compatibility. The release matrix tests SDK, schema, config, content, event/job, and asset compatibility together.

## Presentation and generator

Site Edition default is `integrated`: admin + public Templ site + public JSON API. `headless`: admin + API, public Templ site disabled. `hybrid` is not a separate execution engine; it is a way of using the integrated API with external clients. The ambiguity among the three modes in the old document is thus removed.

`generated/` is fully reproducible; `custom/` is not written by the generator. The generator version and spec hash are recorded. Reproduction first presents a diff; it does not modify an existing published migration. Security decisions and destructive transformations are not guessed.

## Alternatives, impacts and validation

Making a broad interface immediately v1 is premature abstraction; runtime Go plugin is a trust/distribution cost. The selected build composition requires a client-specific rebuild, but narrows the runtime and support surface.

SQLite/PostgreSQL and future adapters are subject to the module contract kit. A passive module does not open a cache namespace/worker. Compact passes minimum build resource tests; Distributed passes bounded pool/worker tests. The old/new module set, job payload, and asset retention are evaluated together in blue/green.

Evidence: an external consumer cannot import internal; two Client Editions build without copying Core; no route/asset/job/migration record is created when a module is deactivated; a dependency cycle is rejected; the same spec does not create a diff in the second generation; custom file hashes are preserved; an old supported client passes the new Core contract test.

## References

- [Go internal package boundary](https://go.dev/doc/go1.4#internalpackages). Go behavior verified with source; the packaging and SDK lifecycle selected in this document are HydroGo design decisions.
