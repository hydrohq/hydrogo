import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { getNewsBySlug, getNewsList, getMediaUrl, type NewsArticle } from '#lib';

export const load: PageServerLoad = async ({ params, setHeaders }) => {
  // Rule 12 Zero-build: Cache article on edge for 60s, stale-while-revalidate 10 min
  setHeaders({
    'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=600',
  });

  const article = await getNewsBySlug(params.slug);
  if (!article) {
    throw error(404, 'Haber bulunamadı');
  }

  // Related news in same category
  const related = await getNewsList({
    category: article.category,
    limit: 4,
  });

  // Filter out current article
  const relatedFiltered = related.filter((r: NewsArticle) => r.slug !== article.slug).slice(0, 3);

  // Structured Data (JSON-LD) for Google News SEO
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'NewsArticle',
    'headline': article.title,
    'description': article.excerpt || article.title,
    'datePublished': article.created_at,
    'dateModified': article.updated_at || article.created_at,
    'author': {
      '@type': 'Person',
      'name': article.author_name || 'HydroPost Haber Merkezi',
    },
    'image': article.cover_media_id ? [getMediaUrl(article.cover_media_id)] : [],
  };

  return {
    article,
    related: relatedFiltered,
    jsonLd,
  };
};
