import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { getPage, getViewCount } from '#lib';

export const load: PageLoad = async ({ params }) => {
  if (params.slug.includes('.')) {
    throw error(404, 'Not found');
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
