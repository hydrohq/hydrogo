import { hydro, locale, toHttpError } from '$lib/server/hydro';

export async function load({ setHeaders }) {
  try {
    const news = await hydro.listNews(locale, 10);
    // Short shared-cache window: still near-instant publishing, far fewer API hits.
    setHeaders({ 'cache-control': 'public, max-age=0, s-maxage=30, stale-while-revalidate=60' });
    return { news };
  } catch (err) {
    toHttpError(err);
  }
}
