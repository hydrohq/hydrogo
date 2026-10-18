import { error, redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { getNewsList, getAds, getPage, slugify, type NewsArticle } from '#lib';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
  if (params.category.includes('.')) {
    throw error(404, 'Not found');
  }

  // Edge cache 60s
  try {
    setHeaders({
      'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=600',
    });
  } catch {
    // ignore if header was already set
  }

  const categoryParam = params.category.toLowerCase();
  const allArticles = await getNewsList({ limit: 100 });
  const inCategory = allArticles.filter((n: NewsArticle) => {
    return slugify(n.category) === categoryParam || (n.category && n.category.toLowerCase() === categoryParam);
  });

  if (inCategory.length === 0) {
    try {
      const page = await getPage(categoryParam);
      if (page) {
        throw redirect(307, `/sayfa/${categoryParam}`);
      }
    } catch (e: any) {
      if (e?.status === 307 || e?.location) throw e;
    }
  }

  // The top highlight card prefers an editor-marked "Featured" article within
  // this category (the same flag that drives the homepage manşet) over just
  // whatever published most recently, so editors can pin it from the panel.
  const pinned = inCategory.find((n: NewsArticle) => n.featured);
  const articles = pinned ? [pinned, ...inCategory.filter((n) => n.id !== pinned.id)] : inCategory;

  const [headerTopAds, footerAds] = await Promise.all([
    getAds('header_top', 'kategori'),
    getAds('footer', 'kategori'),
  ]);

  return {
    category: params.category,
    articles,
    headerTopAd: headerTopAds[0] || null,
    footerAd: footerAds[0] || null,
  };
};
