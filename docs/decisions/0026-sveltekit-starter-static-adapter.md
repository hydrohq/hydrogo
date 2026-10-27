# 0026: `sveltekit-starter` converted to a static SPA (adapter-static)

- Status: Accepted
- Date: 2026-10-24

## Context

0023 and 0025 left `sveltekit-starter` (and `crm-starter`/`commerce-starter`)
on `@sveltejs/kit@2.70.3` with `@sveltejs/adapter-node`: a long-running Node
process rendering every request server-side. This needs a host that can run
Node, which rules out plain CGI/shared hosting -- exactly the deployment
model (buy a cheap shared-hosting plan, no server to rent or manage) the
project's own `hydro.cgi` CGI/FastCGI mode exists for.

`examples/hydropost-news-starter` turned out to already be a working,
deployed proof that the alternative works: it uses `@sveltejs/adapter-static`
with a `fallback: 'index.html'` SPA shell -- no Node process, the static
build serves from the same host as the `hydro.cgi` binary, confirmed live
on a real shared-hosting account (read-only, over FTP, with the user's
explicit permission) and confirmed by actually running `npm run build`
against it locally. Two things about that example were corrected in the
same pass as this one:

- Its `.htaccess` (bot detection, SPA fallback, CGI routing, static-asset
  caching) only ever existed on the live server, never committed -- added
  to `static/.htaccess` so it isn't a single point of loss.
- Its `bot-meta.php` (serves real OpenGraph/Twitter Card/JSON-LD tags to
  crawlers that don't execute JavaScript -- WhatsApp, Facebook, Twitter/X,
  LinkedIn, Telegram, Discord, Googlebot's own preview fetcher) read the
  SQLite file directly, bypassing HydroGo's API entirely -- fragile (an
  internal schema change silently breaks it) and a boundary violation
  (frontends talk to the API, not the database). Rewritten to call
  `/api/v1/content/site_settings` and `/api/v1/news/{locale}/{slug}`
  instead; it also had a latent bug fixed in the same pass, treating
  `cover_media_id` as always being a HydroGo-internal media id when it is
  sometimes a full external image URL (an editor pasting a source's own
  image link), producing a broken image URL.

## Decision

Converted `sveltekit-starter` to the same pattern:

- `@sveltejs/adapter-node` -> `@sveltejs/adapter-static` with
  `fallback: 'index.html'`, `strict: false`.
- Every `+page.server.ts`/`+layout.server.ts` (server-only `load`, needs a
  running server) became `+page.ts`/`+layout.ts` (universal `load`, runs in
  the browser). `$lib/server/hydro.ts` became `$lib/hydro.ts`; the only real
  logic change is where the API origin comes from -- previously
  `$env/dynamic/private` (server-only, needs a running server to resolve);
  now, in the browser, an empty string (same-origin relative request,
  working because the deployed site and `hydro.cgi` share one domain
  through `.htaccess`) except on `localhost`/`127.0.0.1`, where the Vite dev
  server and HydroGo run on different ports and need the real address.
- `robots.txt`/`rss.xml`/`sitemap.xml` (`+server.ts` endpoints) gained
  `export const prerender = true` -- adapter-static only outputs prerendered
  routes plus the fallback shell, and these three are a reasonable,
  explicitly-named case to accept "freshens on the next rebuild" for (most
  RSS readers and search engines already poll on their own schedule; this
  isn't the zero-build-on-publish guarantee content pages get, and isn't
  meant to be).
- `src/params/locale.ts` needed the same two SvelteKit-3-only fixes already
  applied to `crm-starter` and `commerce-starter` in 0023's investigation: a
  `src/params/index.ts` aggregator (matchers load as one file exporting a
  `params` map now, not one file per matcher) and reading
  `process.env.PUBLIC_LOCALES` directly instead of through any `$env`
  module (matchers load via a plain Node import outside the Vite pipeline).
