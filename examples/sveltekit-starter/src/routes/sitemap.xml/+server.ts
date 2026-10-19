import { hydro, locale, siteOrigin } from '$lib/server/hydro';

const escape = (value: string) => value.replace(/&/g, '&amp;').replace(/</g, '&lt;');

export async function GET() {
  const [pages, news] = await Promise.all([hydro.listPages(locale, 100), hydro.listNews(locale, 100)]);
  const urls = [
    { loc: `${siteOrigin}/`, lastmod: '' },
    ...pages.map((p) => ({ loc: `${siteOrigin}/${p.slug}`, lastmod: p.updated_at })),
    ...news.map((n) => ({ loc: `${siteOrigin}/news/${n.slug}`, lastmod: n.updated_at }))
  ];
  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.map((u) => `  <url><loc>${escape(u.loc)}</loc>${u.lastmod ? `<lastmod>${u.lastmod}</lastmod>` : ''}</url>`).join('\n')}
</urlset>`;
  return new Response(body, { headers: { 'content-type': 'application/xml', 'cache-control': 'public, max-age=0, s-maxage=300' } });
}
