import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/server/hydro';

const PAGE_SIZE = 12;

export async function load({ params, url, setHeaders }) {
  const locale = localeOf(params.lang);
  const pageNumber = Math.max(1, Number(url.searchParams.get('page')) || 1);
  try {
    const result = await hydro.listNewsPage(locale, PAGE_SIZE, (pageNumber - 1) * PAGE_SIZE);
    setHeaders({ 'cache-control': 'public, max-age=0, s-maxage=30, stale-while-revalidate=60' });
    return {
      news: result.data.map((item) => ({ item, cover: item.cover_media_id ? hydro.mediaUrl(item.cover_media_id) : '' })),
      pageNumber,
      hasMore: result.metadata.has_more,
      siteOrigin
    };
  } catch (err) {
    toHttpError(err);
  }
}
