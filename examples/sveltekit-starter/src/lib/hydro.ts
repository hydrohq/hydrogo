import { error } from '@sveltejs/kit';
import { createHydroGoClient, HydroGoError } from './hydrogo';
import { defaultLocale } from './config';

// Runs in the browser (this starter is a static SPA, no server-side
// rendering -- see docs/decisions on the Vite 8/Rolldown $env/dynamic
// regression for why). In production the static build and the HydroGo API
// are served from the same domain (hydro.cgi routed by .htaccess), so a
// relative request is all that's ever needed and nothing has to be
// configured per deployment. The local dev server runs on its own Vite
// port, so it needs the real HydroGo dev address explicitly.
const API_BASE =
  typeof window !== 'undefined'
    ? window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
      ? 'http://127.0.0.1:8080'
      : ''
    : 'http://127.0.0.1:8080';

export const hydro = createHydroGoClient({ origin: API_BASE });

// A build-time constant (see .env.example): canonical links, sitemap and RSS
// need an absolute URL and are generated once at build time (prerendered),
// when there is no window to read a runtime origin from.
export const siteOrigin = ((import.meta.env.VITE_PUBLIC_SITE_ORIGIN as string | undefined) ?? 'http://localhost:3000').replace(/\/$/, '');

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
