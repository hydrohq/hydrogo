import { error, isHttpError, isRedirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { createHydroGoClient, HydroGoError } from '$lib/hydrogo';

export const hydro = createHydroGoClient({ origin: env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080' });

/** Turn a catalog read failure into the right HTTP status for visitors and
 * crawlers. A load function's own `error(404, ...)` (a product that
 * doesn't exist) and `redirect(...)` must pass through unchanged when
 * thrown from inside the same try block this wraps -- they are not catalog
 * API failures, and re-wrapping them as a 502 would turn an intentional
 * 404 into a wrong one. */
export function toHttpError(err: unknown): never {
  if (isHttpError(err) || isRedirect(err)) throw err;
  if (err instanceof HydroGoError) {
    if (err.status === 404) error(404, 'Not found');
    error(502, `Content API error${err.requestId ? ` (request ${err.requestId})` : ''}`);
  }
  error(502, 'Content API unreachable');
}