- `$env/static/public` itself would not resolve for `src/lib/config.ts` and
  `src/lib/hydro.ts` when combined with this starter's `alias: { $lib:
  'src/lib' }` (needed because SvelteKit 3 removed the built-in `$lib`
  alias) -- confirmed via `svelte-check`'s own diagnostic
  (`"paths" was overwritten`), a real interaction bug in this exact
  combination, not a mistake in how the alias or the env import were
  written. Worked around by using Vite's own native `import.meta.env.VITE_*`
  handling instead, which is independent of SvelteKit's `$env` wrapper and
  works identically in the browser and during the build-time prerender of
  the three `+server.ts` endpoints above. `.env.example` renamed its three
  variables to the `VITE_` prefix accordingly.
- `hooks.server.ts` removed: with no server at runtime, it would only ever
  have executed once, at build time, while generating the fallback shell --
  its security headers moved to `.htaccess` (`Header set` directives,
  something the old `.htaccess` above didn't have either, a gap named in
  0025's review and closed here); its CVE-2026-114060 `//`-path rejection
  was not carried over, because the vulnerable client-side pattern
  (`use:enhance` on a route accepting a `//`-prefixed path) requires
  `use:enhance` to exist in the app at all, and this starter has never used
  it (grepped to confirm, same as `hydropost-news-starter`).
- `app.html`'s `%lang%` placeholder (only ever replaced by a per-request
  server hook, so always meaningless for a prerendered fallback shell --
  confirmed it already always fell back to `"en"` even before this change,
  since `event.locals.lang` was never actually set before the hook ran) was
  replaced with a static `lang="en"` plus a client-side
  `document.documentElement.lang` update once the real locale is known. A
  static `<title>` default was added for the same reason -- the fallback
  shell previously had none at all.
- `packages/client/src/hydrogo.ts` (the canonical client `sveltekit-starter`
  vendors a copy of, enforced by
  `internal/scaffold.TestStarterVendoredClientMatchesPackage`) gained the
  same empty-origin-means-relative support, since the vendored copy must
  match exactly. This is a backward-compatible addition (an absolute URL
  still behaves exactly as before); any other project using
  `@hydrogo/client` is unaffected unless it opts in by passing `origin: ""`.
- `scripts/smoke.mjs` rewritten: it used to spawn the adapter-node server
  and check that a missing article/API outage produced a real HTTP
  404/502. A static SPA cannot do that -- those states are now a
  client-side error-boundary render, not a server response status, which
  is a genuine, named trade-off of this architecture (also true of
  `hydropost-news-starter`, not new here). The rewritten smoke test
  instead serves `build/` the way the deployed `.htaccess` does (real
  file as itself; `_app/` 404s instead of falling back; everything else
  gets the SPA shell) and checks that contract.

## Consequences

- `go run ./tools/check` passes. `npm run check`/`build`/`smoke` all pass
  for both `hydropost-news-starter` and `sveltekit-starter`; `npm audit`
  is 0 vulnerabilities for `sveltekit-starter` now that it is genuinely on
  `@sveltejs/kit@3.1.0` -- the two CVEs 0025 mitigated without a version
  bump are closed by the real fix here, for this starter specifically.
- `bot-meta.php`'s rewrite was verified against a live, real deployment's
  API (read-only `file_get_contents` calls, no credentials needed), not
  just locally: fetched a real published article, confirmed the generated
  title/OG tags/JSON-LD are correct, and confirmed the not-found and
  no-slug fallback paths degrade gracefully.
- `crm-starter` and `commerce-starter` remain on `adapter-node` /
  `@sveltejs/kit@2.70.3`: both have their own server-side secrets (session
  cookies for `crm-starter`; PayTR merchant credentials for
  `commerce-starter`) that cannot move to a browser-only static SPA
  regardless of adapter choice, so this conversion does not apply to them.
  `commerce-starter`'s own vendored `src/lib/hydrogo.ts` copy was not
  touched -- it has no enforcing sync test and is unrelated to this change.
- The home page's first paint still waterfalls (download JS -> run it ->
  fetch content -> render) rather than arriving fully rendered, same
  trade-off already accepted for `hydropost-news-starter`'s architecture.
  Not addressed here; `bot-meta.php` solves the crawler/link-preview half
  of this, not the human-visitor perceived-performance half.
