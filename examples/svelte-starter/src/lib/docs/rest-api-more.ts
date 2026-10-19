import { type DocPage, page, p, h2, h3, code, tabs, note, tip, warn, table, ul } from './types';

export const restApiMore: DocPage[] = [
  page(
    'public-endpoints',
    'Other public endpoints',
    "Media, site settings, navigation, redirects, forms, search, sitemaps, Open Graph images and operational endpoints that need no credentials.",
    h2('Media'),
    table(
      ['Path', 'Description'],
      ['`GET /media/{id}`', 'Serve a public media item or one of its derivatives (WebP, thumbnails).'],
    ),
    p("Media is referenced by id from `media` fields and from page and news content. Use `media.cdn_base_url` to serve files from a CDN origin, and `media.hotlink_protection` to stop other sites embedding them."),
    h2('Site'),
    table(
      ['Path', 'Description'],
      ['`GET /api/v1/site/settings`', 'Public site settings such as business profile and social links.'],
      ['`GET /api/v1/site/navigation/{locale}`', 'Navigation menus for a locale.'],
      ['`GET /api/v1/site/redirects`', 'Configured redirects so your frontend can honour them.'],
    ),
    h2('Forms'),
    p("Forms are defined in the admin (**Site > Forms**) and submissions land in the **Inbox**."),
    table(
      ['Path', 'Description'],
      ['`GET /api/v1/forms/{form}`', 'The form definition (fields and labels).'],
      ['`POST /api/v1/forms/{form}/submit`', 'Submit a form. Supports Cloudflare Turnstile when its keys are configured.'],
      ['`POST /api/v1/forms/{form}/view`', 'Record that a form was displayed.'],
    ),
    h2('Search'),
    table(
      ['Path', 'Description'],
      ['`GET /api/v1/search/autocomplete`', 'Suggestions across published content (needs the search capability to be enabled).'],
    ),
    h2('SEO helpers'),
    table(
      ['Path', 'Description'],
      ['`GET /sitemap.xml`, `/sitemap-index.xml`, `/sitemap-pages.xml`, `/sitemap-articles.xml`', 'XML sitemaps built from published content.'],
      ['`GET /og/{locale}/{slug}`', 'Generated Open Graph image for a page or article.'],
    ),
    h2('Operations'),
    table(
      ['Path', 'Description'],
      ['`GET /healthz`', 'Liveness: the process is up.'],
      ['`GET /readyz`', 'Readiness: the process can serve (database reachable).'],
      ['`GET /metrics`', 'Prometheus metrics.'],
      ['`GET /csrf-token`', 'The current CSRF token as JSON.'],
    ),
    tip("Wire `/readyz` into your load balancer or orchestrator, and `/healthz` into restart policies. See [Monitoring](doc:deployment/monitoring)."),
  ),

  page(
    'members-api',
    'Member API',
    "Sign-up, sign-in and profile endpoints for the visitors of your site, designed for mobile apps and token-based clients.",
    p("Members are visitor accounts, separate from admin users. The token API lives under `/api/v1/mobile/member` and never uses browser cookies (the client sends `credentials: omit`). It needs the `members` module."),
    table(
      ['Method and path', 'Description'],
      ['`POST /api/v1/mobile/member/auth/register`', 'Create an account; returns a token pair.'],
      ['`POST /api/v1/mobile/member/auth/login`', 'Sign in; returns a token pair.'],
      ['`POST /api/v1/mobile/member/auth/refresh`', 'Exchange a refresh token for a new pair.'],
      ['`POST /api/v1/mobile/member/auth/revoke`', 'Sign out this device.'],
      ['`POST /api/v1/mobile/member/auth/revoke-all`', 'Sign out every device.'],
      ['`GET /api/v1/mobile/member/auth/devices`', 'List signed-in devices.'],
      ['`GET /api/v1/mobile/member/me`', 'The member profile.'],
      ['`POST` / `DELETE /api/v1/mobile/member/push/subscription`', 'Register or remove a push subscription.'],
    ),
    h2('Sign in'),
    tabs(
      {
        label: 'curl',
        lang: 'bash',
        code: `curl -X POST https://cms.example.com/api/v1/mobile/member/auth/login \\
  -H 'Content-Type: application/json' \\
  -d '{"email":"reader@example.com","password":"…","device_id":"phone-1","device_name":"Pixel","platform":"android"}'`,
      },
      {
        label: '@hydrogo/client',
        lang: 'ts',
        code: `const tokens = await hydro.memberLogin({
  email: 'reader@example.com',
  password: '…',
  device_id: 'phone-1',
  device_name: 'Pixel',
  platform: 'android',
});

const me = await hydro.memberMe(tokens.access_token);
const renewed = await hydro.memberRefresh(tokens.refresh_token);
await hydro.memberLogout(tokens.access_token, /* allDevices */ false);`,
      },
    ),
    code('json', `{
  "access_token": "…",
  "refresh_token": "…",
  "token_type": "Bearer",
  "expires_in": 900,
  "expires_at": "2026-04-01T12:15:00Z"
}`),
    note("Member endpoints return their own documents (token pair, profile) directly rather than inside `data`. @hydrogo/client already handles each shape for you."),
    h2('Authenticated calls'),
    code('bash', "curl https://cms.example.com/api/v1/mobile/member/me -H 'Authorization: Bearer <access_token>'"),
    p("Browser-based member sign-in (cookie sessions) and OAuth routes exist at `/member/login`, `/member/register` and `/member/oauth/{provider}/start`; they are meant for same-origin apps. The mobile OpenAPI contracts are stored in the repository under `docs/api/`."),
  ),

  page(
    'admin-api',
    'Admin API',
    "A map of the /api/v1/admin endpoints. The admin panel is a client of this API, so anything it does you can script.",
    p("All endpoints below require a credential (see [Authentication](doc:rest-api/authentication)), respect roles and permissions, and use the same envelope and error format as the public API. Writes use optimistic concurrency: send the record's current `version` and expect `409` if it changed."),
    warn("This page is an orientation, not the contract. The authoritative description is the OpenAPI document: read `docs/api/` in the repository or generate a client with `hydro sdk generate`."),
    h2('Identity and account'),
    table(
      ['Path', 'Description'],
      ['`GET /api/v1/admin/me`', 'Current user, role, permissions, permission catalog, CSRF token.'],
      ['`PUT /api/v1/admin/account/locale`', 'Set the signed-in user\'s admin interface language.'],
      ['`GET` / `POST /api/v1/admin/account/tokens`', 'List or create personal tokens. `DELETE .../{id}` revokes.'],
      ['`POST /api/v1/admin/account/mfa/enroll`, `enable`, `disable`', 'Manage TOTP for your own account.'],
      ['`GET /api/v1/admin/i18n/{locale}`', 'UI translations for the admin panel (`en`, `tr`).'],
    ),
    h2('Content'),
    table(
      ['Path', 'Description'],
      ['`/api/v1/admin/pages`', 'List, create, bulk actions, import, read, update, `publish`, `translate`, revisions (`/{id}/revisions`, `.../restore`), presence and `preview-link`.'],
      ['`/api/v1/admin/news`', 'List, create, read, update, taxonomy (categories and tags).'],
      ['`/api/v1/admin/media`', 'Upload, list, edit and delete media.'],
      ['`/api/v1/admin/members`', 'Member management.'],
      ['`/api/v1/admin/content-types`', 'List, create and delete content type definitions.'],
      ['`/api/v1/admin/content/{type}`', 'Records of a content type: list with `search`, `sort`, `filter[f]`, `status` and `locale`; create; read; update; delete; `workflow`; revisions; per-locale values.'],
    ),
    code('bash', `# publish a page with a Bearer token
curl -X POST https://cms.example.com/api/v1/admin/pages/PAGE_ID/publish \\
  -H 'Authorization: Bearer hyd_…'

# list drafts of a content type
curl 'https://cms.example.com/api/v1/admin/content/faq?status=draft&limit=50' \\
  -H 'Authorization: Bearer hyd_…'`),
    h2('Site'),
    table(
      ['Path', 'Description'],
      ['`/api/v1/admin/site/settings`, `business-profile`, `social-links`, `maintenance`, `staff-notice`', 'Read with `GET`, change with `PUT`.'],
      ['`/api/v1/admin/site/navigation/{locale}`', 'Menus per locale.'],
      ['`/api/v1/admin/site/redirects`', 'CRUD plus `POST .../import`, and `site/not-found` to turn 404s into redirects.'],
      ['`/api/v1/admin/site/forms`, `site/inbox`', 'Forms and submissions, with CSV and XLSX export.'],
      ['`/api/v1/admin/site/translations/{locale}`', 'Site-level translation strings.'],
    ),
    h2('People and security'),
    table(
      ['Path', 'Description'],
      ['`/api/v1/admin/users`', 'List and read users; `POST .../invites`, `.../{id}/role`, `.../permissions`, `.../disable`, `.../password-reset-link`.'],
      ['`/api/v1/admin/api-keys`', 'List, create and revoke API keys.'],
      ['`/api/v1/admin/audit`', 'Search the audit trail.'],
      ['`/api/v1/admin/security/ip-bans`', 'List, add and remove IP bans.'],
    ),
    h2('System'),
    table(
      ['Path', 'Description'],
      ['`GET /api/v1/admin/dashboard`', 'Counts and recent activity for the dashboard.'],
      ['`GET /api/v1/admin/system/health`, `info`, `metrics`', 'Runtime status.'],
      ['`/api/v1/admin/backups`', 'List and trigger backups.'],
      ['`GET /api/v1/admin/search`', 'Global admin search.'],
      ['`GET /api/v1/admin/reports`, `notifications`, `schema`, `permissions`', 'Reports, in-app notifications, the schema that drives generic screens, the permission catalog.'],
    ),
  ),

  page(
    'previews',
    'Draft previews',
    "Show unpublished pages and articles in your own frontend with signed, short-lived preview links.",
    p("Editors often need to see a draft inside the real site before they publish. HydroGo issues **revision-bound, signed preview tokens**; your frontend uses them to fetch the draft."),
    h2('How it works'),
    ul(
      "An editor asks the admin for a preview link: `POST /api/v1/admin/pages/{id}/preview-link`.",
      "The response contains a signed token bound to that revision.",
      "Your frontend calls `GET /api/v1/pages/{id}/preview?token=<token>` (news has the matching `/api/v1/news/{id}/preview`) from the server side and renders the draft.",
      "Responses carry `Cache-Control: private, no-store`, so previews are never cached by a CDN.",
    ),
    code('ts', `// SvelteKit: src/routes/preview/[id]/+page.server.ts
import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';

export async function load({ params, url }) {
  const token = url.searchParams.get('token');
  if (!token) error(400, 'Missing preview token');
  const res = await fetch(
    \`\${env.HYDRO_API_ORIGIN}/api/v1/pages/\${params.id}/preview?token=\${encodeURIComponent(token)}\`,
  );
  if (!res.ok) error(res.status, 'Preview unavailable');
  const { data } = await res.json();
  return { page: data };
}`),
    warn("Set `security.preview_signing_key` (at least 32 bytes, or `HYDRO_PREVIEW_SIGNING_KEY`) in production so preview links survive restarts and cannot be forged. Do not index preview routes: serve them with `X-Robots-Tag: noindex`."),
    tip("API keys with the `preview` scope are the other way to read unpublished content from a trusted server."),
  ),
];
