import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const API_BASE = (typeof process !== 'undefined' && process.env?.HYDROGO_API_URL) || 'http://127.0.0.1:8090';
const API_KEY = (typeof process !== 'undefined' && (process.env?.HYDROGO_API_KEY || process.env?.HYDROGO_ADMIN_KEY)) || '';

// Comments are moderated: a visitor's submission is created as a draft
// (HydroGo's content-type drafts workflow) and only shows up publicly once
// an editor publishes it from the admin panel.
export const GET: RequestHandler = async ({ url }) => {
  const slug = url.searchParams.get('slug');
  if (!slug) return json({ data: [] });

  const res = await fetch(`${API_BASE}/api/v1/content/comments?limit=50&filter[article_slug]=${encodeURIComponent(slug)}&sort=-created_at`, {
    headers: { Accept: 'application/json' },
  });
  if (!res.ok) return json({ data: [] });
  const body = await res.json();
  return json({ data: body.data || [] });
};

export const POST: RequestHandler = async ({ request }) => {
  let payload: { slug?: string; author_name?: string; author_email?: string; body?: string };
  try {
    payload = await request.json();
  } catch {
    return json({ code: 'bad_request', message: 'Geçersiz JSON verisi.' }, { status: 400 });
  }

  const { slug, author_name, author_email, body } = payload;
  if (!slug || !author_name?.trim() || !body?.trim()) {
    return json({ code: 'bad_request', message: 'Ad ve yorum metni zorunludur.' }, { status: 400 });
  }
  if (body.length > 2000) {
    return json({ code: 'bad_request', message: 'Yorum çok uzun (en fazla 2000 karakter).' }, { status: 400 });
  }

  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'application/json' };
  if (API_KEY) headers['X-API-Key'] = API_KEY;

  const res = await fetch(`${API_BASE}/api/v1/admin/content/comments`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      locale: 'tr',
      article_slug: slug,
      author_name: author_name.trim().slice(0, 80),
      author_email: (author_email || '').trim().slice(0, 160),
      body: body.trim(),
    }),
  });

  if (!res.ok) {
    return json({ code: 'submit_failed', message: 'Yorum gönderilemedi, lütfen tekrar deneyin.' }, { status: 502 });
  }

  return json({ success: true, message: 'Yorumunuz alındı, moderasyon sonrası yayınlanacak.' });
};
