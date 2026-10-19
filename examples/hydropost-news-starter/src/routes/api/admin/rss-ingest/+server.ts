import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const RSS_FEEDS = [
  { category: 'Gündem', url: 'https://www.haberturk.com/rss/kategori/gundem.xml', backupUrl: 'https://www.trthaber.com/gundem_articles.rss' },
  { category: 'Ekonomi', url: 'https://www.haberturk.com/rss/kategori/ekonomi.xml', backupUrl: 'https://www.trthaber.com/ekonomi_articles.rss' },
  { category: 'Dünya', url: 'https://www.haberturk.com/rss/kategori/dunya.xml', backupUrl: 'https://www.trthaber.com/dunya_articles.rss' },
  { category: 'Teknoloji', url: 'https://www.haberturk.com/rss/kategori/teknoloji.xml', backupUrl: 'https://www.trthaber.com/bilim_teknoloji_articles.rss' },
  { category: 'Spor', url: 'https://www.haberturk.com/rss/kategori/spor.xml', backupUrl: 'https://www.trthaber.com/spor_articles.rss' },
  { category: 'Sağlık', url: 'https://www.haberturk.com/rss/kategori/saglik.xml' },
  { category: 'Yaşam', url: 'https://www.haberturk.com/rss/kategori/yasam.xml' },
];

function decodeEntities(str: string): string {
  return str
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/ğ/g, 'g')
    .replace(/ü/g, 'u')
    .replace(/ş/g, 's')
    .replace(/ı/g, 'i')
    .replace(/ö/g, 'o')
    .replace(/ç/g, 'c')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 90) || 'haber';
}

export const POST: RequestHandler = async ({ request }) => {
  const body = await request.json().catch(() => ({}));
  const target = body.target === 'draft' ? 'draft' : 'publish';
  const limit = Math.min(30, Math.max(1, Number(body.limit) || 10));
  const categoryFilter = body.category ? String(body.category).toLowerCase() : null;

  const ingestToken = process.env.RSS_INGEST_TOKEN;
  const authHeader = request.headers.get('Authorization') || request.headers.get('X-Ingest-Token');
  if (ingestToken && authHeader !== `Bearer ${ingestToken}` && authHeader !== ingestToken) {
    return json({ error: 'Unauthorized: missing or invalid ingest token' }, { status: 401 });
  }

  const hydroBase = process.env.HYDROGO_API_URL || 'http://127.0.0.1:8090';
  const email = process.env.HYDRO_ADMIN_EMAIL;
  const password = process.env.HYDRO_ADMIN_PASSWORD;

  if (!email || !password) {
    return json({ error: 'Admin credentials not configured in environment' }, { status: 500 });
  }

  // 1. Authenticate to HydroGo
  let cookieHeader = '';
  let csrfToken = '';

  try {
    const csrfRes = await fetch(`${hydroBase}/csrf-token`);
    const csrfData = await csrfRes.json();
    csrfToken = csrfData.token;

    const setCookies = csrfRes.headers.getSetCookie ? csrfRes.headers.getSetCookie() : [];
    cookieHeader = setCookies.map((c) => c.split(';')[0]).join('; ');

    const loginRes = await fetch(`${hydroBase}/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
        Cookie: cookieHeader,
      },
      body: JSON.stringify({ email, password }),
    });

    if (!loginRes.ok) {
      return json({ error: 'HydroGo authentication failed' }, { status: 502 });
    }

    const loginCookies = loginRes.headers.getSetCookie ? loginRes.headers.getSetCookie() : [];
    cookieHeader = [...setCookies, ...loginCookies].map((c) => c.split(';')[0]).join('; ');
  } catch (err: any) {
    return json({ error: `Connection failed: ${err.message}` }, { status: 502 });
  }

  // 2. Fetch and ingest RSS feeds
  const feeds = categoryFilter
    ? RSS_FEEDS.filter((f) => f.category.toLowerCase() === categoryFilter)
    : RSS_FEEDS;

  let ingested = 0;
  let published = 0;
  let drafts = 0;

  for (const feed of feeds) {
    try {
      let xml = '';
      const res = await fetch(feed.url, {
        headers: { 'User-Agent': 'Mozilla/5.0' },
      });
      if (res.ok) {
        xml = await res.text();
      } else if (feed.backupUrl) {
        const bRes = await fetch(feed.backupUrl, { headers: { 'User-Agent': 'Mozilla/5.0' } });
        if (bRes.ok) xml = await bRes.text();
      }

      if (!xml) continue;

      const itemRegex = /<item>([\s\S]*?)<\/item>/gi;
      let count = 0;
      let match;

      while ((match = itemRegex.exec(xml)) !== null && count < limit) {
        const chunk = match[1];
        const titleMatch = chunk.match(/<title[^>]*>(?:<!\[CDATA\[)?([\s\S]*?)(?:\]\]>)?<\/title>/i);
        const descMatch = chunk.match(/<description[^>]*>(?:<!\[CDATA\[)?([\s\S]*?)(?:\]\]>)?<\/description>/i);
        const imgMatch =
          chunk.match(/<image[^>]*>(?:<!\[CDATA\[)?([\s\S]*?)(?:\]\]>)?<\/image>/i) ||
          chunk.match(/url=["']([^"']+\.(?:jpg|jpeg|png|webp)[^"']*)["']/i);

        const title = titleMatch ? decodeEntities(titleMatch[1]) : '';
        const desc = descMatch ? decodeEntities(descMatch[1]) : '';
        const img = imgMatch ? imgMatch[1].trim() : '';

        if (!title || title.length < 5) continue;
        count++;

        const slug = slugify(title);

        const articlePayload = {
          locale: 'tr',
          slug,
          title,
          body: `<p>${desc}</p>\n\n<p>İhlas Haber Ajansı (İHA) ve ajans haber kaynaklarından derlenen son dakika gelişmeleriyle birlikte resmi açıklamalar anbean haber merkezimiz tarafından takip edilmektedir.</p>`,
          cover_media_id: img,
          excerpt: desc.slice(0, 220),
          key_takeaways: `• ${title}\n• İHA ve ajans kaynaklı son durum\n• Gelişmeler takip ediliyor`,
          category: feed.category,
          author_name: 'İHA / Haber Masası',
          featured: count === 1,
          tags: `${feed.category.toLowerCase()},haber,turkiye`,
          seo_title: `${title} — HydroPost`,
          seo_description: desc.slice(0, 150),
        };

        const createRes = await fetch(`${hydroBase}/api/v1/admin/news`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': csrfToken,
            Cookie: cookieHeader,
          },
          body: JSON.stringify(articlePayload),
        });

        if (createRes.ok) {
          const created = await createRes.json();
          ingested++;
          if (target === 'publish' && created?.data?.id) {
            await fetch(`${hydroBase}/api/v1/admin/news/${created.data.id}/publish`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken,
                Cookie: cookieHeader,
              },
              body: JSON.stringify({ version: created.data.version }),
            });
            published++;
          } else {
            drafts++;
          }
        }
      }
    } catch {
      // Continue to next feed
    }
  }

  return json({
    success: true,
    totalIngested: ingested,
    published,
    drafts,
    target,
  });
};
