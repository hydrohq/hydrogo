import type { PageServerLoad } from './$types';
import { getNewsList, slugify, type NewsArticle } from '#lib';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
  // Edge cache 60s
  setHeaders({
    'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=600',
  });

  const categoryParam = params.category.toLowerCase();
  const allArticles = await getNewsList({ limit: 100 });
  const articles = allArticles.filter((n: NewsArticle) => {
    return slugify(n.category) === categoryParam || (n.category && n.category.toLowerCase() === categoryParam);
  });

  return {
    category: params.category,
    articles,
  };
};
