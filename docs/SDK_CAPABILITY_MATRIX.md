# HydroGo SDK capability matrix

This matrix is the compatibility snapshot for Client Editions. A capability is
listed only when a versioned contract and local evidence exist; internal
packages are not SDK surface.

| Surface | Namespace | Maturity | Compatibility rule | Evidence |
|---|---|---|---|---|
| API version negotiation | `extension/v1` | Candidate | Same major version; candidate status is explicit | `version_test.go` |
| Module identity and routes | `extension/v1` | Candidate | `Module`/`ModuleRoute` aliases preserve existing source compatibility | `module_test.go` |
| Module dependency validation | `extension/v1` | Candidate | Names normalized; unknown nodes and cycles rejected | `dependencies_test.go` |
| Optional module lifecycle hooks | `extension/v1` | Candidate | Install/enable/disable/upgrade and migration preflight are separate opt-in contracts | `lifecycle_test.go` |
| Theme rendering | `extension/v1` | Candidate | `Theme` is the minimum; `BlockTheme` is optional | `theme_manifest_test.go`, edition builds |
| Theme manifests/assets | `extension/v1` | Candidate | Validated names, slots, assets, and safe paths | `theme_manifest_test.go`, fuzz smoke |
| Taxonomy contract | `extension/v1` | Candidate | Bounded localized terms, parent graph and content relations; persistence stays module-owned | `taxonomy_test.go`, ADR-0120 |
| Forms contract | `extension/v1` | Candidate | Bounded field definitions and submission normalization; inbox/retention stay module-owned | `forms_test.go`, ADR-0121 |
| Editorial workflow contract | `extension/v1` | Candidate | Bounded custom states/transitions and same-origin preview path; scheduling/auth remain module-owned | `workflow_test.go`, ADR-0122 |
| Search provider contract | `extension/v1` | Candidate | Bounded queries and deterministic hard-bounded result pages; authorization remains module-owned | `search_test.go`, ADR-0123 |
| Generated module scaffold | `internal/generator` (tooling only) | Local tooling | Generated code requires business-rule review | generator tests and smoke |
| Core module registry/lifecycle | `internal/modules` | Internal | Not importable by Client Editions | import-boundary checks |
| SQL/cache/queue adapters | `internal/*` | Internal | Never exposed through domain/application contracts | import guard and architecture ADRs |

## Support policy

`extension/v1` is candidate-stable, not an LTS promise. Breaking changes must
use a new versioned namespace or an ADR-backed deprecation window. The matrix
must be updated with contract tests before a capability is advertised to an
external edition.
