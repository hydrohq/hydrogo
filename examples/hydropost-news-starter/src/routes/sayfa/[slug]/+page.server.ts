import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { getPage } from '#lib';
import { getViewCount } from '#lib/server/views';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
  if (params.slug.includes('.')) {
    throw error(404, 'Not found');
  }

  try {
    setHeaders({
      'Cache-Control': 'public, s-maxage=120, stale-while-revalidate=600',
    });
  } catch {
    // ignore
  }

  const page = await getPage(params.slug, 'tr');
  if (!page) {
    throw error(404, 'Sayfa bulunamadı');
  }

  return {
    page,
    views: getViewCount('page_' + params.slug),
  };
};
