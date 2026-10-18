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
    getAds('home_grid_1'),
    getAds('home_grid_2'),
    getHomepageSections(),
  ]);

  // Combine featured list with latest news to provide a full 12-item Turkish manset
  const seenIds = new Set(featuredList.map((f: NewsArticle) => f.id));
  const additional = allLatest.filter((n: NewsArticle) => !seenIds.has(n.id));
  const featured = [...featuredList, ...additional].slice(0, 12);

  const breakingNews: NewsArticle[] = allLatest.slice(0, 5);

  // Flat pools for the simple uniform card grids and the "most read" list —
  // not category-specific, just the latest items in rotation.
  const latestGridA = allLatest.slice(0, 6);
  const latestGridB = allLatest.slice(6, 12);
  const popularNews = allLatest.slice(0, 4);

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
  };
};
