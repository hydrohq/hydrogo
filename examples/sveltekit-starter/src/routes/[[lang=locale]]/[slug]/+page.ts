import { hydro, localeOf, toHttpError, siteOrigin } from '$lib/hydro';

export async function load({ params }) {
  const locale = localeOf(params.lang);
  try {
    const page = await hydro.getPage(locale, params.slug);
    return { page, siteOrigin };
  } catch (err) {
    toHttpError(err);
  }
}
