import type { RequestHandler } from './$types';
import { getNewsList, getSiteSettings } from '#lib';

function xmlEscape(value: string): string {
  return (value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

export const GET: RequestHandler = async ({ url }) => {
  const origin = url.origin;

  let newsItems: any[] = [];
  let siteName = 'HydroPost';
  let tagline = '';
  try {
    const [list, settings] = await Promise.all([getNewsList({ limit: 50 }), getSiteSettings()]);
    newsItems = list || [];
    siteName = settings?.site_name || siteName;
    tagline = settings?.tagline || '';
  } catch (err) {
    console.error('Failed to build rss.xml:', err);
  }

  const items = newsItems
    .map((item) => {
      const link = `${origin}/${item.category || 'haber'}/${item.slug}`;
      const pubDate = new Date(item.created_at).toUTCString();
      return `    <item>
      <title>${xmlEscape(item.title)}</title>
      <link>${link}</link>
      <guid isPermaLink="true">${link}</guid>
      <pubDate>${pubDate}</pubDate>
      ${item.category ? `<category>${xmlEscape(item.category)}</category>` : ''}
      <description>${xmlEscape(item.excerpt || '')}</description>
    </item>`;
    })
    .join('\n');

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>${xmlEscape(siteName)}</title>
    <link>${origin}</link>
    <description>${xmlEscape(tagline)}</description>
    <language>tr</language>
${items}
  </channel>
</rss>`;

  return new Response(body, {
    headers: {
      'Content-Type': 'application/rss+xml; charset=utf-8',
      'Cache-Control': 'public, max-age=900, s-maxage=900',
    },
  });
};
