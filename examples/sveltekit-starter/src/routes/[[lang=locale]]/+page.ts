import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/hydro';

export async function load({ params }) {
  const locale = localeOf(params.lang);
  try {
    const news = await hydro.listNewsPage(locale, 6);
    return {
      news: news.data.map((item) => ({ item, cover: item.cover_media_id ? hydro.mediaUrl(item.cover_media_id) : '' })),
      siteOrigin
    };
  } catch (err) {
    toHttpError(err);
  }
}
