import type { PageServerLoad } from './$types';
import { getNewsList } from '#lib';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
  // Edge cache 60s
  setHeaders({
    'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=600',
  });

  const category = params.category.toLowerCase();
  const articles = await getNewsList({
    category,
    limit: 24,
  });

  return {
    category,
    articles,
  };
};
