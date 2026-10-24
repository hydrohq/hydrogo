// Vite's own native env handling (import.meta.env.VITE_*), not SvelteKit's
// $env/static wrapper: this is a build-time choice (which locales does this
// deployment ship), inlined into the bundle either way, but Vite's own
// mechanism works uniformly in the browser and during the build-time
// prerender of robots.txt/rss.xml/sitemap.xml without depending on
// SvelteKit's ambient-type generation (see docs/decisions/0026 for the
// SvelteKit 3 + custom $lib alias interaction that broke $env/static here).
const locales_raw = import.meta.env.VITE_PUBLIC_LOCALES as string | undefined;

/** Site languages from VITE_PUBLIC_LOCALES; the first one is the default. */
export const locales = (locales_raw ?? 'en')
  .split(',')
  .map((value) => value.trim())
  .filter(Boolean);
export const defaultLocale = locales[0] ?? 'en';

/** "" for the default language, "/tr" for the others. */
export const localePrefix = (locale: string) => (locale === defaultLocale ? '' : `/${locale}`);

/** Swap the language prefix of a path: ("/tr/news", "en") -> "/news". */
export function switchLocalePath(pathname: string, target: string): string {
  const [, first = '', ...rest] = pathname.split('/');
  const bare = locales.includes(first) ? `/${rest.join('/')}` : pathname;
  const path = bare === '/' || bare === '' ? '' : bare;
  return `${localePrefix(target)}${path}` || '/';
}
