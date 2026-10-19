import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/server/hydro';

export async function load({ params, setHeaders }) {
  const locale = localeOf(params.lang);
  try {
    const news = await hydro.listNewsPage(locale, 6);
    // Visitors and CDNs may reuse the page for 30 s: publishing still feels instant,
    // and a traffic spike does not hit your HydroGo server for every request.
    setHeaders({ 'cache-control': 'public, max-age=0, s-maxage=30, stale-while-revalidate=60' });
    return {
      news: news.data.map((item) => ({ item, cover: item.cover_media_id ? hydro.mediaUrl(item.cover_media_id) : '' })),
      siteOrigin
    };
  } catch (err) {
    toHttpError(err);
  }
}
