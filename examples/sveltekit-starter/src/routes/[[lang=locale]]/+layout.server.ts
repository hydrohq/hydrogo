import { headerNavigation, localeOf, singleRecord } from '$lib/server/hydro';
import { defaultLocale, locales } from '$lib/config';

export async function load({ params, locals }) {
  const locale = localeOf(params.lang);
  locals.lang = locale;
  const [settings, navigation] = await Promise.all([singleRecord('site_settings'), headerNavigation()]);
  return {
    locale,
    locales,
    defaultLocale,
    siteName: String(settings.site_name ?? 'My site'),
    tagline: String(settings.tagline ?? ''),
    footer: String(settings.copyright_text ?? ''),
    navigation
  };
}
