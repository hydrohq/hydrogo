// Ingestion and seeding script that pulls real news from RSS feeds into HydroGo CMS.
//
// Usage:
//   node scripts/ingest-rss.mjs                  # Ingests & publishes ~120-150 real news items across categories
//   node scripts/ingest-rss.mjs --target draft   # Ingests as drafts for admin review in HydroGo panel
//   node scripts/ingest-rss.mjs --limit 15       # Pulls 15 items per category
//   node scripts/ingest-rss.mjs --source <url>   # Pulls from custom RSS URL (e.g. IHA / custom feed)

const BASE = process.env.HYDRO_BASE || 'http://127.0.0.1:8090';
const EMAIL = process.env.HYDRO_ADMIN_EMAIL || 'admin@haberdiyari.com';
const PASSWORD = process.env.HYDRO_ADMIN_PASSWORD || 'HydroNews2026!SecureKey';

// Supported default category feeds
const RSS_FEEDS = [
  {
    category: 'Gündem',
    url: 'https://www.haberturk.com/rss/kategori/gundem.xml',
    backupUrl: 'https://www.trthaber.com/gundem_articles.rss',
  },
  {
    category: 'Ekonomi',
    url: 'https://www.haberturk.com/rss/kategori/ekonomi.xml',
    backupUrl: 'https://www.trthaber.com/ekonomi_articles.rss',
  },
  {
    category: 'Dünya',
    url: 'https://www.haberturk.com/rss/kategori/dunya.xml',
    backupUrl: 'https://www.trthaber.com/dunya_articles.rss',
  },
  {
    category: 'Teknoloji',
    url: 'https://www.haberturk.com/rss/kategori/teknoloji.xml',
    backupUrl: 'https://www.trthaber.com/bilim_teknoloji_articles.rss',
  },
  {
    category: 'Spor',
    url: 'https://www.haberturk.com/rss/kategori/spor.xml',
    backupUrl: 'https://www.trthaber.com/spor_articles.rss',
  },
  {
    category: 'Sağlık',
    url: 'https://www.haberturk.com/rss/kategori/saglik.xml',
  },
  {
    category: 'Yaşam',
    url: 'https://www.haberturk.com/rss/kategori/yasam.xml',
  },
];

let cookieJar = [];
let csrfToken = '';

function storeCookies(res) {
  const raw = res.headers.getSetCookie ? res.headers.getSetCookie() : res.headers.raw?.()['set-cookie'] || [];
  for (const c of raw) {
    const pair = c.split(';')[0];
    const name = pair.split('=')[0];
    cookieJar = cookieJar.filter((e) => !e.startsWith(`${name}=`));
    cookieJar.push(pair);
  }
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    Cookie: cookieJar.join('; '),
    ...(options.headers || {}),
  };
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes((options.method || 'GET').toUpperCase())) {
    headers['X-CSRF-Token'] = csrfToken;
  }
  const res = await fetch(`${BASE}${path}`, { ...options, headers });
  storeCookies(res);
  const text = await res.text();
  const body = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new Error(`${options.method || 'GET'} ${path} -> ${res.status}: ${JSON.stringify(body)}`);
  }
  return body;
}

async function login() {
  const csrf = await request('/csrf-token');
  csrfToken = csrf.token;
  await request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email: EMAIL, password: PASSWORD }),
  });
  console.log('✓ Logged in as HydroGo admin.');
}

function decodeHtmlEntities(str = '') {
  return str
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&rsquo;/g, "'")
    .replace(/&lsquo;/g, "'")
    .replace(/&rdquo;/g, '"')
    .replace(/&ldquo;/g, '"')
    .replace(/&nbsp;/g, ' ')
    .replace(/&ndash;/g, '–')
    .replace(/&mdash;/g, '—')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function slugify(text = '') {
  return text
    .toString()
    .toLowerCase()
    .trim()
    .replace(/ğ/g, 'g')
    .replace(/ü/g, 'u')
    .replace(/ş/g, 's')
    .replace(/ı/g, 'i')
    .replace(/ö/g, 'o')
    .replace(/ç/g, 'c')
    .replace(/&/g, 've')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 90) || 'haber';
}

function parseRssXml(xml, categoryName) {
  const items = [];
  const regex = /<item>([\s\S]*?)<\/item>/gi;
  let match;

  while ((match = regex.exec(xml)) !== null) {
    const chunk = match[1];

    const getTag = (tag) => {
      const m = chunk.match(new RegExp(`<${tag}[^>]*>(?:<!\\[CDATA\\[)?([\\s\\S]*?)(?:\\]\\]>)?<\\/${tag}>`, 'i'));
      return m ? m[1].trim() : '';
    };

    const getAttr = (tag, attr) => {
      const m = chunk.match(new RegExp(`<${tag}[^>]*\\s+${attr}=["']([^"']+)["']`, 'i'));
      return m ? m[1].trim() : '';
    };

    const rawTitle = getTag('title');
    const rawDesc = getTag('description');
    const pubDate = getTag('pubDate');
    const link = getTag('link');
    const img =
      getTag('image') ||
      getAttr('enclosure', 'url') ||
      getAttr('media:content', 'url') ||
      getAttr('media:thumbnail', 'url');

    const title = decodeHtmlEntities(rawTitle);
    const desc = decodeHtmlEntities(rawDesc);

    if (title && title.length > 5) {
      items.push({
        title,
        desc,
        img: img || '',
        pubDate,
        link,
        category: categoryName,
      });
    }
  }

  return items;
}

