import { type DocSection, page, p, h2, h3, code, tabs, note, tip, warn, table, ul, ol } from './types';
import { restApiMore } from './rest-api-more';

export const restApi: DocSection = {
  id: 'rest-api',
  title: 'REST API',
  pages: [
    page(
      'overview',
      'API overview',
      "Base URL, the response envelope, the error shape, pagination and status codes shared by every endpoint.",
      p("The API is plain JSON over HTTP under `/api/v1`. Public endpoints return published content without credentials; admin endpoints live under `/api/v1/admin` and need one of the credentials in [Authentication](doc:rest-api/authentication)."),
      code('bash', "curl -s https://cms.example.com/api/v1/news?locale=en\\&limit=3"),
      h2('Success envelope'),
      p("Every successful JSON response wraps its payload in `data`. List endpoints add `metadata` with paging information."),
      code('json', `{
  "data": [ { "id": "…", "slug": "hello-world", "title": "Hello world" } ],
  "metadata": {
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}`),
      table(
        ['Metadata field', 'Meaning'],
        ['`limit`', 'Page size used for this response.'],
        ['`offset`', 'Records skipped (offset-paged lists).'],
        ['`total`', 'Total matching records, when it is cheap to know.'],
        ['`has_more`', '`true` when another page exists.'],
        ['`next_cursor`', 'Opaque cursor for cursor-paged lists; pass it back to continue.'],
        ['`locale`', 'The locale served (content API).'],
      ),
      h2('Errors', 'errors'),
      p("Errors use a single shape, so clients can handle them in one place. Always show or log the `request_id`: it is also sent in the `X-Request-ID` response header and lets you find the matching server log line."),
      code('json', `{
  "code": "unprocessable_entity",
  "message": "validation failed",
  "request_id": "7c2f4a1e9b3d",
  "fields": { "title": "required" }
}`),
      table(
        ['HTTP status', '`code`', 'When'],
        ['400', '`bad_request`', 'Malformed JSON, invalid query parameter (for example `limit` out of range), unknown locale.'],
        ['401', '`unauthorized`', 'Missing, expired or invalid credential.'],
        ['403', '`forbidden`', 'Authenticated but not allowed (role, permission, scope or missing CSRF token).'],
        ['404', '`not_found`', 'Unknown route, record, or a type that is not `public`.'],
        ['409', '`conflict`', 'Version conflict: someone saved a newer version first.'],
        ['413', '`request_entity_too_large`', 'Body larger than the endpoint accepts (JSON bodies default to 1 MiB).'],
        ['422', '`unprocessable_entity`', 'Validation failed. `fields` maps field ids to codes such as `required`, `invalid`, `taken`.'],
        ['429', '`too_many_requests`', 'Rate limited. Honour the `Retry-After` header (seconds).'],
        ['500 / 503', '`internal_server_error` / `service_unavailable`', 'Server problem. Report the `request_id`.'],
      ),
      h2('Pagination'),
      ul(
        "`limit` accepts 1 to 100. The default is 20 for pages and news and 25 for content types.",
        "Content types use `offset`; continue while `metadata.has_more` is `true`.",
        "Always loop on `has_more` rather than computing pages from `total`, which may be absent.",
      ),
      code('ts', `const all = [];
let offset = 0;
for (;;) {
  const res = await fetch(\`\${origin}/api/v1/content/faq?limit=100&offset=\${offset}\`);
  const body = await res.json();
  all.push(...body.data);
  if (!body.metadata.has_more) break;
  offset += body.metadata.limit;
}`),
      h2('Conventions'),
      ul(
        "Field names are `snake_case`; timestamps are RFC 3339 strings in UTC.",
        "Ids are opaque strings. Do not parse them.",
        "Query strings are strict: unsupported parameters on the content API answer `400` instead of being ignored.",
        "Browsers on another origin can read published content only if the origin is listed in `security.public_api_origins`. Server-side callers are not affected.",
      ),
      note("The machine-readable contract is the OpenAPI baseline kept in the repository under `docs/api/`. CI compares every change to it, so accidental breaking changes are caught before release. Generate a typed SDK with `hydro sdk generate`."),
    ),

    page(
      'authentication',
      'Authentication',
      "Session cookie with CSRF, Bearer tokens and API keys, how to obtain each, and which endpoints need which.",
      p("Public read endpoints need no credential. Everything under `/api/v1/admin` works with **any of three credentials** through the same middleware, so a script can do what the admin panel does."),
      table(
        ['Credential', 'Header or cookie', 'Best for'],
        ['Session', 'Cookie `hydro_session` plus `X-CSRF-Token` on unsafe methods', 'The admin SPA and same-origin browser code.'],
        ['Bearer token', '`Authorization: Bearer hyd_…`', 'Personal scripts and tooling acting as you.'],
        ['API key', '`X-API-Key: hyd_…`', 'Frontends, SSR servers and CI. Read-only.'],
      ),
      h2('Session cookie and CSRF'),
      p("Sign in with e-mail, password and, if enabled, a TOTP code. The server sets an HTTP-only `hydro_session` cookie and a `hydro_csrf` cookie. For every `POST`, `PUT`, `PATCH` and `DELETE`, echo the CSRF token in the `X-CSRF-Token` header. The token is returned by `GET /csrf-token` (`{\"token\": \"…\"}`) and by `GET /api/v1/admin/me`."),
      tabs(
        {
          label: 'curl',
          lang: 'bash',
          code: `# 1. sign in and keep the cookies
curl -c jar.txt -X POST https://cms.example.com/auth/login \\
  -H 'Content-Type: application/json' \\
  -d '{"email":"you@example.com","password":"…","mfa_code":""}'

# 2. fetch the CSRF token
TOKEN=$(curl -s -b jar.txt -c jar.txt https://cms.example.com/csrf-token | jq -r .token)

# 3. make a state-changing request
curl -b jar.txt -X POST https://cms.example.com/api/v1/admin/pages \\
  -H "X-CSRF-Token: $TOKEN" -H 'Content-Type: application/json' \\
  -d '{ … }'`,
        },
        {
          label: 'TypeScript',
          lang: 'ts',
          code: `// In the browser the cookies are sent automatically.
const { token } = await (await fetch('/csrf-token', { credentials: 'same-origin' })).json();

await fetch('/api/v1/admin/pages', {
  method: 'POST',
  credentials: 'same-origin',
  headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token },
  body: JSON.stringify({ /* … */ }),
});`,
        },
      ),
      h2('Bearer tokens (personal)'),
      p("A personal token acts as its owner with admin-level API access. Create one in the admin under **My Account** or via the API from a browser session (tokens cannot mint tokens). The plaintext token is shown **once**."),
      code('bash', `curl -X POST https://cms.example.com/api/v1/admin/account/tokens \\
  -b jar.txt -H "X-CSRF-Token: $TOKEN" -H 'Content-Type: application/json' \\
  -d '{"name":"deploy script","expires_in_days":90}'
# -> { "data": { "token": "hyd_…", "item": { "id": "…", "prefix": "hyd_ab12" } } }

curl https://cms.example.com/api/v1/admin/me -H 'Authorization: Bearer hyd_…'`),
      p("`expires_in_days` is optional (omit or `0` for no expiry, maximum 3650). List with `GET /api/v1/admin/account/tokens` and revoke with `DELETE /api/v1/admin/account/tokens/{id}`."),
      h2('API keys'),
      p("API keys are for machines. They are **read-only** and carry one scope. Only administrators create them (`POST /api/v1/admin/api-keys`, also available in the admin)."),
      table(
        ['Scope', 'Allows'],
        ['`read`', 'Read endpoints.'],
        ['`preview`', 'Read endpoints, and viewing unpublished content once handlers honour it (draft preview).'],
      ),
      code('bash', `curl -X POST https://cms.example.com/api/v1/admin/api-keys \\
  -b jar.txt -H "X-CSRF-Token: $TOKEN" -H 'Content-Type: application/json' \\
  -d '{"name":"frontend ssr","scope":"read","expires_in_days":365}'

curl https://cms.example.com/api/v1/admin/me -H 'X-API-Key: hyd_…'`),
      note("A HydroGo token header (`Authorization: Bearer hyd_…` or `X-API-Key`) is authoritative: when one is present the session cookie is ignored. Because these requests do not rely on cookies, they do not need a CSRF token."),
      h2('Staff mobile tokens'),
      p("Native apps can sign staff in with `POST /api/v1/mobile/auth/login` (`email`, `password`, optional `mfa_code`, `device_id`, `device_name`, `platform`) and renew with `/api/v1/mobile/auth/refresh`. `revoke` and `revoke-all` end one or all device sessions. Members use the parallel `/api/v1/mobile/member/auth/*` family, see [Member API](doc:rest-api/members-api)."),
      warn("Never ship an API key or token in browser code. Use them from a server (SSR loader, serverless function, build step) and keep them in environment variables."),
      h2('Credential cheat sheet'),
      table(
        ['I want to…', 'Use'],
        ['Read published content in my SSR app', 'No credential needed; an API key `read` is optional and lets you separate traffic.'],
        ['Preview drafts in my frontend', 'API key `preview` or a signed preview link. See [Draft previews](doc:rest-api/previews).'],
        ['Script imports and edits', 'Personal Bearer token.'],
        ['Let the admin SPA work', 'Session cookie plus CSRF (automatic).'],
      ),
    ),

    page(
      'pages-and-news',
      'Pages and news',
      "List and read published pages and news articles, with curl and @hydrogo/client examples.",
      h2('Pages'),
      table(
        ['Method and path', 'Description'],
        ['`GET /api/v1/pages?locale=en&limit=20`', 'List published page summaries. `limit` is 1 to 100.'],
        ['`GET /api/v1/pages/by-slug?locale=en&slug=about`', 'One published page, with query-string parameters.'],
        ['`GET /api/v1/pages/{locale}/{slug}`', 'The same page with path segments.'],
      ),
      tabs(
        { label: 'curl', lang: 'bash', code: "curl 'https://cms.example.com/api/v1/pages/by-slug?locale=en&slug=about'" },
        {
          label: '@hydrogo/client',
          lang: 'ts',
          code: `import { createHydroGoClient } from '@hydrogo/client';

const hydro = createHydroGoClient({ origin: 'https://cms.example.com' });
const about = await hydro.getPage('en', 'about');
const summaries = await hydro.listPages('en', 100);`,
        },
      ),
      code('json', `{
  "data": {
    "id": "pg_01",
    "locale": "en",
    "slug": "about",
    "title": "About us",
    "body": "<p>…</p>",
    "blocks": [ { "type": "hero", "payload": { "heading": "Hello" } } ],
    "status": "published",
    "version": 7,
    "revision": 12,
    "published_revision": 12,
    "publication_generation": 3,
    "created_at": "2026-01-10T09:00:00Z",
    "updated_at": "2026-02-02T14:30:00Z"
  }
}`),
      p("List responses contain summaries only (`id`, `locale`, `slug`, `title`, `revision`, `publication_generation`, `updated_at`), which makes them cheap to use for navigation and sitemaps. Fetch the full page by slug."),
      h2('News'),
      table(
        ['Method and path', 'Description'],
        ['`GET /api/v1/news?locale=en&limit=20&category=product`', 'List published articles. `locale` is required; `category` is optional.'],
        ['`GET /api/v1/news/by-slug?locale=en&slug=launch`', 'One published article.'],
        ['`GET /api/v1/news/{locale}/{slug}`', 'The same article with path segments.'],
        ['`POST /api/v1/news/views`, `GET /api/v1/news/views`', 'Record and read article view counts.'],
      ),
      tabs(
        { label: 'curl', lang: 'bash', code: "curl 'https://cms.example.com/api/v1/news?locale=en&limit=5&category=product'" },
        {
          label: '@hydrogo/client',
          lang: 'ts',
          code: `const latest = await hydro.listNews('en', 5);
const article = await hydro.getNews('en', 'launch');
console.log(article.title, article.updated_at);`,
        },
      ),
      p("An article carries `id`, `locale`, `slug`, `title`, `body` and, when set, `excerpt`, `category`, `author_name`, `tags`, `cover_media_id`, `seo_title`, `seo_description`, `featured`, `scheduled_at`, `expires_at`, plus `revision` and `updated_at`."),
      h2('Scheduled publishing and retraction'),
      p("News supports scheduled publishing (`scheduled_at`) and expiry (`expires_at`). Articles appear in the API when their schedule arrives and disappear when they are retracted or expire, without any rebuild. Each of these events can fire a [webhook](doc:frontends/caching-and-webhooks)."),
      tip("Use `GET /api/v1/pages?locale=…` and `GET /api/v1/news?locale=…` to generate your sitemap and navigation. They return `updated_at` for `<lastmod>`."),
      h2('Errors you will see'),
      table(
        ['Case', 'Status'],
        ['Unknown slug or unpublished item', '404'],
        ['Missing `locale` on news list', '400'],
        ['`limit` outside 1 to 100', '400'],
      ),
    ),

    page(
      'content-api',
      'Content API',
      "Read published records of your YAML content types, with search, filters, sorting and locale.",
      p("Every content type with `public: true` is readable without credentials. Types without it, and unknown types, both answer `404`."),
      table(
        ['Method and path', 'Description'],
        ['`GET /api/v1/content/{type}`', 'List published records.'],
        ['`GET /api/v1/content/{type}/{id}`', 'One published record. A draft id answers 404.'],
      ),
      h2('Query parameters'),
      table(
        ['Parameter', 'Meaning'],
        ['`limit`', '1 to 100, default 25.'],
        ['`offset`', 'Records to skip, default 0.'],
        ['`sort`', 'A field id, or `created_at` / `updated_at`. Prefix with `-` for descending: `sort=-updated_at`.'],
        ['`search`', 'Case-insensitive contains match across all fields marked `searchable`. If the type has none, the request answers 400.'],
        ['`filter[field]`', 'Exact match on a field marked `filterable`, for example `filter[category]=billing`. Several filters combine with AND.'],
        ['`locale`', 'A configured site locale. Localized fields return that locale and fall back to the default per field. Unknown locale answers 400.'],
      ),
      note("Each parameter may appear once. Unsupported parameters, repeated parameters and filters on fields that are not `filterable` answer 400 with a message naming the problem."),
      h2('Examples'),
      tabs(
        {
          label: 'curl',
          lang: 'bash',
          code: `# newest FAQ entries about refunds in the billing category
curl -G 'https://cms.example.com/api/v1/content/faq' \\
  --data-urlencode 'search=refund' \\
  --data-urlencode 'filter[category]=billing' \\
  --data-urlencode 'sort=-updated_at' \\
  --data-urlencode 'limit=10'`,
        },
        {
          label: '@hydrogo/client',
          lang: 'ts',
          code: `type Faq = { id: string; question: string; answer: string; category: string };

const res = await hydro.listContent<Faq>('faq', {
  search: 'refund',
  filter: { category: 'billing' },
  sort: '-updated_at',
  limit: 10,
  locale: 'en',
});

res.data.forEach((f) => console.log(f.question));
const one = await hydro.getContent<Faq>('faq', res.data[0].id, 'en');`,
        },
      ),
      code('json', `{
  "data": [
    {
      "id": "rec_8f2",
      "question": "How do refunds work?",
      "answer": "<p>…</p>",
      "category": "billing",
      "created_at": "2026-03-01T10:00:00Z",
      "updated_at": "2026-03-04T08:12:00Z",
      "locale": "en"
    }
  ],
  "metadata": { "limit": 10, "offset": 0, "has_more": false, "locale": "en" }
}`),
      h2('Notes'),
      ul(
        "Rich text fields are returned as HTML ready to render.",
        "Each record also contains `id`, `created_at`, `updated_at` and, when localization is active, `locale`.",
        "Search, filters and sorting always use the default locale's values, even when `locale` asks for another.",
        "Types with `drafts: true` only expose published records; the list is filtered server-side.",
      ),
      warn("Keep secrets and private data out of `public: true` types. Everything in a public type is world-readable once published."),
    ),

    ...restApiMore,
  ],
};
