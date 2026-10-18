import type { PageServerLoad } from './$types';
import {
  getNewsList,
  getAds,
  getGalleries,
  getVideos,
  getActivePoll,
  getHomepageSections,
  type NewsArticle,
} from '#lib';

export const load: PageServerLoad = async ({ setHeaders }) => {
  // Rule 12 Zero-build: Cache on edge/CDN for 30 seconds, serve stale up to 5 minutes while revalidating
  setHeaders({
    'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=300',
  });

  const [
    allLatest,
    featuredList,
    galleries,
    videos,
    poll,
    homeGrid1Ads,
    homeGrid2Ads,
    sections,
  ] = await Promise.all([
    getNewsList({ limit: 30 }),
    getNewsList({ featured: true, limit: 12 }),
    getGalleries(4),
    getVideos(4),
    getActivePoll(),
    getAds('home_grid_1', 'anasayfa'),
    getAds('home_grid_2', 'anasayfa'),
    getHomepageSections(),
  ]);

  // Combine featured list with latest news to provide a full 12-item Turkish manset
  const seenIds = new Set(featuredList.map((f: NewsArticle) => f.id));
  const additional = allLatest.filter((n: NewsArticle) => !seenIds.has(n.id));
  const featured = [...featuredList, ...additional].slice(0, 12);

  // Each "kategori_vitrini" homepage section names a category via
  // category_filter — fetch that category's own article list so the
  // section can show its own manşet-style showcase.
  const categoryKeys = [...new Set(
    sections
      .map((s) => s.category_filter)
      .filter((c): c is string => Boolean(c))
  )];
  const categoryLists = await Promise.all(categoryKeys.map((c) => getNewsList({ category: c, limit: 5 })));
  const categoryShowcases: Record<string, NewsArticle[]> = {};
  categoryKeys.forEach((c, i) => {
    categoryShowcases[c] = categoryLists[i];
  });

  const breakingNews: NewsArticle[] = allLatest.slice(0, 5);

  // Flat pools for the simple uniform card grids and the "most read" list —
  // not category-specific, just the latest items in rotation.
  const latestGridA = allLatest.slice(0, 8);
  const latestGridB = allLatest.slice(8, 14);
  const popularNews = allLatest.slice(0, 10);

  return {
    breakingNews,
    featured,
    latestGridA: latestGridA.length > 0 ? latestGridA : allLatest.slice(0, 6),
    latestGridB: latestGridB.length > 0 ? latestGridB : allLatest.slice(0, 6),
    popularNews: popularNews.length > 0 ? popularNews : allLatest.slice(0, 4),
    galleries,
    videos,
    poll,
    homeGrid1Ad: homeGrid1Ads[0] || null,
    homeGrid2Ad: homeGrid2Ads[0] || null,
    sections,
    categoryShowcases,
  };
};
