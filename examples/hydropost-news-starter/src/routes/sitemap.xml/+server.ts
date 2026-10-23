import type { RequestHandler } from './$types';
import { getNewsList, getSiteNavigation } from '#lib';

function xmlEscape(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

const STATIC_PATHS = ['/', '/son-dakika', '/iletisim', '/kunye', '/resmi-ilanlar', '/yazarlar', '/ara'];

export const GET: RequestHandler = async ({ url }) => {
  const origin = url.origin;

  let headerNav: Array<{ href: string }> = [];
  let newsItems: any[] = [];
  try {
    [headerNav, newsItems] = await Promise.all([
      getSiteNavigation('header'),
      getNewsList({ limit: 200 }),
    ]);
  } catch (err) {
    console.error('Failed to build sitemap.xml:', err);
  }

  const categoryPaths = Array.from(new Set(headerNav.map((item) => item.href))).filter(Boolean);

  const urls: Array<{ loc: string; lastmod?: string; priority: string }> = [
    ...STATIC_PATHS.map((path) => ({ loc: `${origin}${path}`, priority: path === '/' ? '1.0' : '0.6' })),
    ...categoryPaths.map((href) => ({ loc: `${origin}${href}`, priority: '0.8' })),
    ...newsItems.map((item) => ({
      loc: `${origin}/${item.category || 'haber'}/${item.slug}`,
      lastmod: item.updated_at || item.created_at,
      priority: '0.7',
    })),
  ];

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls
  .map(
    (u) => `  <url>
    <loc>${xmlEscape(u.loc)}</loc>${u.lastmod ? `\n    <lastmod>${new Date(u.lastmod).toISOString()}</lastmod>` : ''}
    <priority>${u.priority}</priority>
  </url>`
  )
  .join('\n')}
</urlset>`;

  return new Response(body, {
    headers: {
      'Content-Type': 'application/xml; charset=utf-8',
      'Cache-Control': 'public, max-age=1800, s-maxage=1800',
    },
  });
};
