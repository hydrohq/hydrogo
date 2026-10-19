import { error, redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { getNewsList, getAds, getPage, getSiteNavigation, slugify, type NewsArticle } from '#lib';

export const load: PageLoad = async ({ params }) => {
  if (params.category.includes('.')) {
    throw error(404, 'Not found');
  }

  const categoryParam = params.category.toLowerCase();

  const headerNav = await getSiteNavigation('header');
  const matchedNav = headerNav.find((item) => {
    const cleanHref = item.href.replace(/^\//, '').toLowerCase();
    return cleanHref === categoryParam || slugify(item.label) === categoryParam;
  });

  const queryCat = matchedNav ? matchedNav.label : params.category;
  let inCategory = await getNewsList({ category: queryCat, limit: 50 });
  if (inCategory.length === 0 && queryCat.toLowerCase() !== params.category.toLowerCase()) {
    inCategory = await getNewsList({ category: params.category, limit: 50 });
  }

  if (inCategory.length === 0) {
    try {
      const page = await getPage(categoryParam);
      if (page) {
        throw redirect(307, `/sayfa/${categoryParam}`);
      }
    } catch (e: any) {
      if (e?.status === 307 || e?.location) throw e;
    }

    if (!matchedNav) {
      throw error(404, 'Sayfa veya kategori bulunamadı');
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
