import { env } from '$env/dynamic/private';
import { createHydroGoClient, HydroGoError } from '@hydrogo/client';
import { error } from '@sveltejs/kit';

export const hydro = createHydroGoClient({ origin: env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080' });
export const locale = env.HYDRO_LOCALE ?? 'en';
export const siteOrigin = (env.PUBLIC_SITE_ORIGIN ?? 'http://localhost:3000').replace(/\/$/, '');

/** Turn an API failure into a proper HTTP status for the visitor and crawlers. */
export function toHttpError(err: unknown): never {
  if (err instanceof HydroGoError) {
    if (err.status === 404) error(404, 'Not found');
    error(502, `Content API error${err.requestId ? ` (request ${err.requestId})` : ''}`);
  }
  error(502, 'Content API unreachable');
}
