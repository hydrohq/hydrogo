import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/hydro';

export async function load({ params }) {
  const locale = localeOf(params.lang);
  try {
    const item = await hydro.getNews(locale, params.slug);
    return { item, cover: item.cover_media_id ? hydro.mediaUrl(item.cover_media_id) : '', siteOrigin };
  } catch (err) {
    toHttpError(err);
  }
}
