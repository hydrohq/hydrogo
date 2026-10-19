import { hydro, locale, siteOrigin, toHttpError } from '$lib/server/hydro';

export async function load({ params }) {
  try {
    const item = await hydro.getNews(locale, params.slug);
    return { item, canonical: `${siteOrigin}/news/${item.slug}` };
  } catch (err) {
    toHttpError(err);
  }
}