async function fetchRssFeed(feed, limit = 20) {
  const tryUrl = async (url) => {
    try {
      const res = await fetch(url, {
        headers: {
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
          Accept: 'application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.8',
        },
      });
      if (!res.ok) return null;
      return await res.text();
    } catch {
      return null;
    }
  };

  let xml = await tryUrl(feed.url);
  if (!xml && feed.backupUrl) {
    xml = await tryUrl(feed.backupUrl);
  }
  if (!xml) {
    console.warn(`[WARN] Could not fetch RSS for category: ${feed.category}`);
    return [];
  }

  const items = parseRssXml(xml, feed.category);
  return items.slice(0, limit);
}

function buildArticleBody(item) {
  const paragraphs = [];
  const text = item.desc || item.title;

  // Split into readable sentences or paragraphs
  const sentences = text.split(/(?<=[.!?])\s+/);
  if (sentences.length <= 2) {
    paragraphs.push(`<p>${text}</p>`);
    paragraphs.push(
      `<p>Gelişmeler İhlas Haber Ajansı (İHA) ve sahadaki muhabirlerimiz tarafından anbean takip edilmekte olup, konuyla ilgili tüm resmi açıklamalar anlık olarak güncellenmektedir.</p>`
    );
  } else {
    const mid = Math.ceil(sentences.length / 2);
    paragraphs.push(`<p>${sentences.slice(0, mid).join(' ')}</p>`);
    paragraphs.push(`<p>${sentences.slice(mid).join(' ')}</p>`);
    paragraphs.push(
      `<p>Olayla ilgili yetkili mercilerden yapılan son değerlendirmeler ve teknik raporlar doğrultusunda gelişmeler aktarılmaya devam edecektir.</p>`
    );
  }

  return paragraphs.join('\n\n');
}

function buildKeyTakeaways(item) {
  const desc = item.desc || item.title;
  const parts = desc.split(/[.,;]/).map((s) => s.trim()).filter((s) => s.length > 15);
  if (parts.length >= 2) {
    return parts.slice(0, 3).map((p) => `• ${p}`).join('\n');
  }
  return `• ${item.title}\n• İHA ve ajans kaynaklı son dakika gelişmesi\n• Ayrıntılar haber merkezimizce teyit edilmektedir`;
}

async function ingestNews({ target = 'publish', limitPerCategory = 20, customSource = null } = {}) {
  await login();

  const feedsToProcess = customSource
    ? [{ category: 'Gündem', url: customSource }]
    : RSS_FEEDS;

  console.log(`Starting RSS ingestion: target=${target}, limitPerCategory=${limitPerCategory}...`);

  let totalIngested = 0;
  let totalPublished = 0;
  let totalDrafts = 0;
  const usedSlugs = new Set();

  for (const feed of feedsToProcess) {
    console.log(`\nFetching ${feed.category} from RSS...`);
    const items = await fetchRssFeed(feed, limitPerCategory);
    console.log(`Found ${items.length} items for ${feed.category}.`);

    let catIndex = 0;
    for (const item of items) {
      catIndex++;
      let baseSlug = slugify(item.title);
      let slug = baseSlug;
      let counter = 1;
      while (usedSlugs.has(slug)) {
        slug = `${baseSlug}-${counter++}`;
      }
      usedSlugs.add(slug);

      // Check if article with this slug already exists in CMS
      try {
        const existing = await request(`/api/v1/news/tr/${slug}`);
        if (existing?.data?.id) {
          console.log(`- Exists: ${slug}`);
          continue;
        }
      } catch {
        // Not found, can create
      }

      const body = buildArticleBody(item);
      const keyTakeaways = buildKeyTakeaways(item);
      const isFeatured = catIndex === 1; // Mark the lead category article as featured

      const payload = {
        locale: 'tr',
        slug,
        title: item.title,
        body,
        cover_media_id: item.img || '',
        excerpt: item.desc ? item.desc.slice(0, 220) : item.title,
        key_takeaways: keyTakeaways,
        category: item.category,
        author_name: 'İHA / Haber Masası',
        featured: isFeatured,
        tags: `${item.category.toLowerCase()},haber,gundem,turkiye`,
        seo_title: `${item.title} — HydroPost`,
        seo_description: item.desc ? item.desc.slice(0, 150) : item.title,
      };

      try {
        const created = await request('/api/v1/admin/news', {
          method: 'POST',
          body: JSON.stringify(payload),
        });

        if (created?.data?.id) {
          totalIngested++;
          if (target === 'publish') {
            await request(`/api/v1/admin/news/${created.data.id}/publish`, {
              method: 'POST',
              body: JSON.stringify({ version: created.data.version }),
            });
            totalPublished++;
            console.log(`[PUBLISHED] (${feed.category}) ${item.title.slice(0, 45)}...`);
          } else {
            totalDrafts++;
            console.log(`[DRAFT] (${feed.category}) ${item.title.slice(0, 45)}...`);
          }
        }
      } catch (err) {
        console.warn(`[WARN] Failed to insert ${slug}:`, err.message);
      }
    }
  }

  console.log(`\n========================================`);
  console.log(`Ingestion completed!`);
  console.log(`Total Ingested: ${totalIngested}`);
  console.log(`Published:      ${totalPublished}`);
  console.log(`Drafts:         ${totalDrafts}`);
  console.log(`========================================\n`);

  return { totalIngested, totalPublished, totalDrafts };
}

// Parse CLI flags
const args = process.argv.slice(2);
let target = 'publish';
let limit = 22; // 7 categories * 22 ≈ 150 articles
let source = null;

for (let i = 0; i < args.length; i++) {
  if (args[i] === '--target' && args[i + 1]) target = args[++i];
  if (args[i] === '--limit' && args[i + 1]) limit = parseInt(args[++i], 10) || 20;
  if (args[i] === '--source' && args[i + 1]) source = args[++i];
}

ingestNews({ target, limitPerCategory: limit, customSource: source }).catch((err) => {
  console.error('Ingestion failed:', err);
  process.exit(1);
});
