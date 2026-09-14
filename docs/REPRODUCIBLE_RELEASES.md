# Reproducible release artifacts

HydroGo release builds use `-trimpath`, `-buildvcs=false`, fixed `-ldflags`,
and an empty Go build ID. The CI and production-gate workflows build the
binary twice and compare SHA-256 digests before publishing release evidence.
The local gate also runs the same normalized build path.

Artifact provenance is intentionally not claimed until the CI identity,
signing-key ceremony, registry, and retention policy are selected. A release
must record the commit, toolchain version, build command, digest, SBOM digest,
and verification timestamp in its evidence bundle.
