import { env } from '$env/dynamic/public';

// /tr/... is a language prefix only when "tr" is one of PUBLIC_LOCALES;
// otherwise the segment is an ordinary page slug.
export function match(param: string): boolean {
  const configured = (env.PUBLIC_LOCALES ?? 'en').split(',').map((v) => v.trim());
  return configured.slice(1).includes(param);
}
