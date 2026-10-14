# ADR-0342: Bilingual, build-time Tailwind documentation surface

- **Status:** Accepted
- **Date:** 2026-10-02
- **Supersedes:** ADR-0338's `custom/ui/tailwind/`-only scope for the existing public `/docs` route, and only for that route
- **Superseded by:** None
- **Related decisions:** ADR-0146, ADR-0209, ADR-0212, ADR-0338, ADR-0340

## Context

The existing public `GET /docs` route is a developer/operator reading surface.
Its first implementation used hand-written CSS and linked to repository files.
The requested product behavior is a self-contained, responsive documentation
page in English and Turkish, matching the blue palette of the technical root.
The user explicitly chose Tailwind for this route. ADR-0338 previously limited
standalone Tailwind compilation to `custom/ui/tailwind/`; this exception needs
to be recorded rather than applied silently.

## Decision

1. `GET /docs` keeps its existing public route. English is the default.
   `?lang=tr` selects Turkish; `?lang=en` selects English. Unsupported values
   fall back to English. The response declares `Content-Language`.
2. Both languages contain the same section identifiers and the full technical
   guide. Language switching uses ordinary links and requires no browser state
   or JavaScript. Documentation content is embedded with the runtime, without
   links to the repository or a network documentation dependency.
3. The page has an isolated Tailwind v4 source file under `cmd/hydro/`.
   `scripts/build-docs-tailwind.ps1` invokes the standalone CLI at build time;
   the compiled `docs_portal.css` is committed and embedded into the Go binary.
   The production process does not run the CLI, Node, or a CSS watcher.
4. The docs stylesheet is loaded only on `/docs`. It is never mixed with
   Admin Studio family styles or `custom/ui/tailwind` output. The blue,
   dark palette follows the existing technical root surface.
5. Content describes Core contracts and their practical limits. It must not
   turn a plan, local test, or optional module into a production guarantee.

## Alternatives

- Keep hand-written CSS: would ignore the explicit Tailwind requirement and
  perpetuate a parallel, one-off styling method for this route.
- Load Tailwind from a CDN: adds a runtime network dependency and conflicts
  with the single-binary release model.
- Serve repository Markdown as separate public routes: adds routes and a
  source-file publication policy that the requested self-contained page does
  not need.

## Consequences

### Benefits

- The route can be read offline, in either language, with no extra service.
- Blue visual identity and responsive navigation match the technical root.
- The Tailwind compilation boundary is explicit and repeatable.

### Costs and risks

- Two translations must be maintained together. Tests assert shared sections
  and no external repository links.
- The compiled CSS must be regenerated after source changes. Tests check that
  the embedded output includes the route's selectors and blue token.

## Scale impact

- Compact: static embedded HTML/CSS only; no new route, worker, connection,
  cache, or frontend runtime.
- Standard: same static response path.
- Distributed: no shared state or coordination; language is in the URL.

## Database impact

- SQLite: no queries, schema changes, or migration.
- PostgreSQL: no queries, schema changes, or migration.
- Future adapters: no database contract dependency.
- Certified database behavior tests and migration metadata are not applicable
  because this route does not access or change a database.

## Cache impact

The server has no docs-specific cache. Language is explicit in the query URL;
the page is rendered per request with the request's CSP nonce. No invalidation,
stampede, or failure fallback is needed beyond the embedded static asset and
the existing server behavior.

## Deployment and compatibility

Old and new binaries can coexist during Blue/Green deployment. Both serve the
same `/docs` path; the new binary additionally honors `lang`. No persistent
state, media, session, job, or asset URL changes are required. Rollback means
switching back to the previous binary. The committed CSS is compiled before
building the new binary and requires no deployment-time toolchain.

## Validation

- `go test ./cmd/hydro` covers both locales, default fallback, CSP nonce
  escaping, shared section IDs, compiled CSS presence, and no GitHub links.
- Browser checks cover desktop and narrow layouts, language switching, section
  navigation, overflow, and readable contrast.
- Existing moduleguard, migrationguard, and importguard CI gates remain
  mandatory and unchanged.

## References

- `cmd/hydro/docs_portal.go`
- `cmd/hydro/docs_portal_en.html`
- `cmd/hydro/docs_portal_tr.html`
- `cmd/hydro/docs_portal.source.css`
- `scripts/build-docs-tailwind.ps1`
