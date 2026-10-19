import { hydro, locale, siteOrigin, toHttpError } from '$lib/server/hydro';

export async function load({ params }) {
  try {
    const page = await hydro.getPage(locale, params.slug);
    return { page, canonical: `${siteOrigin}/${page.slug}` };
  } catch (err) {
    toHttpError(err);
  }
}
