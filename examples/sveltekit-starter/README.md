# HydroGo SvelteKit starter

A server-rendered public site on the HydroGo API. Pages are rendered on the server for every request, so search engines and link-preview bots get complete HTML (title, canonical, Open Graph, hreflang, JSON-LD), and a publish in the HydroGo admin is live on the next request: **no rebuild when content changes**.

Included: home, news list with pagination, article page, pages by slug, header navigation and site name from the `site_navigation` / `site_settings` content types, multi-language URLs (`/tr/...`), `sitemap.xml`, `rss.xml`, `robots.txt`, a styled error page, dark mode, and short CDN-friendly cache headers. It has no dependency on `@hydrogo/client`: a copy of the client lives in `src/lib/hydrogo.ts`.

## Run it

```bash
cp .env.example .env     # set HYDRO_API_ORIGIN to your HydroGo server
npm install
npm run dev              # http://localhost:5173
```

Production: `npm run build && node build` (port 3000), or `docker build -t my-site . && docker run -p 3000:3000 --env-file .env my-site`.

With a project made by `hydro new mysite --starter sveltekit`, this starter is already in `mysite/frontend`.

## Configure

| Variable | Meaning |
|---|---|
| `HYDRO_API_ORIGIN` | Where HydroGo runs (server-side only). |
| `PUBLIC_SITE_ORIGIN` | Public URL of this site; used for canonical links, sitemap, RSS. |
| `PUBLIC_LOCALES` | `en` or `en,tr`. The first is the default (no URL prefix); the rest get `/tr/...`. Must match your content's locales. |

## Where to change things

| You want to… | Edit |
|---|---|
| Change layout, header, footer | `src/routes/[[lang=locale]]/+layout.svelte`, `src/lib/components/` |
| Change colours and typography | `src/app.css` (CSS variables at the top) |
| Show another content type | copy `news/` and call `hydro.listContent('<type>')` ([content types](../../docs/content-types.md)) |
| Change what a page loads | the `+page.server.ts` next to it |
| Cache longer or shorter | the `cache-control` in `setHeaders(...)` |

## Notes

- Rich text is rendered with `{@html}`: HydroGo sanitizes it on the server. Do not render untrusted HTML from other sources this way.
- If the API is down, pages answer `502` (not an empty page), so CDNs and crawlers do not cache an outage.
- Drafts never appear: the public API only returns published content.
