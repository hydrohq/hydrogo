import type { PageServerLoad } from './$types';
import {
  getNewsList,
  getColumnists,
  getAds,
  getGalleries,
  getVideos,
  getActivePoll,
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
    columnists,
    galleries,
    videos,
    headerAds,
    mansetAds,
    poll,
  ] = await Promise.all([
    getNewsList({ limit: 20 }),
    getNewsList({ featured: true, limit: 10 }),
    getColumnists(),
    getGalleries(4),
    getVideos(4),
    getAds('header_top'),
    getAds('manset_alti'),
    getActivePoll(),
  ]);

  // If there are no specifically featured items, fallback to latest news
  const featured = featuredList.length > 0 ? featuredList : allLatest.slice(0, 8);
  const surManset = allLatest.slice(0, 4);

  // Group by categories
  const gundemNews = allLatest.filter((n: NewsArticle) => n.category === 'gundem').slice(0, 4);
  const dunyaNews = allLatest.filter((n: NewsArticle) => n.category === 'dunya').slice(0, 4);
  const ekonomiNews = allLatest.filter((n: NewsArticle) => n.category === 'ekonomi').slice(0, 4);

  // Fallback demo items if database is freshly seeded and has few articles
  const breakingNews: NewsArticle[] = allLatest.slice(0, 5);

  return {
    breakingNews,
    featured,
    surManset,
    gundemNews: gundemNews.length > 0 ? gundemNews : allLatest.slice(0, 4),
    dunyaNews: dunyaNews.length > 0 ? dunyaNews : allLatest.slice(4, 8),
    ekonomiNews: ekonomiNews.length > 0 ? ekonomiNews : allLatest.slice(8, 12),
    columnists,
    galleries,
    videos,
    poll,
    headerAd: headerAds[0] || null,
    mansetAltiAd: mansetAds[0] || null,
  };
};
