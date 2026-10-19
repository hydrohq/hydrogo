import { type DocSection, page, p, h2, h3, code, tabs, note, tip, warn, table, ul, ol } from './types';

export const frontends: DocSection = {
  id: 'frontends',
  title: 'Frontends',
  pages: [
    page(
      'client',
      '@hydrogo/client',
      "A typed, dependency-free fetch client for the public API. Works in browsers, Node 20+, Deno, Bun and edge runtimes.",
      p("`@hydrogo/client` wraps the public endpoints with TypeScript types, request timeouts, size-capped responses and a proper error class. You can also call the API with plain `fetch`; the client just removes boilerplate."),
      h2('Install'),
      p("The client is not published to a public registry. Use the copy bundled with the SvelteKit starter (`hydro new --starter sveltekit` puts it at `frontend/src/lib/hydrogo.ts`), or install it from a HydroGo checkout:"),
      code('bash', "npm install ../path/to/hydrogo/packages/client"),
      h2('Create a client'),
      code('ts', `import { createHydroGoClient } from '@hydrogo/client';

const hydro = createHydroGoClient({
  origin: 'https://cms.example.com',
  timeoutMs: 8000,        // default 15000, maximum 120000
  // fetch: customFetch,  // inject your own fetch (tests, edge runtimes)
});`),
      table(
        ['Option', 'Default', 'Meaning'],
        ['`origin`', 'required', 'Base URL of your HydroGo, without a trailing path.'],
        ['`timeoutMs`', '`15000`', 'Per-request timeout. Capped at 120000.'],
        ['`fetch`', 'global `fetch`', 'Custom fetch implementation.'],
        ['`signal`', 'none', 'An `AbortSignal` applied to every request.'],
      ),
      h2('Methods'),
      table(
        ['Method', 'Calls', 'Returns'],
        ['`getPage(locale, slug)`', '`GET /api/v1/pages/by-slug`', '`Page`'],
        ['`listPages(locale, limit = 20)`', '`GET /api/v1/pages`', '`PageSummary[]`'],
        ['`listNews(locale, limit = 20)`', '`GET /api/v1/news`', '`NewsItem[]`'],
        ['`getNews(locale, slug)`', '`GET /api/v1/news/by-slug`', '`NewsItem`'],
        ['`listContent<T>(type, query)`', '`GET /api/v1/content/{type}`', '`{ data: T[], metadata }`'],
        ['`getContent<T>(type, id, locale?)`', '`GET /api/v1/content/{type}/{id}`', '`T`'],
        ['`memberLogin`, `memberRegister`, `memberRefresh`, `memberLogout`, `memberDevices`, `memberMe`', '`/api/v1/mobile/member/*`', 'Token pairs, devices, profile'],
      ),
      p("Every method takes an optional last argument `{ signal }` for cancellation. The `query` for `listContent` accepts `limit`, `offset`, `sort`, `search`, `locale` and `filter`."),
      h2('Examples'),
      code('ts', `const page = await hydro.getPage('en', 'about');
const latest = await hydro.listNews('en', 5);

type Faq = { id: string; question: string; answer: string };
const faq = await hydro.listContent<Faq>('faq', { limit: 20, search: 'refund' });
const one = await hydro.getContent<Faq>('faq', faq.data[0].id);`),
      h2('Error handling'),
      p("Failures throw `HydroGoError` with `status`, `code`, `message` and `requestId`. Quote the request id when you report a problem."),
      code('ts', `import { HydroGoError } from '@hydrogo/client';

try {
  await hydro.getPage('en', 'missing');
} catch (err) {
  if (err instanceof HydroGoError && err.status === 404) {
    // render your 404
  } else if (err instanceof HydroGoError) {
    console.error(err.code, err.message, err.requestId);
  } else {
    throw err;
  }
}`),
      ul(
        "Requests time out after 15 seconds by default.",
        "Response bodies are size-capped to protect your server from runaway responses.",
        "Member methods never send browser cookies (`credentials: 'omit'`).",
      ),
      h2('Generating a full SDK'),
      p("The client is hand-written for the public surface. For admin tooling and every OpenAPI type, generate an SDK:"),
      code('bash', "hydro sdk generate --lang typescript --out ./sdk\nhydro sdk generate --openapi docs/api/public-content-v1.openapi.json --lang all --out ./sdk"),
      h2('Without the client'),
      p("Plain `fetch` works everywhere. Unwrap `data` yourself:"),
      code('ts', `const res = await fetch('https://cms.example.com/api/v1/news?locale=en&limit=5');
if (!res.ok) throw new Error((await res.json()).request_id);
const { data } = await res.json();`),
    ),

    page(
      'sveltekit-starter',
      'SvelteKit starter',
      "A small server-rendered site on the HydroGo API: the recommended shape for a real frontend.",
      p("`examples/sveltekit-starter` in the repository is a minimal, intentionally unstyled SvelteKit app that renders on the server with `adapter-node`. Search engines and link-preview bots receive complete HTML, and a publish in the admin is live on the next request. It contains no build step for content."),
      h2('Run it'),
      code('bash', `cd examples/sveltekit-starter
cp .env.example .env     # point HYDRO_API_ORIGIN at your HydroGo
npm install
npm run dev              # http://localhost:5173
npm run build && npm start   # production server on :3000`),
      table(
        ['Variable', 'Meaning'],
        ['`HYDRO_API_ORIGIN`', 'Where HydroGo runs. Server-side only, never sent to the browser.'],
        ['`PUBLIC_SITE_ORIGIN`', 'Public URL of this frontend, used for canonical links and the sitemap.'],
        ['`HYDRO_LOCALE`', 'Locale requested from the API (default `en`).'],
      ),
      h2('What is inside'),
      table(
        ['Route', 'Does'],
        ['`/`', 'Latest news with a short shared cache.'],
        ['`/news`, `/news/[slug]`', 'News list and article.'],
        ['`/[slug]`', 'A page by slug.'],
        ['`/sitemap.xml`', 'Generated from `listPages` and `listNews`.'],
        ['`src/lib/server/hydro.ts`', 'The shared client plus `toHttpError`, which turns API failures into proper HTTP statuses.'],
      ),
      code('ts', `// src/routes/news/[slug]/+page.server.ts
import { hydro, locale, siteOrigin, toHttpError } from '$lib/server/hydro';

export async function load({ params }) {
  try {
    const item = await hydro.getNews(locale, params.slug);
    return { item, canonical: \`\${siteOrigin}/news/\${item.slug}\` };
  } catch (err) {
    toHttpError(err);
  }
}`, '+page.server.ts'),
      code('ts', `// src/lib/server/hydro.ts
export function toHttpError(err: unknown): never {
  if (err instanceof HydroGoError) {
    if (err.status === 404) error(404, 'Not found');
    error(502, \`Content API error\${err.requestId ? \` (request \${err.requestId})\` : ''}\`);
  }
  error(502, 'Content API unreachable');
}`, 'hydro.ts'),
      p("A 404 from HydroGo becomes a real 404 for visitors and crawlers; other failures become a 502 that carries the request id."),
      tip("This is a starting point, not a theme. Replace the markup and keep the `+page.server.ts` loaders: they hold the data-access logic."),
      h2('Using another framework'),
      p("Nothing here is specific to SvelteKit. In Next.js, Nuxt, Astro, Remix or Express, fetch from server-side code on each request (or with a short revalidate window), render HTML and set cache headers as described in [Caching and webhooks](doc:frontends/caching-and-webhooks)."),
      code('ts', `// Next.js (app router): revalidate every 30 seconds
export default async function Page({ params }: { params: { slug: string } }) {
  const res = await fetch(
    \`\${process.env.HYDRO_API_ORIGIN}/api/v1/pages/by-slug?locale=en&slug=\${params.slug}\`,
    { next: { revalidate: 30 } },
  );
  const { data } = await res.json();
  return <article dangerouslySetInnerHTML={{ __html: data.body }} />;
}`),
    ),

    page(
      'seo',
      'SEO',
      "Canonical URLs, Open Graph, JSON-LD and sitemaps for a headless site.",
      p("HydroGo stores SEO fields and generates helpers, but your frontend emits the HTML. Because the starter renders on the server, crawlers see everything on first response."),
      h2('Canonical, title and description'),
      p("Pages and news carry `title`; news also has `seo_title` and `seo_description`. Prefer the SEO fields when present and always emit a canonical link based on your **public** origin, not the CMS origin."),
      code('svelte', `<svelte:head>
  <title>{data.item.seo_title ?? data.item.title}</title>
  <meta name="description" content={data.item.seo_description ?? data.item.excerpt} />
  <link rel="canonical" href={data.canonical} />
</svelte:head>`),
      h2('Open Graph and social images'),
      code('svelte', `<svelte:head>
  <meta property="og:type" content="article" />
  <meta property="og:title" content={data.item.title} />
  <meta property="og:url" content={data.canonical} />
  <meta property="og:image" content="{cms}/og/{data.item.locale}/{data.item.slug}" />
  <meta name="twitter:card" content="summary_large_image" />
</svelte:head>`),
      p("`GET /og/{locale}/{slug}` generates an Open Graph image for pages and articles. Configure fonts with `seo.fonts_dir` and the cache with `seo.og_cache_dir`."),
      h2('JSON-LD structured data'),
      code('ts', `const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'NewsArticle',
  headline: item.title,
  datePublished: item.created_at,
  dateModified: item.updated_at,
  mainEntityOfPage: canonical,
  author: item.author_name ? { '@type': 'Person', name: item.author_name } : undefined,
};
// <script type="application/ld+json">{@html JSON.stringify(jsonLd)}</script>`),
      h2('Sitemaps'),
      ul(
        "HydroGo serves `/sitemap.xml`, `/sitemap-index.xml`, `/sitemap-pages.xml` and `/sitemap-articles.xml`, built from published content.",
        "If your public site lives on another origin, generate the sitemap in your frontend from `listPages` and `listNews`, as the starter does, so the URLs match your routes.",
        "Use `updated_at` for `<lastmod>` and send `cache-control: public, max-age=0, s-maxage=300`.",
      ),
      h2('Redirects and 404s'),
      p("Fetch `GET /api/v1/site/redirects` at start-up or on a short cache and apply them in your frontend or edge. In the admin, the **Not found** tab lists URLs visitors requested that do not exist and converts them into redirects."),
      warn("Do not index preview routes or staging copies. Send `X-Robots-Tag: noindex` for them and keep `security` headers strict on the CMS origin."),
    ),

    page(
      'caching-and-webhooks',
      'Caching and webhooks',
      "Keep the API fast with shared caching, and notify other systems when content changes.",
      h2('Caching strategy'),
      p("Zero build on publish does not mean zero caching. Put a short cache between visitors and HydroGo and let freshness be a number you choose."),
      table(
        ['Layer', 'Setting', 'Effect'],
        ['Your frontend response', '`cache-control: public, max-age=0, s-maxage=30, stale-while-revalidate=60`', 'A CDN serves a response for 30 s, then refreshes in the background. Edits appear within about half a minute.'],
        ['Sitemap', '`s-maxage=300`', 'Crawlers rarely need fresher data.'],
        ['Media', 'Long cache lifetime at your CDN', 'Media objects are immutable: a replaced file gets a new id, so old URLs stay valid.'],
        ['Previews and admin', '`private, no-store`', 'Never cache.'],
      ),
      code('ts', `// SvelteKit load function
export async function load({ setHeaders }) {
  const news = await hydro.listNews('en', 10);
  setHeaders({ 'cache-control': 'public, max-age=0, s-maxage=30, stale-while-revalidate=60' });
  return { news };
}`),
      tip("Need instant updates for a specific page? Skip the cache for it, or purge the CDN from a webhook (below)."),
      h2('Webhooks'),
      p("HydroGo can POST signed JSON events to your systems when content changes, for example to purge a CDN, trigger a static rebuild or notify a chat room."),
      code('yaml', `webhooks:
  - id: frontend
    url: https://ci.example.com/hooks/hydrogo
    topics: [page.published, news.published, news.retracted]`, 'config.yaml'),
      code('bash', `# the secret never lives in YAML; it is read from the environment
HYDRO_WEBHOOK_SECRET=change-me
# shortcut for a single endpoint without any YAML
HYDRO_WEBHOOK_URL=https://ci.example.com/hooks/hydrogo
HYDRO_WEBHOOK_TOPICS=page.published,news.published,news.retracted`),
      table(
        ['Topic', 'Fires when'],
        ['`page.created`, `page.updated`, `page.published`', 'A page changes or is published.'],
        ['`news.created`, `news.updated`, `news.scheduled`, `news.published`, `news.retracted`', 'The news lifecycle.'],
        ['`user.invited`', 'A staff invitation is created (contains personal data and is only sent to endpoints with `allow_pii: true`).'],
        ['`order.created`, `tenant.provisioned`', 'Commerce and tenancy events.'],
      ),
      p("List the exact catalog your binary supports with `hydro events catalog`."),
      h3('Event payload'),
      code('json', `{
  "id": "evt_01",
  "topic": "news.published",
  "aggregate_id": "art_42",
  "payload_version": 1,
  "data": { }
}`),
      h3('Verify the signature'),
      p("Each request carries `X-Hydro-Event`, `X-Hydro-Timestamp`, `X-Hydro-Signature` and an `Idempotency-Key` (the event id). The signature is `sha256=` followed by the hex HMAC-SHA256 of `<timestamp>.<raw body>` using your secret. Reject requests with an old timestamp and de-duplicate by `Idempotency-Key`; deliveries are retried on failure."),
      code('ts', `import { createHmac, timingSafeEqual } from 'node:crypto';

export function verify(rawBody: string, headers: Headers, secret: string): boolean {
  const ts = headers.get('x-hydro-timestamp') ?? '';
  const sig = headers.get('x-hydro-signature') ?? '';
  if (Math.abs(Date.now() / 1000 - Number(ts)) > 300) return false;
  const expected = 'sha256=' + createHmac('sha256', secret).update(\`\${ts}.\${rawBody}\`).digest('hex');
  const a = Buffer.from(sig);
  const b = Buffer.from(expected);
  return a.length === b.length && timingSafeEqual(a, b);
}`),
      h2('Static frontends'),
      p("If you deliberately use a static site generator, point a webhook at your CI provider's build trigger and subscribe to `page.published`, `news.published` and `news.retracted`. This is an option for your frontend, not a requirement of HydroGo: server rendering with a short cache avoids rebuilds entirely."),
      warn("Webhook URLs must be plain `http` or `https` URLs without embedded credentials, and redirects are validated again. Use HTTPS and always verify the signature."),
    ),
  ],
};
