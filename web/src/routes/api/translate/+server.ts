import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const HYDROGO_API_URL = process.env.HYDROGO_API_URL || 'http://127.0.0.1:8090';
const HYDROGO_ADMIN_API_KEY = process.env.HYDROGO_ADMIN_API_KEY;
const ANTHROPIC_API_KEY = process.env.ANTHROPIC_API_KEY;

export const POST: RequestHandler = async ({ request }) => {
  if (!HYDROGO_ADMIN_API_KEY) {
    return json({ error: 'Server misconfiguration: HYDROGO_ADMIN_API_KEY is not configured' }, { status: 500 });
  }

  const body = await request.json().catch(() => null);
  if (!body || typeof body.id !== 'string' || !body.id.trim()) {
    return json({ error: 'Valid news item ID is required' }, { status: 400 });
  }

  const id = body.id.trim();
  const rawLocale = typeof body.targetLocale === 'string' ? body.targetLocale.trim().toLowerCase() : 'en';
  const targetLocale = /^[a-z]{2,5}(-[a-z0-9]{2,5})?$/.test(rawLocale) ? rawLocale : 'en';

  try {
    const adminHeaders = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${HYDROGO_ADMIN_API_KEY}`,
      'X-API-Key': HYDROGO_ADMIN_API_KEY,
    };

    // 1. Fetch source news item from HydroGo admin API
    const itemRes = await fetch(`${HYDROGO_API_URL}/api/v1/admin/news/${id}`, {
      headers: adminHeaders,
    });

    if (!itemRes.ok) {
      return json({ error: 'Article not found or admin unauthorized' }, { status: itemRes.status });
    }

    const { data: item } = await itemRes.json();

    // 2. Perform AI Translation (Claude API or fallback)
    let translatedTitle = item.title;
    let translatedExcerpt = item.excerpt || '';
    let translatedBody = item.body || '';

    if (ANTHROPIC_API_KEY) {
      const prompt = `You are an expert news translator for international news agencies. Translate the following article into natural, professional ${targetLocale === 'en' ? 'English' : targetLocale}:\n\nTitle: ${item.title}\n\nExcerpt: ${item.excerpt || ''}\n\nBody: ${item.body || ''}\n\nReturn ONLY a valid JSON object with keys "title", "excerpt", "body". Do not use markdown backticks.`;

      const aiRes = await fetch('https://api.anthropic.com/v1/messages', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': ANTHROPIC_API_KEY,
          'anthropic-version': '2023-06-01',
        },
        body: JSON.stringify({
          model: 'claude-3-5-haiku-20241022',
          max_tokens: 4000,
          messages: [{ role: 'user', content: prompt }],
        }),
      });

      if (aiRes.ok) {
        const aiData = await aiRes.json();
        const text = aiData?.content?.[0]?.text;
        if (text) {
          try {
            const parsed = JSON.parse(text.trim());
            translatedTitle = parsed.title || translatedTitle;
            translatedExcerpt = parsed.excerpt || translatedExcerpt;
            translatedBody = parsed.body || translatedBody;
          } catch (e) {
            console.warn('AI translation parse error, using source text fallback:', e);
          }
        }
      }
    }

    // 3. In modules/news, locales are stored as distinct news.Item records (Locale field).
    // Check if a translation of this item already exists in the target locale.
    const searchRes = await fetch(
      `${HYDROGO_API_URL}/api/v1/admin/news?locale=${targetLocale}&search=${encodeURIComponent(item.slug)}`,
      { headers: adminHeaders }
    );

    let existingTranslationId: string | null = null;
    let existingVersion = 1;

    if (searchRes.ok) {
      const { data: searchResults } = await searchRes.json();
      const existing = (searchResults || []).find(
        (r: any) => r.slug === item.slug && r.locale === targetLocale
      );
      if (existing) {
        existingTranslationId = existing.id;
        existingVersion = existing.version || 1;
      }
    }

    const translationTag = `translation_of:${item.id}`;
    const combinedTags = item.tags
      ? item.tags.includes(translationTag)
        ? item.tags
        : `${item.tags}, ${translationTag}`
      : translationTag;

    const payload = {
      locale: targetLocale,
      slug: item.slug,
      title: translatedTitle,
      excerpt: translatedExcerpt,
      body: translatedBody,
      key_takeaways: item.key_takeaways || '',
      cover_media_id: item.cover_media_id || '',
      category: item.category || '',
      author_name: item.author_name || '',
      featured: item.featured || false,
      tags: combinedTags,
      seo_title: translatedTitle,
      seo_description: translatedExcerpt,
    };

    let resultRes: Response;

    if (existingTranslationId) {
      // Update existing translation with optimistic concurrency (version)
      resultRes = await fetch(`${HYDROGO_API_URL}/api/v1/admin/news/${existingTranslationId}`, {
        method: 'PUT',
        headers: adminHeaders,
        body: JSON.stringify({
          ...payload,
          version: existingVersion,
        }),
      });
    } else {
      // Create new news.Item in target locale via POST /api/v1/admin/news
      resultRes = await fetch(`${HYDROGO_API_URL}/api/v1/admin/news`, {
        method: 'POST',
        headers: adminHeaders,
        body: JSON.stringify(payload),
      });
    }

    if (!resultRes.ok) {
      const errText = await resultRes.text();
      return json({ error: 'Failed to save translated news item in modules/news', details: errText }, { status: resultRes.status });
    }

    const { data: savedItem } = await resultRes.json();

    return json({
      success: true,
      locale: targetLocale,
      action: existingTranslationId ? 'updated' : 'created',
      item: savedItem,
    });
  } catch (err: any) {
    console.error('Translation workflow failed:', err);
    return json({ error: err?.message || 'Translation failed' }, { status: 500 });
  }
};
