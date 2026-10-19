import { hydro, locale, toHttpError } from '$lib/server/hydro';

export async function load() {
  try {
    return { news: await hydro.listNews(locale, 50) };
  } catch (err) {
    toHttpError(err);
  }
}
