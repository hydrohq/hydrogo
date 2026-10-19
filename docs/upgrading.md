# Upgrading and versions

## Versioning

HydroGo follows [Semantic Versioning](https://semver.org). While the major version is 0, a minor release (0.x) may change the API or config, and every such change is listed under "Changed" or "Removed" in [CHANGELOG.md](../CHANGELOG.md) with what to do. Patch releases only fix bugs. From 1.0, breaking API changes only come with a major version.

Database migrations are forward-only and never edited after release, so any 0.x database can be upgraded to the next release by running the new binary's `hydro migrate`. Security fixes go into the latest release and the one before it (see [SECURITY.md](SECURITY.md)).

The public API contract is in [`api/`](api); CI compares every change against the committed baseline (`tools/openapidiff`) so accidental breaks are caught before release.

## Upgrade steps

```bash
hydro backup sqlite --destination backups/ --media-dir data/media   # 1. back up (or pg_dump / mysqldump)
# 2. replace the hydro binary (re-run install.sh, the Windows installer, or pull the new image)
hydro upgrade check --config config.yaml                            # 3. what will change
hydro migrate --config config.yaml                                  # 4. apply migrations
hydro doctor --config config.yaml                                   # 5. verify the install
hydro serve --config config.yaml
```

`hydro migrate` takes a snapshot of a SQLite database before it applies anything; `hydro rollback --snapshot PATH --confirm` restores it. Pin a version with `HYDRO_VERSION=v0.2.0 sh install.sh` or the matching image tag.

## Releasing (maintainers)

Releases are published only to this repository: GitHub Releases for the binaries and installers, and the repository's own container registry (GHCR) for the image. Their visibility follows the repository's. Nothing is sent to npm, Docker Hub, winget, Homebrew or any other public registry.

1. Move the "Unreleased" notes in `CHANGELOG.md` under a new `## [x.y.z] - date` heading.
2. `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. `.github/workflows/release.yml` builds binaries for Linux, macOS and Windows (amd64, arm64), the Windows `Setup.exe`, `SHA256SUMS` and the container image, and creates the GitHub release with that changelog section. Tags with a hyphen (`v0.2.0-rc.1`) publish as pre-releases and do not move the `latest` image tag.
4. If the public API changed, update `packages/client` and copy `packages/client/src/hydrogo.ts` to `examples/sveltekit-starter/src/lib/hydrogo.ts` (a test fails when they differ). Nothing is published to npm.
