import type { RequestHandler } from './$types';
import { getNewsList } from '#lib';

export const GET: RequestHandler = async ({ url }) => {
  const origin = url.origin;
  const now = new Date();
  const twoDaysAgo = new Date(now.getTime() - 48 * 60 * 60 * 1000);

  let newsItems: any[] = [];
  try {
    const list = await getNewsList({ locale: 'tr', limit: 100 });
    // Filter to last 48 hours for Google News compliance
    newsItems = list.filter((item: any) => {
      const pubDate = new Date(item.created_at);
      return pubDate >= twoDaysAgo;
    });
  } catch (err) {
    console.error('Failed to fetch news for Google News sitemap:', err);
  }

  const sitemapXml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
${newsItems
  .map((item) => {
    const loc = `${origin}/${item.category || 'haber'}/${item.slug}`;
    const pubDate = new Date(item.created_at).toISOString();
    const title = (item.title || '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&apos;');

    return `  <url>
    <loc>${loc}</loc>
    <news:news>
      <news:publication>
        <news:name>HydroPost</news:name>
        <news:language>${item.locale || 'tr'}</news:language>
      </news:publication>
      <news:publication_date>${pubDate}</news:publication_date>
      <news:title>${title}</news:title>
    </news:news>
  </url>`;
  })
  .join('\n')}
</urlset>`;

  return new Response(sitemapXml, {
    headers: {
      'Content-Type': 'application/xml; charset=utf-8',
      'Cache-Control': 'public, max-age=900, s-maxage=900', // 15 min cache
    },
  });
};
