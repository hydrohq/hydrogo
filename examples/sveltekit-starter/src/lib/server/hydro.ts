import { env } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';
import { error } from '@sveltejs/kit';
import { createHydroGoClient, HydroGoError } from '$lib/hydrogo';
import { defaultLocale } from '$lib/config';

export const hydro = createHydroGoClient({ origin: env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080' });
export const siteOrigin = (publicEnv.PUBLIC_SITE_ORIGIN ?? 'http://localhost:3000').replace(/\/$/, '');

/** The API language for a request: the URL prefix, or the default language. */
export const localeOf = (lang: string | undefined) => lang ?? defaultLocale;

/** Turn an API failure into the right HTTP status for visitors and crawlers. */
export function toHttpError(err: unknown): never {
  if (err instanceof HydroGoError) {
    if (err.status === 404) error(404, 'Not found');
    error(502, `Content API error${err.requestId ? ` (request ${err.requestId})` : ''}`);
  }
  error(502, 'Content API unreachable');
}

type Record = { [field: string]: unknown };

/** The first record of a single-record content type such as site_settings; {} if it does not exist. */
export async function singleRecord(type: string): Promise<Record> {
  try {
    const list = await hydro.listContent<Record>(type, { limit: 1 });
    return list.data[0] ?? {};
  } catch {
    return {};
  }
}

export type NavItem = { label: string; href: string };

/** Header links from the site_navigation content type (placement "header"), in sort order. */
export async function headerNavigation(): Promise<NavItem[]> {
  try {
    const list = await hydro.listContent<{ label: string; href: string; placement: string; sort_order?: number }>('site_navigation', {
      limit: 50,
      filter: { placement: 'header' },
      sort: 'sort_order'
    });
    return list.data.map(({ label, href }) => ({ label, href }));
  } catch {
    return [];
  }
}
