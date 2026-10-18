import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getViewCount, incrementViewCount, getViewCounts } from '#lib/server/views';

export const GET: RequestHandler = async ({ url }) => {
  const slug = url.searchParams.get('slug');
  const slugsParam = url.searchParams.get('slugs');

  if (slugsParam) {
    const list = slugsParam.split(',').map((s) => s.trim()).filter(Boolean);
    const data = getViewCounts(list);
    return json({ data });
  }

  if (slug) {
    const views = getViewCount(slug);
    return json({ slug, views });
  }

  return json({ code: 'bad_request', message: 'slug parametresi gereklidir' }, { status: 400 });
};

export const POST: RequestHandler = async ({ request, cookies }) => {
  let body: { slug?: string; type?: string };
  try {
    body = await request.json();
  } catch {
    return json({ code: 'bad_request', message: 'Geçersiz JSON formatı' }, { status: 400 });
  }

  const { slug } = body;
  if (!slug || typeof slug !== 'string') {
    return json({ code: 'bad_request', message: 'slug alanı zorunludur' }, { status: 400 });
  }

  // Device deduplication via 365-day secure cookie
  const cookieName = `hp_v_${slug.slice(0, 32)}`;
  const alreadyViewed = cookies.get(cookieName);

  if (alreadyViewed) {
    const current = getViewCount(slug);
    return json({
      code: 'already_viewed',
      slug,
      views: current,
      incremented: false,
    });
  }

  // Set deduplication cookie for 1 year
  cookies.set(cookieName, '1', {
    path: '/',
    maxAge: 365 * 24 * 3600,
    httpOnly: true,
    sameSite: 'lax',
  });

  const next = incrementViewCount(slug);
  return json({
    code: 'ok',
    slug,
    views: next,
    incremented: true,
  });
};
