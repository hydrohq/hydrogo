# Hydro SDK Artifact Broker API Contract

This document specifies the minimum contract between the public hydro CLI and the
private runtime artifact service. The broker never exposes source code or personal access
tokens (PATs) to the client; it exclusively issues short-lived, project- and version-bound
download grants.

## Authentication

hydro login authenticates users via OAuth 2.1 Authorization Code with PKCE or a headless
device flow. Refresh tokens are persisted exclusively within the operating system's native
credential store. CLI logs strictly redact access/refresh tokens, Authorization headers,
and signed artifact download URLs.

Broker requests:

`	ext
POST /v1/runtime/grants
Authorization: Bearer <short-lived access token>
Content-Type: application/json
`

`json
{
   project_id: project-123,
  core_version: 1.4.0,
  sdk_version: 1.2.0,
  target: linux-amd64,
  edition: growth,
  features: [news, media]
}
`

The broker validates organization and project entitlements, edition/feature compatibility,
and platform target support on the server side. Repository membership alone does not constitute
a valid license or artifact entitlement.

## Successful Grant

The HTTP 200 response must be a JSON payload of at most 64 KiB containing the following fields:

`json
{
  access_token: opaque-short-lived-token,
  project_id: project-123,
  core_version: 1.4.0,
  sdk_version: 1.2.0,
  artifact_url: https://artifacts.example/v1/...,
  issued_at: 2026-09-16T12:00:00Z,
  expires_at: 2026-09-16T12:10:00Z
}
`

Grants are valid for at most 24 hours; normal download grants should expire within a few minutes.
The URL must use HTTPS, omit embedded credentials and fragments, and redirect following must be
disabled. The CLI will not initiate a download until grant metadata matches the lockfile and release manifest.

## Error Contract

| HTTP | Meaning | CLI Behavior |
| --- | --- | --- |
| 400 | Invalid target, version, or JSON | Present user-correctable fields; redact token |
| 401 | Invalid or expired session | Clear local session; prompt to re-authenticate |
| 403 | Missing entitlement | Access denied; omit repository or credential details |
| 404 | Artifact not found | Suggest verifying version or target platform support |
| 409 | Lockfile and edition mismatch | Suggest verifying lockfile against project settings |
| 429 | Broker rate limit exceeded | Bounded retry respecting Retry-After header |
| 5xx | Transient broker error | Bounded retry, then fail closed |

The error payload consists of bounded, machine-readable code, message, and equest_id fields;
stack traces, internal SQL errors, GitHub URLs, and secrets are strictly forbidden.

## Internal Distribution

The broker accesses private GitHub Apps or container/artifact registry credentials exclusively
on the server side. Artifacts and signed release manifests produced by CI are pulled from internal
object storage or registries. Source repositories, PATs, and persistent registry secrets are never
transmitted to clients.

## Operations and Acceptance

- Grant issuance, denial, expiration, and revocation events emit audit log entries free of PII and secrets.
- Signing keys reside in HSM/KMS or access-segregated release environments; key IDs and overlap windows govern rotation.
- Revocation, mirroring, and air-gapped transport workflows are verified through separate runbooks and drills.
- Contract test suites cover valid grants, mismatches, expiration, oversized responses, HTTP redirects, 401/403/429/5xx status handling, and token redaction.

This document specifies the interface contract, not the full backend service implementation. Production
acceptance requires verified live broker endpoints, OAuth client registration, registry credentials, and
a key ceremony.
