import { type DocSection, page, p, h2, h3, code, note, tip, warn, table, ul, ol } from './types';

export const upgrading: DocSection = {
  id: 'upgrading',
  title: 'Upgrading',
  pages: [
    page(
      'versioning',
      'Versioning policy',
      "How HydroGo numbers releases, what a version bump may break, and how database migrations are handled.",
      p("HydroGo follows [Semantic Versioning](https://semver.org). The version is injected at build time and shown by `hydro version`."),
      table(
        ['Release', 'May contain'],
        ['Patch (`0.2.1`)', 'Bug fixes only.'],
        ['Minor (`0.3.0`) while the major is 0', 'New features **and** changes to the API or config. Each such change is listed under "Changed" or "Removed" in the changelog with what to do.'],
        ['Major (`1.0.0` and later)', 'Breaking API changes only come with a new major version.'],
        ['Pre-release (`0.3.0-rc.1`)', 'Published as a GitHub pre-release; does not move the `latest` image tag.'],
      ),
      h2('Migrations'),
      ul(
        "Database migrations are **forward-only** and never edited after release, so any 0.x database can be upgraded to the next release by running the new binary's `hydro migrate`.",
        "Every migration ships SQL for SQLite, PostgreSQL and MySQL.",
        "A migration directory is never deleted: a database that applied it refuses to start without it.",
      ),
      h2('API stability'),
      p("The public API contract is kept in the repository under `docs/api/`. CI compares every change against the committed baseline, so accidental breaking changes are caught before release. Your frontends can pin the contract by generating a client from it with `hydro sdk generate`."),
      h2('Security fixes'),
      p("Fixes land in the latest release and the one before it. Stay within one release of the latest. See [Reporting vulnerabilities](doc:security/reporting)."),
      tip("Read `CHANGELOG.md` before every minor upgrade. The Unreleased section shows what is coming."),
    ),

    page(
      'upgrade-steps',
      'Upgrade steps and rollback',
      "A safe, repeatable procedure: back up, replace the binary, check, migrate, verify. And how to roll back.",
      h2('Procedure'),
      ol(
        "**Back up.** Database and media (see [Backups](doc:deployment/backups)).",
        "**Replace the binary.** Re-run `install.sh` or the Windows installer, or pull the new image tag.",
        "**Preview the change** with `hydro upgrade check`.",
        "**Apply migrations** with `hydro migrate`.",
        "**Verify** with `hydro doctor`, then start the server.",
      ),
      code('bash', `hydro backup sqlite --destination backups/ --media-dir data/media   # 1. back up (or pg_dump / mysqldump)
# 2. replace the hydro binary
hydro upgrade check --config config.yaml                            # 3. what will change
hydro migrate --config config.yaml                                  # 4. apply migrations
hydro doctor --config config.yaml                                   # 5. verify the install
hydro serve --config config.yaml`),
      p("Pin a version in automation instead of taking whatever is newest:"),
      code('bash', "HYDRO_VERSION=v0.2.0 sh install.sh\n# or the matching image tag\ndocker pull ghcr.io/ozyildirims/hydrogo:v0.2.0"),
      h2('Rolling back'),
      p("`hydro migrate` takes a snapshot of a **SQLite** database before it applies anything. If the upgrade misbehaves, restore it:"),
      code('bash', "hydro rollback --snapshot PATH --confirm --config config.yaml"),
      ul(
        "Reinstall the previous binary version first, then run the rollback.",
        "For PostgreSQL and MySQL, restore the backup you took in step 1; the snapshot mechanism is for SQLite.",
        "Never reverse a migration by deleting tables or editing the migration ledger.",
      ),
      warn("Rolling back the database discards content written after the snapshot. Put the site in maintenance mode (Site settings, Maintenance) while you upgrade a busy site."),
      h2('Zero-downtime notes'),
      ul(
        "Run `hydro migrate` as a separate step before starting new instances, then roll instances one at a time behind your load balancer using `/readyz`.",
        "Keep the previous image tag so you can switch back quickly.",
        "Because HydroGo is stateless apart from the database and media, a blue/green switch is a proxy change.",
      ),
      h2('Related commands'),
      table(
        ['Command', 'Purpose'],
        ['`hydro upgrade check`', 'Show pending migrations and what will change.'],
        ['`hydro db diff`', 'Compare the database with the expected schema.'],
        ['`hydro db reindex`', 'Rebuild database indexes (maintenance).'],
        ['`hydro schema diff --baseline A --current B --public-key KEY`', 'Compare signed release migration ledgers.'],
        ['`hydro migration create --id YYYYMMDDHHMM_name --module MODULE --out migrations`', 'Scaffold a migration (contributors).'],
      ),
    ),

    page(
      'releasing',
      'Release process',
      "For maintainers: how a tagged release produces binaries, the Windows installer, checksums and container images.",
      ol(
        "Move the **Unreleased** notes in `CHANGELOG.md` under a new `## [x.y.z] - date` heading.",
        "Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.",
        "The release workflow (`.github/workflows/release.yml`) builds binaries for Linux, macOS and Windows (amd64 and arm64), the Windows `Setup.exe`, `SHA256SUMS` and the container image, and creates the GitHub release with that changelog section.",
        "Tags containing a hyphen (`v0.2.0-rc.1`) publish as pre-releases and do not move the `latest` image tag.",
        "If the public API changed, update `packages/client` and copy it to `examples/sveltekit-starter/src/lib/hydrogo.ts` (a test fails when they differ). Nothing is published to npm.",
      ),
      code('bash', "git tag v0.2.0\ngit push origin v0.2.0"),
      h2('Verifying a download'),
      p("Every release includes `SHA256SUMS`. The install scripts verify it automatically; to check by hand:"),
      code('bash', "sha256sum -c SHA256SUMS --ignore-missing"),
      p("Release manifests can also be verified with the built-in command:"),
      code('bash', "hydro release verify --manifest PATH --public-key PATH [--root PATH]"),
      note("Contributors: do not edit a released migration. Add a new one."),
    ),
  ],
};
