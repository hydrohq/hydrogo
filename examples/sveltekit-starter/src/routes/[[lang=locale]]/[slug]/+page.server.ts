import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/server/hydro';

export async function load({ params, setHeaders }) {
  const locale = localeOf(params.lang);
  try {
    const page = await hydro.getPage(locale, params.slug);
    setHeaders({ 'cache-control': 'public, max-age=0, s-maxage=30, stale-while-revalidate=60' });
    return { page, siteOrigin };
  } catch (err) {
    toHttpError(err);
  }
}
