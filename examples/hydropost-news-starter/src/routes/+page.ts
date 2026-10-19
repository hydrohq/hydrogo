import type { PageLoad } from './$types';
import {
  getNewsList,
  getAds,
  getGalleries,
  getVideos,
  getActivePoll,
  getHomepageSections,
  slugify,
  type NewsArticle,
} from '#lib';

export const load: PageLoad = async () => {
  const [
    allLatest,
    galleries,
    videos,
    poll,
    ads,
    sections,
  ] = await Promise.all([
    getNewsList({ limit: 60 }),
    getGalleries(4),
    getVideos(4),
    getActivePoll(),
    getAds(undefined, 'anasayfa'),
    getHomepageSections(),
  ]);

  // Featured list for Turkish manset
  const markedFeatured = allLatest.filter((n: NewsArticle) => n.featured);
  const seenIds = new Set(markedFeatured.map((f: NewsArticle) => f.id));
  const additional = allLatest.filter((n: NewsArticle) => !seenIds.has(n.id));
  const featured = [...markedFeatured, ...additional].slice(0, 14);

  // Populate category showcases from the pre-fetched pool to avoid 7+ separate CGI requests
  const categoryKeys = [...new Set(
    sections
      .map((s) => s.category_filter)
      .filter((c): c is string => Boolean(c))
  )];
  const categoryShowcases: Record<string, NewsArticle[]> = {};
  const missingCategories: string[] = [];

  for (const c of categoryKeys) {
    const targetSlug = slugify(c);
    const inPool = allLatest.filter((n) => slugify(n.category) === targetSlug).slice(0, 5);
    if (inPool.length >= 2) {
      categoryShowcases[c] = inPool;
    } else {
      missingCategories.push(c);
    }
  }

  // Only fetch missing categories individually if the pool did not have enough
  if (missingCategories.length > 0) {
    const extraLists = await Promise.all(missingCategories.map((c) => getNewsList({ category: c, limit: 5 })));
    missingCategories.forEach((c, i) => {
      categoryShowcases[c] = extraLists[i];
    });
  }

  const breakingNews: NewsArticle[] = allLatest.slice(0, 5);

  const latestGridA = allLatest.slice(0, 8);
  const latestGridB = allLatest.slice(8, 14);
  const popularNews = allLatest.slice(0, 10);

  const homeGrid1Ad = ads.find((a) => a.zone === 'home_grid_1') || null;
  const homeGrid2Ad = ads.find((a) => a.zone === 'home_grid_2') || null;

  return {
    breakingNews,
    featured,
    latestGridA: latestGridA.length > 0 ? latestGridA : allLatest.slice(0, 6),
    latestGridB: latestGridB.length > 0 ? latestGridB : allLatest.slice(0, 6),
    popularNews: popularNews.length > 0 ? popularNews : allLatest.slice(0, 4),
    galleries,
    videos,
    poll,
    homeGrid1Ad,
    homeGrid2Ad,
    sections,
    categoryShowcases,
  };
};
