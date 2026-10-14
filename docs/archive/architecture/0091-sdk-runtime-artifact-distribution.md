# ADR-0091: Public SDK and Runtime Artifact Distribution

## Status

Proposed

## Context

Providing a smooth CLI onboarding experience across Linux, Windows, and macOS requires a standalone `hydro` CLI. In architectures using standalone binaries or precompiled distributions, client projects should not be burdened with private repository clones, permanent personal access tokens, or unpinned dependencies.

## Decision

The Public SDK provides the CLI, project generator, starters, contracts, frontend adapters, and developer tooling. Precompiled runtime artifacts are produced and signed via CI pipelines. When an artifact broker is configured, the CLI acquires short-lived download authorization; private registry credentials remain on the broker side. Diagnostic commands like `git ls-remote` serve only as connectivity diagnostics, not entitlement or authentication mechanisms.

Projects lock their SDK, runtime, and module versions and SHA-256 hashes inside `hydrogo.lock`. The CLI performs strict checksum, signature, platform, and compatibility verification prior to executing runtimes. Offline operation is supported via local signed manifests and controlled artifact caching.

Signed manifest verification ensures every artifact path is canonical; absolute paths, relative traversal sequences (`../`), or invalid size/checksum metadata are rejected fail-closed.

The Public SDK sends only entitlement coordinates in JSON bodies and short-lived tokens in Authorization headers. Responses are parsed with strict byte bounds; redirects, oversized payloads, and non-200 statuses result in fail-closed errors.

The runtime cache lifecycle is part of the distribution contract: artifacts are stored under `root/<version>/<canonical-target>`. Cache garbage collection operates with an explicit preserve list, offering dry-run planning before deleting files. Symlinks, directory traversals, and non-regular files are rejected.

## Consequences

- Positive: Clean separation between developer tooling and runtime binaries; seamless onboarding experience comparable to modern package managers; air-gapped environments supported; release integrity cryptographically verified.
- Costs & Tradeoffs: Requires broker infrastructure, cross-platform signing, installer maintenance, and artifact cache management.

## References

- [ADR-0010: SDK lifecycle and Edition manifest](0010-sdk-lifecycle-and-edition-manifest.md)
- [ADR-0028: Core isolation and commercial licensing](0028-core-isolation-and-licensing.md)
- [Payload parity execution plan](../PAYLOAD_PARITY_EXECUTION_PLAN.md)
- [SDK runtime distribution plan](../HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md)
