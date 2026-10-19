import { siteOrigin } from '$lib/server/hydro';

export function GET() {
  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${siteOrigin}/sitemap.xml\n`, { headers: { 'content-type': 'text/plain' } });
}
