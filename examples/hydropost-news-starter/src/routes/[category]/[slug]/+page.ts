import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { getNewsBySlug, getNewsList, getMediaUrl, getAds, getComments, getSiteSettings, getViewCount, type NewsArticle } from '#lib';

export const load: PageLoad = async ({ params }) => {
  if (!params.slug) {
    error(404, { message: 'Haber bulunamadı' });
  }

  try {
    const article = await getNewsBySlug(params.slug);
    if (!article) {
      error(404, { message: 'Haber bulunamadı' });
    }

    // Related news in same category or latest
    const allNews = await getNewsList({ limit: 20 });
    const relatedFiltered = allNews
      .filter((r: NewsArticle) => r.slug !== article.slug)
      .slice(0, 3);

    const popularNews = allNews
      .filter((r: NewsArticle) => r.slug !== article.slug)
      .slice(0, 6);

    // Structured Data (JSON-LD) for Google News SEO
    const jsonLd = {
      '@context': 'https://schema.org',
      '@type': 'NewsArticle',
      headline: article.title,
      description: article.excerpt || article.title,
      datePublished: article.created_at,
      dateModified: article.updated_at || article.created_at,
      author: {
        '@type': 'Person',
        name: article.author_name || 'HydroPost Haber Merkezi',
      },
      image: article.cover_media_id ? [getMediaUrl(article.cover_media_id, article.category)] : [getMediaUrl('', article.category)],
    };

    const [inArticleAds, footerAds, sidebarAds, comments, siteSettings] = await Promise.all([
      getAds('haber_ici', 'haber_detay'),
      getAds('footer', 'haber_detay'),
      getAds('sidebar', 'haber_detay'),
      getComments(article.slug),
      getSiteSettings(),
    ]);

    return {
      article,
      views: getViewCount(article.slug),
      related: relatedFiltered,
      popularNews,
      jsonLd,
      inArticleAd: inArticleAds[0] || null,
      footerAd: footerAds[0] || null,
      sidebarAd: sidebarAds[0] || null,
      comments,
      commentsEnabled: Boolean(siteSettings.comments_enabled),
      commentsRequireEmail: Boolean(siteSettings.comments_require_email),
    };
  } catch (err: any) {
    if (err?.status) throw err;
    console.error('Error loading article:', err);
    error(404, { message: 'Haber bulunamadı' });
  }
};
