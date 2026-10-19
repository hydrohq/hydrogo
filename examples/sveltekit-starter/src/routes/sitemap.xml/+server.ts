import { hydro, siteOrigin } from '$lib/server/hydro';
import { locales, localePrefix } from '$lib/config';

const escape = (value: string) => value.replace(/&/g, '&amp;').replace(/</g, '&lt;');

export async function GET() {
  const urls: { loc: string; lastmod?: string }[] = [];
  for (const locale of locales) {
    const prefix = localePrefix(locale);
    urls.push({ loc: `${siteOrigin}${prefix}/` });
    const [pages, news] = await Promise.all([hydro.listPages(locale, 100).catch(() => []), hydro.listNews(locale, 100).catch(() => [])]);
    for (const p of pages) urls.push({ loc: `${siteOrigin}${prefix}/${p.slug}`, lastmod: p.updated_at });
    for (const n of news) urls.push({ loc: `${siteOrigin}${prefix}/news/${n.slug}`, lastmod: n.updated_at });
  }
  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.map((u) => `  <url><loc>${escape(u.loc)}</loc>${u.lastmod ? `<lastmod>${u.lastmod}</lastmod>` : ''}</url>`).join('\n')}
</urlset>`;
  return new Response(body, { headers: { 'content-type': 'application/xml', 'cache-control': 'public, max-age=0, s-maxage=300' } });
}
