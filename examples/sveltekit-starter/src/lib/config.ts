import { env } from '$env/dynamic/public';

/** Site languages from PUBLIC_LOCALES; the first one is the default. */
export const locales = (env.PUBLIC_LOCALES ?? 'en')
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
