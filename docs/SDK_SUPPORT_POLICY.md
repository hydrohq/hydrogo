# SDK v1 Support Policy

HydroGo SDK compatibility follows the same-major version rule. A `1.x` client surface operates with a `1.x` runtime; a major change is not considered automatically compatible and requires a dedicated migration plan and ADR. Candidate versions (`-candidate`) do not carry a stable release guarantee.

Each minor release targets backward-compatible additions. Deprecated symbols are maintained with deprecation warnings for at least one subsequent minor release; removals are performed exclusively in new major releases. Critical security fixes are backported to the latest supported stable major release. APIs in candidate releases remain subject to change prior to a final v1 release.

This policy does not substitute for pre-sales support agreements or the production key-signing ceremony. When a stable v1 is released, the supported major versions, deprecation schedules, and security SLAs will be formally announced alongside the release manifest.
