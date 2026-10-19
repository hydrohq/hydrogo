import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/server/hydro';

export async function load({ params, setHeaders }) {
  const locale = localeOf(params.lang);
  try {
    const item = await hydro.getNews(locale, params.slug);
    setHeaders({ 'cache-control': 'public, max-age=0, s-maxage=30, stale-while-revalidate=60' });
    return { item, cover: item.cover_media_id ? hydro.mediaUrl(item.cover_media_id) : '', siteOrigin };
  } catch (err) {
    toHttpError(err);
  }
}
