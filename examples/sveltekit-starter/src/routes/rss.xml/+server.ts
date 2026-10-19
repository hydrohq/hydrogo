import { hydro, siteOrigin, singleRecord } from '$lib/server/hydro';
import { defaultLocale } from '$lib/config';

const escape = (value: string) => value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

export async function GET() {
  const [settings, news] = await Promise.all([singleRecord('site_settings'), hydro.listNews(defaultLocale, 30).catch(() => [])]);
  const name = String(settings.site_name ?? 'News');
  const items = news
    .map(
      (n) => `  <item>
    <title>${escape(n.title)}</title>
    <link>${siteOrigin}/news/${n.slug}</link>
    <guid isPermaLink="true">${siteOrigin}/news/${n.slug}</guid>
    <pubDate>${new Date(n.created_at ?? n.updated_at).toUTCString()}</pubDate>${n.excerpt ? `\n    <description>${escape(n.excerpt)}</description>` : ''}
  </item>`
    )
    .join('\n');
  const body = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
  <title>${escape(name)}</title>
  <link>${siteOrigin}/</link>
  <description>${escape(String(settings.tagline ?? name))}</description>
${items}
</channel></rss>`;
  return new Response(body, { headers: { 'content-type': 'application/rss+xml; charset=utf-8', 'cache-control': 'public, max-age=0, s-maxage=300' } });
}
