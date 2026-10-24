import { siteOrigin } from '$lib/hydro';

// Generated once at build time (static adapter) -- see docs/decisions on
// why this starter doesn't run a per-request server. A content change
// never needs this to be regenerated; a locale/domain change does.
export const prerender = true;

export function GET() {
  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${siteOrigin}/sitemap.xml\n`, { headers: { 'content-type': 'text/plain' } });
}
